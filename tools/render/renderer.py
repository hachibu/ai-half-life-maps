"""Software preview renderer for compiled BSPs: textures, baked lightmaps, skybox, additive brushes,
animated sprites and moving brush entities (rotating fans, sliding doors via `dyn`).

    r = Renderer("build/prairieloop/prairieloop.bsp", asset_root="build/prairieloop")
    img = r.frame(eye, target, t)          # numpy RGB image

asset_root holds sky/<skyname>XX.tga and sprites/*.spr exactly as the game's valve/ folder would.
It is an approximation of the game (no texture filtering), used for previews and videos.
"""
import math
import os
import struct

import numpy as np
from PIL import Image

from tools.bsp.reader import load


def load_spr(path):
    d_ = open(path, "rb").read()
    nf = struct.unpack_from("<4siiifiiifi", d_, 0)[7]
    o = 40
    ncol = struct.unpack_from("<h", d_, o)[0]; o += 2
    pal_ = np.frombuffer(d_, np.uint8, ncol * 3, o).reshape(ncol, 3).astype(float); o += ncol * 3
    frames_ = []
    for _ in range(nf):
        o += 4
        ox, oy, w_, h_ = struct.unpack_from("<iiii", d_, o); o += 16
        frames_.append((ox, oy, pal_[np.frombuffer(d_, np.uint8, w_ * h_, o).reshape(h_, w_)])); o += w_ * h_
    return frames_


class Renderer:
    def __init__(self, bsp, asset_root):
        m = self.m = load(bsp)
        self.g = {"models": m.models}
        self.faces, self.fverts, self.fext = m.faces, m.fverts, m.fext
        self.texinfo, self.tex, self.planes, self.light = m.texinfo, m.tex, m.planes, m.light
        world = m.entities[0]
        sky = world.get("skyname", "")
        self.SKY = {}
        for k in ("rt", "lf", "bk", "ft", "up", "dn"):
            p = os.path.join(asset_root, "gfx", "env", "%s%s.tga" % (sky, k))
            self.SKY[k] = np.asarray(Image.open(p).convert("RGB")) if os.path.exists(p) else np.zeros((4, 4, 3), np.uint8)
        self.SPR, self.SPRITES, self.ADD = {}, [], {}
        for e in m.entities:
            if e.get("classname") == "env_sprite" and e.get("model"):
                path = os.path.join(asset_root, e["model"])
                if not os.path.exists(path):
                    continue
                if path not in self.SPR:
                    self.SPR[path] = load_spr(path)
                self.SPRITES.append((np.array(list(map(float, e["origin"].split()))), float(e.get("scale", 1)), path,
                                     float(e.get("renderamt", 255)) / 255, float(e.get("framerate", 10))))
            if e.get("model", "").startswith("*") and e.get("rendermode") == "5":
                mm = m.models[int(e["model"][1:])]
                for fi_ in range(mm[14], mm[14] + mm[15]):
                    self.ADD[fi_] = float(e.get("renderamt", 255)) / 255.0
        F = len(self.faces)
        models_ = m.models
        self.fmodel = np.zeros(F, int)
        for mi, mdl in enumerate(models_):
            self.fmodel[mdl[14]:mdl[14] + mdl[15]] = mi
        self.minfo, self.mtrans = {}, {}
        for kv in m.entities:
            if kv.get("model", "").startswith("*"):
                mi = int(kv["model"][1:]); self.minfo[mi] = kv
                if "origin" in kv:
                    self.mtrans[mi] = (np.eye(3), np.array(list(map(float, kv["origin"].split()))))
        self.lverts = [v.astype(float) for v in self.fverts]
        self.cent0 = np.array([v.mean(0) for v in self.fverts], float)
        self.rad = np.array([np.linalg.norm(v - v.mean(0), axis=1).max() for v in self.fverts])
        N = np.zeros((F, 3)); D = np.zeros(F)
        for i, fc in enumerate(self.faces):
            pl = self.planes[fc[0]]; n = np.array(pl[:3]); d = pl[3]
            if fc[1]: n, d = -n, -d
            N[i], D[i] = n, d
        self.N, self.D = N, D
        self.lm = []
        for i, fc in enumerate(self.faces):
            ex = self.fext[i]
            if ex is None: self.lm.append(None); continue
            (smin, sext), (tmin, text_) = ex
            lw, lh = sext // 16 + 1, text_ // 16 + 1
            self.lm.append((smin, tmin, lw, lh, np.frombuffer(self.light, np.uint8, lw * lh * 3, fc[9]).reshape(lh, lw, 3).astype(float)))

    def frame(self, eye, target, t, W_=854, H_=480, fov=85, dyn=None):
        trans = dict(self.mtrans); trans.update(dyn or {})
        Nw, Dw, cent, wverts, ftr = self.N.copy(), self.D.copy(), self.cent0.copy(), {}, {}
        for mi, (Rm, tv) in trans.items():
            m = self.g["models"][mi]
            for fi_ in range(m[14], m[14] + m[15]):
                vw = self.lverts[fi_] @ Rm.T + tv
                wverts[fi_] = vw; ftr[fi_] = (Rm, tv)
                Nw[fi_] = Rm @ self.N[fi_]; Dw[fi_] = self.D[fi_] + Nw[fi_] @ tv; cent[fi_] = vw.mean(0)
        eye = np.array(eye, float); fwd = np.array(target, float) - eye; fwd /= np.linalg.norm(fwd)
        right = np.cross(fwd, [0, 0, 1]); right /= np.linalg.norm(right); up = np.cross(right, fwd)
        f = (W_ / 2) / math.tan(math.radians(fov / 2))
        zbuf = np.full((H_, W_), np.inf); img = np.zeros((H_, W_, 3))
        # bulk culling: backfaces, behind the camera, outside the view cone
        rel = cent - eye
        cz = rel @ fwd
        front = (Nw @ eye - Dw) > 0
        halfdiag = math.atan(math.hypot(W_ / 2, H_ / 2) / f)
        dist = np.linalg.norm(rel, axis=1) + 1e-6
        ang = np.arccos(np.clip(cz / dist, -1, 1)) - np.arcsin(np.clip(self.rad / dist, 0, 1))
        keep = np.where(front & (cz > -self.rad) & ((ang < halfdiag) | (dist < self.rad + 4)))[0]
        ys, xs = np.mgrid[0:H_, 0:W_]
        for fi in keep:
            fc = self.faces[fi]
            vs = wverts[fi] if fi in wverts else self.lverts[fi]
            n, d = Nw[fi], Dw[fi]
            # clip polygon to the near plane
            dd = (vs - eye) @ fwd - 1.0
            if (dd < 0).all(): continue
            if (dd < 0).any():
                out = []
                for i in range(len(vs)):
                    a, b = vs[i], vs[(i + 1) % len(vs)]; da, db = dd[i], dd[(i + 1) % len(vs)]
                    if da >= 0: out.append(a)
                    if (da >= 0) != (db >= 0): out.append(a + (b - a) * (da / (da - db)))
                cv = np.array(out)
            else:
                cv = vs
            r = cv - eye; czv = r @ fwd
            sx = W_ / 2 + f * (r @ right) / czv; sy = H_ / 2 - f * (r @ up) / czv
            x0, x1 = int(max(0, sx.min())), int(min(W_ - 1, sx.max() + 1))
            y0, y1 = int(max(0, sy.min())), int(min(H_ - 1, sy.max() + 1))
            if x0 > x1 or y0 > y1: continue
            X = xs[y0:y1 + 1, x0:x1 + 1]; Y = ys[y0:y1 + 1, x0:x1 + 1]
            Dd = fwd[None, None] * f + right[None, None] * (X - W_ / 2)[..., None] - up[None, None] * (Y - H_ / 2)[..., None]
            denom = Dd @ n
            with np.errstate(divide="ignore", invalid="ignore"):
                tt = (d - n @ eye) / denom
            P = eye + Dd * tt[..., None]
            inside = (tt > 0) & np.isfinite(tt)
            k = len(vs)
            for i in range(k):
                a, b = vs[i], vs[(i + 1) % k]
                inside &= (np.cross(b - a, P - a) @ n) <= 1e-3
            zsub = zbuf[y0:y1 + 1, x0:x1 + 1]
            draw = inside & (tt < zsub)
            if not draw.any(): continue
            ti = self.texinfo[fc[4]]; tname, tw, th, pix, pal = self.tex[ti[8]]
            if tname == "sky":
                col = self.sky(Dd)
                zsub[draw] = tt[draw]; img[y0:y1 + 1, x0:x1 + 1][draw] = col[draw]; continue
            Pl = ((P - ftr[fi][1]) @ ftr[fi][0]) if fi in ftr else P        # texture/lightmap in the model's own space
            s = Pl @ np.array(ti[0:3]) + ti[3]; tq = Pl @ np.array(ti[4:7]) + ti[7]
            si = s.astype(int) % tw; tj = tq.astype(int) % th
            col = pal[pix[tj, si]].astype(float)
            smin, tmin, lw, lh, lmap = self.lm[fi]
            li = np.clip(np.round((s - smin) / 16).astype(int), 0, lw - 1)
            lj = np.clip(np.round((tq - tmin) / 16).astype(int), 0, lh - 1)
            col = col * lmap[lj, li] / 128.0
            if fi in self.ADD:
                img[y0:y1 + 1, x0:x1 + 1][draw] += col[draw] * self.ADD[fi]; continue
            zsub[draw] = tt[draw]
            img[y0:y1 + 1, x0:x1 + 1][draw] = col[draw]
        # sprites (additive, depth tested at their centre depth)
        for o, sc, path, amt, fr_ in self.SPRITES:
            frames_ = self.SPR[path]
            ox, oy, pix = frames_[int(t * fr_) % len(frames_)]
            rel = o - eye; czs = rel @ fwd
            if czs < 16: continue
            sx = W_ / 2 + f * (rel @ right) / czs; sy = H_ / 2 - f * (rel @ up) / czs
            ph_, pw_ = pix.shape[:2]; k_ = sc * f / czs
            xa, xb = int(sx + ox * k_), int(sx + (ox + pw_) * k_); ya, yb = int(sy - oy * k_), int(sy + (ph_ - oy) * k_)
            if xb <= 0 or ya >= H_ or xa >= W_ or yb <= 0 or xb - xa < 1 or yb - ya < 1: continue
            cx0, cy0 = max(0, xa), max(0, ya); cx1, cy1 = min(W_, xb), min(H_, yb)
            # sample only the visible window of the scaled sprite
            uu = ((np.arange(cx0, cx1) - xa) / (xb - xa) * pw_).astype(int).clip(0, pw_ - 1)
            vv = ((np.arange(cy0, cy1) - ya) / (yb - ya) * ph_).astype(int).clip(0, ph_ - 1)
            sub = pix[vv][:, uu]
            vis = zbuf[cy0:cy1, cx0:cx1] * f > czs
            img[cy0:cy1, cx0:cx1][vis] += sub[vis] * amt
        return np.clip(img, 0, 255).astype(np.uint8)

    def sky(self, Dd):
        Dd = Dd / np.linalg.norm(Dd, axis=-1, keepdims=True)
        col = np.zeros(Dd.shape); ax = np.argmax(np.abs(Dd), -1)
        x_, y_, z_ = Dd[..., 0], Dd[..., 1], Dd[..., 2]
        with np.errstate(divide="ignore", invalid="ignore"):
            cases = [("rt", (ax == 0) & (x_ > 0), -y_ / x_, z_ / x_), ("lf", (ax == 0) & (x_ <= 0), y_ / -x_, z_ / -x_),
                     ("bk", (ax == 1) & (y_ > 0), x_ / y_, z_ / y_), ("ft", (ax == 1) & (y_ <= 0), -x_ / -y_, z_ / -y_),
                     ("up", (ax == 2) & (z_ > 0), -y_ / z_, -x_ / z_), ("dn", (ax == 2) & (z_ <= 0), -y_ / -z_, x_ / -z_)]
        for nm, m, ss, tt_ in cases:
            if not m.any(): continue
            im_ = self.SKY[nm]; N_ = im_.shape[0]
            col[m] = im_[np.clip(((1 - tt_[m]) / 2 * N_).astype(int), 0, N_ - 1), np.clip(((ss[m] + 1) / 2 * N_).astype(int), 0, N_ - 1)]
        return col

