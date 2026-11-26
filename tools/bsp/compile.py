#!/usr/bin/env python3
"""Minimal GoldSrc (BSP v30) compiler for simple sealed brush maps like cagematch.map.

Does: CSG face culling, hull-0 BSP with faces/leaves/marksurfaces, clip hulls 1-3 with
bevels, face subdivision (240 texels), direct lighting with shadows from light entities,
embedded textures from a WAD3. No VIS (engine treats every leaf as visible).
"""
import re, sys, struct, math
sys.setrecursionlimit(20000)
import numpy as np

EPS = 0.01
MAP, WAD, OUT = sys.argv[1], sys.argv[2], sys.argv[3]

# ------------------------------------------------------------------ parse .map
face_re = re.compile(r"\(\s*([-\d.e ]+)\)\s*\(\s*([-\d.e ]+)\)\s*\(\s*([-\d.e ]+)\)\s*(\S+)\s*"
                     r"\[\s*([-\d.e ]+)\]\s*\[\s*([-\d.e ]+)\]\s*([-\d.e]+)\s+([-\d.e]+)\s+([-\d.e]+)")

def parse_map(path):
    ents, depth, cur, brush = [], 0, None, None
    for line in open(path):
        line = line.strip()
        if line == "{":
            depth += 1
            if depth == 1: cur = {"kv": [], "brushes": []}
            else: brush = []
        elif line == "}":
            if depth == 2: cur["brushes"].append(brush)
            else: ents.append(cur)
            depth -= 1
        elif line.startswith('"'):
            k, v = re.findall(r'"([^"]*)"', line); cur["kv"].append((k, v))
        elif line.startswith("("):
            m = face_re.match(line)
            p = [np.array(list(map(float, m.group(i).split()))) for i in (1, 2, 3)]
            n = np.cross(p[0] - p[1], p[2] - p[1]); n /= np.linalg.norm(n)
            u = list(map(float, m.group(5).split())); v = list(map(float, m.group(6).split()))
            sx, sy = float(m.group(8)), float(m.group(9))
            tex = {"name": m.group(4),
                   "s": (u[0] / sx, u[1] / sx, u[2] / sx, u[3]),
                   "t": (v[0] / sy, v[1] / sy, v[2] / sy, v[3])}
            brush.append((n, float(n @ p[1]), tex))
    return ents

# ------------------------------------------------------------------ polygon helpers
def base_winding(n, d, size=1e5):
    up = np.array([0, 0, 1.0]) if abs(n[2]) < 0.9 else np.array([1.0, 0, 0])
    r = np.cross(up, n); r /= np.linalg.norm(r)
    u = np.cross(n, r)
    o = n * d
    return [o + (-r + u) * size, o + (r + u) * size, o + (r - u) * size, o + (-r - u) * size]

def split(poly, n, d):
    """Returns (front, back, status) where status is 'front','back','split','on'."""
    dists = [float(n @ p - d) for p in poly]
    sides = [1 if x > EPS else (-1 if x < -EPS else 0) for x in dists]
    if all(s == 0 for s in sides): return None, None, "on"
    if all(s >= 0 for s in sides): return poly, None, "front"
    if all(s <= 0 for s in sides): return None, poly, "back"
    f, b = [], []
    k = len(poly)
    for i in range(k):
        p, q = poly[i], poly[(i + 1) % k]
        sp, sq = sides[i], sides[(i + 1) % k]
        if sp >= 0: f.append(p)
        if sp <= 0: b.append(p)
        if sp * sq < 0:
            t = dists[i] / (dists[i] - dists[(i + 1) % k])
            m = p + (q - p) * t
            f.append(m); b.append(m)
    return clean(f), clean(b), "split"

def clean(poly):
    out = []
    for p in poly:
        if not out or np.linalg.norm(p - out[-1]) > EPS: out.append(p)
    if len(out) > 1 and np.linalg.norm(out[0] - out[-1]) < EPS: out.pop()
    return out if len(out) >= 3 and area(out) > 0.05 else None

def area(poly):
    s = np.zeros(3)
    for i in range(1, len(poly) - 1):
        s += np.cross(poly[i] - poly[0], poly[i + 1] - poly[0])
    return np.linalg.norm(s) / 2

def centroid(poly): return np.mean(poly, axis=0)

def brush_windings(planes):
    out = []
    for i, (n, d, tex) in enumerate(planes):
        w = base_winding(n, d)
        for j, (n2, d2, _) in enumerate(planes):
            if i == j or w is None: continue
            if np.allclose(n, n2) and abs(d - d2) < EPS: continue
            _, w, st = split(w, n2, d2)
            if st == "on": w = None
        if w is not None:
            out.append((w, n, d, tex))
    return out

def strictly_inside(planes, p, margin=EPS):
    return all(n @ p - d < -margin for n, d, _ in planes)

# ------------------------------------------------------------------ CSG
def visible_faces(brushes, solid_fn):
    """brushes: list of plane lists. Returns list of (poly, n, d, tex) that face empty space."""
    wins = [brush_windings(b) for b in brushes]
    boxes = []
    for w in wins:
        pts = np.array([p for poly, *_ in w for p in poly])
        boxes.append((pts.min(0) - 1, pts.max(0) + 1))
    out = []
    BMN = np.array([b[0] for b in boxes]); BMX = np.array([b[1] for b in boxes])
    for bi, w in enumerate(wins):
        for poly, n, d, tex in w:
            frags = [poly]
            pmn, pmx = np.min(poly, 0), np.max(poly, 0)
            near = np.where(np.all(BMN <= pmx, 1) & np.all(BMX >= pmn, 1))[0]
            for bj in near:
                if bj == bi: continue
                other = brushes[bj]
                new = []
                for f in frags:
                    inside = f
                    for n2, d2, _ in other:
                        fr, bk, st = split(inside, n2, d2)
                        if st == "on": bk = inside
                        if fr: new.append(fr)
                        inside = bk
                        if inside is None: break
                    if inside is not None:
                        if not strictly_inside(other, centroid(inside) + n * 0.1):
                            new.append(inside)
                frags = new
            for f in frags:
                if not solid_fn(centroid(f) + n * 0.1):
                    out.append((f, n, d, tex))
    return out

# ------------------------------------------------------------------ planes
planes, plane_idx = [], {}

def plane_index(n, d):
    """Returns (index, flipped). Axial planes always stored with positive normal."""
    a = np.abs(n)
    ax = int(np.argmax(a))
    flip = n[ax] < 0
    cn, cd = (-n, -d) if flip else (n, d)
    typ = ax if a[ax] > 1 - 1e-6 else 3 + ax
    if typ < 3:
        cn = np.zeros(3); cn[ax] = 1.0
    key = (tuple(np.round(cn, 5)), round(cd, 3))
    if key not in plane_idx:
        plane_idx[key] = len(planes)
        planes.append((cn, cd, typ))
    return plane_idx[key], flip

# ------------------------------------------------------------------ BSP build
def split_region(region, n, d):
    fr, bk, caps = [], [], []
    for poly in region:
        f, b, st = split(poly, n, d)
        if st == "on":
            continue
        if f: fr.append(f)
        if b: bk.append(b)
        if st == "split":
            for p in (f or []):
                if abs(n @ p - d) < EPS: caps.append(p)
    # include vertices lying exactly on the plane
    for poly in region:
        for p in poly:
            if abs(n @ p - d) < EPS: caps.append(p)
    if len(caps) >= 3:
        caps = np.unique(np.round(np.array(caps), 4), axis=0)
        if len(caps) >= 3:
            c = caps.mean(0)
            up = np.array([0, 0, 1.0]) if abs(n[2]) < 0.9 else np.array([1.0, 0, 0])
            r = np.cross(up, n); r /= np.linalg.norm(r); u = np.cross(n, r)
            ang = np.arctan2((caps - c) @ u, (caps - c) @ r)
            cap = [caps[i] for i in np.argsort(ang)]
            cap = clean(cap)
            if cap:
                fr.append(cap); bk.append(list(cap))
    return fr, bk

def box_region(mn, mx):
    x0, y0, z0 = mn; x1, y1, z1 = mx
    V = lambda x, y, z: np.array([x, y, z], float)
    return [
        [V(x0, y0, z0), V(x1, y0, z0), V(x1, y1, z0), V(x0, y1, z0)],
        [V(x0, y0, z1), V(x0, y1, z1), V(x1, y1, z1), V(x1, y0, z1)],
        [V(x0, y0, z0), V(x0, y1, z0), V(x0, y1, z1), V(x0, y0, z1)],
        [V(x1, y0, z0), V(x1, y0, z1), V(x1, y1, z1), V(x1, y1, z0)],
        [V(x0, y0, z0), V(x0, y0, z1), V(x1, y0, z1), V(x1, y0, z0)],
        [V(x0, y1, z0), V(x1, y1, z0), V(x1, y1, z1), V(x0, y1, z1)],
    ]

def clip_to_box(polys, mn, mx):
    out = []
    bplanes = [(np.array([1.0, 0, 0]), mx[0]), (np.array([-1.0, 0, 0]), -mn[0]),
               (np.array([0, 1.0, 0]), mx[1]), (np.array([0, -1.0, 0]), -mn[1]),
               (np.array([0, 0, 1.0]), mx[2]), (np.array([0, 0, -1.0]), -mn[2])]
    for poly, n, d, tex in polys:
        w = poly
        for bn, bd in bplanes:
            if w is None: break
            _, w, st = split(w, bn, bd)
            if st == "on": w = None
        if w is not None:
            out.append((w, n, d, tex))
    return out

def choose_splitter(polys):
    cands = {}
    for poly, n, d, tex in polys:
        k = (tuple(np.round(n, 4)), round(d, 2))
        nk = (tuple(np.round(-n, 4)), round(-d, 2))
        if k in cands or nk in cands: continue
        cands[k] = (n, d)
    items = list(cands.values())
    if len(items) > 48:
        items = items[:: max(1, len(items) // 48)]
    V = np.array([p for poly, *_ in polys for p in poly])
    starts = np.cumsum([0] + [len(poly) for poly, *_ in polys[:-1]])
    best, bscore = None, None
    for n, d in items:
        ds = V @ n - d
        hi = np.maximum.reduceat(ds, starts); lo = np.minimum.reduceat(ds, starts)
        sp = int(np.sum((hi > EPS) & (lo < -EPS)))
        fr = int(np.sum((hi > EPS) & ~(lo < -EPS)))
        bk = int(np.sum((lo < -EPS) & ~(hi > EPS)))
        axial = np.max(np.abs(n)) > 1 - 1e-6
        score = sp * 8 + abs(fr - bk) - (6 if axial else 0)
        if bscore is None or score < bscore:
            best, bscore = (n, d), score
    return best

class Node:  # interior node or leaf
    pass

def build_tree(polys, region, solid_fn):
    node = Node()
    pts = np.array([p for poly in region for p in poly]) if region else np.zeros((1, 3))
    node.mins, node.maxs = pts.min(0), pts.max(0)
    if not polys:
        node.leaf = True
        node.solid = (not region) or solid_fn(pts.mean(0))
        node.faces = []
        return node
    node.leaf = False
    n, d = choose_splitter(polys)
    node.n, node.d = n, d
    on, fr, bk = [], [], []
    for item in polys:
        poly, pn, pd, tex = item
        f, b, st = split(poly, n, d)
        if st == "on": on.append(item)
        else:
            if f: fr.append((f, pn, pd, tex))
            if b: bk.append((b, pn, pd, tex))
    node.on = on
    rf, rb = split_region(region, n, d)
    node.front = build_tree(fr, rf, solid_fn)
    node.back = build_tree(bk, rb, solid_fn)
    return node

def point_in_tree(node, p):
    while not node.leaf:
        node = node.front if node.n @ p - node.d >= 0 else node.back
    return node

# ------------------------------------------------------------------ load data
ents = parse_map(MAP)
world_ent = ents[0]
world_brushes = world_ent["brushes"]
EMISSIVE = {}
for k, v in world_ent["kv"]:
    if k == "_emissive" and v:
        for item in v.split(","):
            nm, val = item.split(":"); EMISSIVE[nm.upper()] = float(val)

allpts = np.array([p for b in world_brushes for w in brush_windings(b) for p in w[0]])
WMIN, WMAX = allpts.min(0), allpts.max(0)

def world_solid(p):
    if np.any(p < WMIN + EPS) or np.any(p > WMAX - EPS): return True
    return any(strictly_inside(b, p) for b in world_brushes)

HULLS = [None, (np.array([-16, -16, -36.]), np.array([16, 16, 36.])),
         (np.array([-32, -32, -32.]), np.array([32, 32, 32.])),
         (np.array([-16, -16, -18.]), np.array([16, 16, 18.]))]

def expand_brush(b, mins, maxs):
    pts = np.array([p for w in brush_windings(b) for p in w[0]])
    bmn, bmx = pts.min(0), pts.max(0)
    pl = [(n, d, t) for n, d, t in b]
    for ax in range(3):
        for sgn in (1, -1):
            n = np.zeros(3); n[ax] = sgn
            if any(np.allclose(n, q[0]) for q in pl): continue
            pl.append((n, (bmx[ax] if sgn > 0 else -bmn[ax]), None))
    out = []
    for n, d, t in pl:
        off = sum(n[i] * (-mins[i] if n[i] > 0 else -maxs[i]) for i in range(3))
        out.append((n, d + off, t))
    return out

# ------------------------------------------------------------------ output arrays
dnodes, dleafs, dfaces, dclip, texinfos, texinfo_idx = [], [], [], [], [], {}
verts, vert_idx, edges, edge_idx, surfedges, marksurf = [], {}, [(0, 0)], {}, [], []
lightdata = bytearray()
dleafs.append(dict(contents=-2, mins=(0, 0, 0), maxs=(0, 0, 0), first=0, num=0))  # leaf 0 = solid

# textures
wad = open(WAD, "rb").read()
num, ofs = struct.unpack("<II", wad[4:12])
wadtex = {}
for i in range(num):
    pos, dsz, sz, typ, comp, pad, name = struct.unpack("<IIIbbH16s", wad[ofs + 32 * i: ofs + 32 * (i + 1)])
    wadtex[name.split(b"\0")[0].decode().upper()] = wad[pos:pos + dsz]
tex_names, tex_lookup = [], {}

def tex_index(name):
    name = name.upper()
    if name not in tex_lookup:
        tex_lookup[name] = len(tex_names); tex_names.append(name)
    return tex_lookup[name]

def tex_size(name):
    return struct.unpack("<II", wadtex[name.upper()][16:24])

def texinfo_index(tex):
    key = (tex["name"].upper(), tuple(np.round(tex["s"], 5)), tuple(np.round(tex["t"], 5)))
    if key not in texinfo_idx:
        texinfo_idx[key] = len(texinfos)
        texinfos.append((tex["s"], tex["t"], tex_index(tex["name"]), 1 if tex["name"].upper().startswith("SKY") else 0))
    return texinfo_idx[key]

def vert_index(p):
    k = tuple(np.round(p, 2))
    if k not in vert_idx:
        vert_idx[k] = len(verts); verts.append(np.array(k, float))
    return vert_idx[k]

def edge_ref(a, b):
    if (b, a) in edge_idx: return -edge_idx[(b, a)]
    if (a, b) not in edge_idx:
        edge_idx[(a, b)] = len(edges); edges.append((a, b))
    return edge_idx[(a, b)]

def subdivide(poly, tex, n, limit=240.0):
    out, stack = [], [poly]
    while stack:
        w = stack.pop()
        done = True
        for key in ("s", "t"):
            v = np.array(tex[key][:3]); off = tex[key][3]
            vals = [p @ v + off for p in w]
            lo, hi = min(vals), max(vals)
            if math.floor(lo / 16) * 16 + limit < hi - 0.01:
                cut = math.floor(lo / 16) * 16 + limit
                L = np.linalg.norm(v)
                cn, cd = v / L, (cut - off) / L
                f, b, st = split(w, cn, cd)
                if f and b:
                    stack.append(f); stack.append(b); done = False
                    break
        if done: out.append(w)
    return out

# ------------------------------------------------------------------ lighting
lights = []
for e in ents:
    kv = dict(e["kv"])
    if kv.get("classname") == "light":
        r, g, b, i = map(float, kv["_light"].split())
        lights.append((np.array(list(map(float, kv["origin"].split()))), np.array([r, g, b]) / 255.0 * i))
shadow_brushes = [(np.array([n for n, d, t in b]), np.array([d for n, d, t in b])) for b in world_brushes]
K_LIGHT, AMBIENT = 30000.0, 14.0

def light_points(P, normal):
    """P: (N,3) sample points. Returns (N,3) RGB 0..255."""
    acc = np.full((len(P), 3), AMBIENT)
    start = P + normal * 1.0
    for lp, col in lights:
        dvec = lp - start
        dist = np.linalg.norm(dvec, axis=1)
        cos = (dvec @ normal) / np.maximum(dist, 1e-6)
        mask = cos > 0
        if not mask.any(): continue
        lit = mask.copy()
        for N, D in shadow_brushes:
            # segment start + t*dvec, t in [0,1], vs convex brush N x <= D
            sn = start @ N.T - D          # (n, planes)
            dn = dvec @ N.T               # (n, planes)
            t0 = np.zeros(len(P)); t1 = np.ones(len(P))
            with np.errstate(divide="ignore", invalid="ignore"):
                t = -sn / dn
                enter = dn < 0; leave = dn > 0
                t0 = np.maximum(t0, np.where(enter, t, -np.inf).max(1))
                t1 = np.minimum(t1, np.where(leave, t, np.inf).min(1))
            outside_par = ((dn == 0) & (sn > 0)).any(1)
            hit = (t0 < t1 - 1e-4) & ~outside_par
            lit &= ~hit
        val = K_LIGHT * cos / np.maximum(dist, 48) ** 2
        acc += np.where(lit[:, None], val[:, None] * col[None, :], 0)
    return np.clip(acc, 0, 255)

PENDING = []
LIGHT_OFS = np.zeros(3)
stats = {"faces": 0, "extent_mismatch": 0}

def emit_face(poly, n, d, tex, light=True):
    """Writes a face (CW from the front). Returns face index."""
    # order clockwise viewed from front: our polys are arbitrary, so sort explicitly
    c = centroid(poly)
    up = np.array([0, 0, 1.0]) if abs(n[2]) < 0.9 else np.array([1.0, 0, 0])
    r = np.cross(up, n); r /= np.linalg.norm(r); u = np.cross(n, r)
    ang = [math.atan2((p - c) @ u, (p - c) @ r) for p in poly]
    poly = [poly[i] for i in np.argsort(ang)][::-1]  # CCW -> CW
    vids = [vert_index(p) for p in poly]
    dedup = [v for i, v in enumerate(vids) if v != vids[i - 1]]
    pidx, flip = plane_index(n, d)
    first = len(surfedges)
    for i in range(len(dedup)):
        surfedges.append(edge_ref(dedup[i], dedup[(i + 1) % len(dedup)]))
    if tex["name"].upper().startswith("SKY"):
        dfaces.append(dict(plane=pidx, side=1 if flip else 0, firstedge=first, numedges=len(dedup),
                           texinfo=texinfo_index(tex), lightofs=-1, sky=True))
        stats["faces"] += 1
        return len(dfaces) - 1
    vs = np.array([verts[v] for v in dedup])
    vs32 = vs.astype(np.float32)
    base_tex = tex
    for nudge in (0.0, 0.05, -0.05, 0.11, -0.11, 0.23, -0.23, 0.37):
        tex = dict(base_tex, s=base_tex["s"][:3] + (base_tex["s"][3] + nudge,),
                   t=base_tex["t"][:3] + (base_tex["t"][3] + nudge,))
        sv, tv = np.array(tex["s"][:3]), np.array(tex["t"][:3])
        so, to = tex["s"][3], tex["t"][3]
        ex, ok = [], True
        for vec, off in ((sv, so), (tv, to)):
            vals = vs @ vec + off
            vals32 = (vs32 @ vec.astype(np.float32)) + np.float32(off)
            mn, mx = math.floor(vals.min() / 16), math.ceil(vals.max() / 16)
            mn32, mx32 = math.floor(vals32.min() / 16), math.ceil(vals32.max() / 16)
            # stay clear of a 16-texel boundary so any float precision gives the same answer
            margin = min(abs(vals.min() / 16 - round(vals.min() / 16)), abs(vals.max() / 16 - round(vals.max() / 16)))
            if (mn, mx) != (mn32, mx32) or (0 < margin < 1e-3): ok = False
            ex.append((mn, mx))
        if ok: break
    else:
        stats["extent_mismatch"] += 1
    ti = texinfo_index(tex)
    (smn, smx), (tmn, tmx) = ex
    assert (smx - smn) * 16 <= 256 and (tmx - tmn) * 16 <= 256, "bad extents"
    w, h = smx - smn + 1, tmx - tmn + 1
    M = np.array([sv, tv, n])
    Minv = np.linalg.inv(M)
    ss, tt = np.meshgrid((np.arange(w) + smn) * 16.0, (np.arange(h) + tmn) * 16.0)
    rhs = np.stack([ss.ravel() - so, tt.ravel() - to, np.full(w * h, d)], 1)
    P = rhs @ Minv.T
    # pull samples that fall outside the polygon slightly inward towards the centroid
    P = P + (c - P) * 0.02
    lofs = len(lightdata)
    lightdata.extend(b"\0" * (w * h * 3))
    PENDING.append((lofs, P + LIGHT_OFS, n, tex["name"].upper()))
    dfaces.append(dict(plane=pidx, side=1 if flip else 0, firstedge=first, numedges=len(dedup),
                       texinfo=ti, lightofs=lofs))
    stats["faces"] += 1
    return len(dfaces) - 1

EMPTY_SHARED = None

def write_hull0(root, is_world):
    """Writes nodes/leaves/faces for a tree. Returns root node index."""
    global EMPTY_SHARED
    face_nodes = []

    def rec(node):
        if node.leaf:
            if node.solid: return -1
            if not is_world:
                return -(EMPTY_SHARED + 1)
            dleafs.append(dict(contents=-1, mins=node.mins, maxs=node.maxs, first=0, num=0, faces=[]))
            node.leaf_index = len(dleafs) - 1
            return -(node.leaf_index + 1)
        idx = len(dnodes)
        dnodes.append(None)
        pidx, flip = plane_index(node.n, node.d)
        cf, cb = rec(node.front), rec(node.back)
        children = (cb, cf) if flip else (cf, cb)
        dnodes[idx] = dict(plane=pidx, children=children, mins=node.mins, maxs=node.maxs, faces=[])
        face_nodes.append((idx, node))
        return idx

    root_idx = rec(root)
    # faces: emit per node, contiguous
    for idx, node in sorted(face_nodes, key=lambda x: x[0]):
        first = len(dfaces)
        for poly, n, d, tex in node.on:
            for piece in subdivide(poly, tex, n):
                fi = emit_face(piece, n, d, tex)
                if is_world:
                    side = node.front if n @ node.n > 0 else node.back
                    mark_leaves(side, piece, fi)
        dnodes[idx]["firstface"], dnodes[idx]["numfaces"] = first, len(dfaces) - first
    return root_idx

def mark_leaves(node, poly, fi):
    if node.leaf:
        if not node.solid:
            dleafs[node.leaf_index]["faces"].append(fi)
        return
    f, b, st = split(poly, node.n, node.d)
    if st == "on":
        mark_leaves(node.front, poly, fi); return
    if f: mark_leaves(node.front, f, fi)
    if b: mark_leaves(node.back, b, fi)

def write_clip(root):
    def rec(node):
        if node.leaf:
            return -2 if node.solid else -1
        idx = len(dclip); dclip.append(None)
        pidx, flip = plane_index(node.n, node.d)
        cf, cb = rec(node.front), rec(node.back)
        dclip[idx] = (pidx, (cb, cf) if flip else (cf, cb))
        return idx
    r = rec(root)
    if r < 0:  # degenerate single-leaf hull: wrap in a dummy node
        pidx, _ = plane_index(np.array([0, 0, 1.0]), 1e6)
        dclip.append((pidx, (r, r))); r = len(dclip) - 1
    return r

def compile_model(brushes, is_world, noclip=False):
    if is_world:
        mn, mx, solid = WMIN, WMAX, world_solid
    else:
        pts = np.array([p for b in brushes for w in brush_windings(b) for p in w[0]])
        mn, mx = pts.min(0), pts.max(0)
        solid = lambda p: any(strictly_inside(b, p) for b in brushes)
    has_faces = not all(t["name"].upper() == "AAATRIGGER" for b in brushes for _, _, t in b)
    vis = visible_faces(brushes, solid)
    bmn, bmx = (mn, mx) if is_world else (mn - 64, mx + 64)
    vis = clip_to_box(vis, bmn, bmx)
    tree = build_tree(vis, box_region(bmn, bmx), solid)
    if not has_faces:
        def strip(nd):
            if not nd.leaf:
                nd.on = []; strip(nd.front); strip(nd.back)
        strip(tree)
    first_face = len(dfaces)
    leaf_start = len(dleafs)
    head0 = write_hull0(tree, is_world)
    numfaces = len(dfaces) - first_face
    heads = [head0]
    if noclip:
        pidx, _ = plane_index(np.array([0, 0, 1.0]), 1e6)
        dclip.append((pidx, (-1, -1)))
        heads += [len(dclip) - 1] * 3
        visleafs = 0
        return dict(mins=mn, maxs=mx, heads=heads, visleafs=visleafs, firstface=first_face, numfaces=numfaces)
    for hmin, hmax in HULLS[1:]:
        eb = [expand_brush(b, hmin, hmax) for b in brushes]
        if is_world:
            hs = lambda p, eb=eb: (np.any(p < WMIN + EPS) or np.any(p > WMAX - EPS)
                                   or any(strictly_inside(b, p) for b in eb))
            hmn, hmx = WMIN, WMAX
        else:
            hs = lambda p, eb=eb: any(strictly_inside(b, p) for b in eb)
            hmn, hmx = mn - 128, mx + 128
        hv = clip_to_box(visible_faces(eb, hs), hmn, hmx)
        heads.append(write_clip(build_tree(hv, box_region(hmn, hmx), hs)))
    visleafs = len(dleafs) - leaf_start if is_world else 0
    return dict(mins=mn, maxs=mx, heads=heads, visleafs=visleafs, firstface=first_face, numfaces=numfaces)

models = []
import time as _t; _T0 = _t.time(); print("world...", flush=True)
models.append(compile_model(world_brushes, True))
dleafs.append(dict(contents=-1, mins=(0, 0, 0), maxs=(0, 0, 0), first=0, num=0, faces=[]))
EMPTY_SHARED = len(dleafs) - 1

ent_text = []
for e in ents:
    kv = [(k, v) for k, v in e["kv"] if k != "wad"]
    if e is world_ent:
        kv.append(("wad", ""))
    elif e["brushes"]:
        print(dict(kv)["classname"], "...", flush=True)
        brs = e["brushes"]
        org = [b for b in brs if all(t["name"].upper() == "ORIGIN" for _, _, t in b)]
        org_ids = {id(b) for b in org}
        if org:
            pts = np.array([p for b in org for w in brush_windings(b) for p in w[0]])
            oc = np.round((pts.min(0) + pts.max(0)) / 2, 3)
            brs = [[(n, d - n @ oc, dict(t, s=t["s"][:3] + (t["s"][3] + np.array(t["s"][:3]) @ oc,),
                                         t=t["t"][:3] + (t["t"][3] + np.array(t["t"][:3]) @ oc,)))
                    for n, d, t in b] for b in brs if id(b) not in org_ids]
            kv = [p for p in kv if p[0] != "origin"] + [("origin", "%g %g %g" % tuple(oc))]
            LIGHT_OFS[:] = oc
        models.append(compile_model(brs, False, noclip=dict(kv)["classname"] in ("func_illusionary", "func_rotating")))
        LIGHT_OFS[:] = 0
        kv.append(("model", "*%d" % (len(models) - 1)))
    ent_text.append("{\n" + "".join('"%s" "%s"\n' % kvp for kvp in kv) + "}\n")
entities = ("".join(ent_text)).encode() + b"\0"

# ---- lighting for every luxel at once
def finalize_lighting():
    import time
    t_start = time.time()
    P = np.concatenate([p for _, p, _, _ in PENDING])
    N = np.concatenate([np.repeat(n[None], len(p), 0) for _, p, n, _ in PENDING])
    sizes = np.array([len(p) for _, p, _, _ in PENDING])
    starts = np.concatenate([[0], np.cumsum(sizes)[:-1]])
    cent = np.concatenate([np.repeat(p.mean(0)[None], len(p), 0) for _, p, _, _ in PENDING])
    # padded plane arrays for every world brush: (B, Pmax) normals / dists; padding is always "inside"
    Pmax = max(len(b) for b in world_brushes)
    BN = np.zeros((len(world_brushes), Pmax, 3)); BD = np.full((len(world_brushes), Pmax), 1e9)
    bmin = np.zeros((len(world_brushes), 3)); bmax = np.zeros((len(world_brushes), 3))
    for i, b in enumerate(world_brushes):
        for j, (n, d, t) in enumerate(b):
            BN[i, j] = n; BD[i, j] = d
        pts = np.array([q for w in brush_windings(b) for q in w[0]])
        bmin[i], bmax[i] = pts.min(0), pts.max(0)
    F = len(PENDING)
    start = P + N * 1.0

    def face_bounds(pts):
        mn = np.minimum.reduceat(pts, starts, axis=0); mx = np.maximum.reduceat(pts, starts, axis=0)
        return mn, mx

    # 1) pull luxels that sit inside solid towards their face centre (face-level culling first)
    for it in range(6):
        fmn, fmx = face_bounds(start)
        moved = 0
        for fi in range(F):
            cand = np.where(np.all(bmin <= fmx[fi], 1) & np.all(bmax >= fmn[fi], 1))[0]
            if len(cand) == 0: continue
            s0, s1 = starts[fi], starts[fi] + sizes[fi]
            X = start[s0:s1]
            inside = np.any(np.all(np.einsum("ij,kpj->ikp", X, BN[cand]) - BD[cand][None] < -0.01, 2), 1)
            if inside.any():
                idx = np.where(inside)[0] + s0
                start[idx] = start[idx] + (cent[idx] + N[idx] - start[idx]) * 0.5
                moved += len(idx)
        if moved == 0: break
    print("  luxel nudge done %.1fs" % (time.time() - t_start), flush=True)

    acc = np.full((len(P), 3), AMBIENT)
    for li, (lp, col) in enumerate(lights):
        R = math.sqrt(K_LIGHT * float(col.max()) / 2.0)
        dvec = lp - start
        dist = np.linalg.norm(dvec, axis=1)
        cos = np.einsum("ij,ij->i", dvec, N) / np.maximum(dist, 1e-6)
        ok = (cos > 0) & (dist < R)
        if not ok.any(): continue
        lit = ok.copy()
        fmn, fmx = face_bounds(start)
        fany = np.add.reduceat(ok.astype(int), starts) > 0
        smn = np.minimum(fmn, lp); smx = np.maximum(fmx, lp)
        near = np.where(np.all(bmin <= lp + R, 1) & np.all(bmax >= lp - R, 1))[0]
        nmin, nmax = bmin[near], bmax[near]
        for fi in np.where(fany)[0]:
            cand = near[np.all(nmin <= smx[fi], 1) & np.all(nmax >= smn[fi], 1)]
            if len(cand) == 0: continue
            s0, s1 = starts[fi], starts[fi] + sizes[fi]
            m = ok[s0:s1]
            S0 = start[s0:s1][m]; Dv = dvec[s0:s1][m]
            sn = np.einsum("ij,kpj->ikp", S0, BN[cand]) - BD[cand][None]     # (n, k, P)
            dn = np.einsum("ij,kpj->ikp", Dv, BN[cand])
            with np.errstate(divide="ignore", invalid="ignore"):
                t = -sn / dn
                t0 = np.maximum(0, np.where(dn < 0, t, -np.inf).max(2))
                t1 = np.minimum(1, np.where(dn > 0, t, np.inf).min(2))
            par_out = ((dn == 0) & (sn > 0)).any(2)
            hit = ((t0 < t1 - 1e-4) & ~par_out).any(1)
            sub = lit[s0:s1]; idx = np.where(m)[0]
            sub[idx[hit]] = False; lit[s0:s1] = sub
        val = K_LIGHT * cos / np.maximum(dist, 64) ** 2
        acc += np.where(lit[:, None], val[:, None] * col[None, :], 0)
        if li % 10 == 0:
            print("  light %d/%d  %.1fs" % (li + 1, len(lights), time.time() - t_start), flush=True)
    for i, (lofs, p, n, name) in enumerate(PENDING):
        a = acc[starts[i]:starts[i] + sizes[i]]
        if name in EMISSIVE:
            a = np.maximum(a, EMISSIVE[name])
        bts = np.clip(a, 0, 255).astype(np.uint8).tobytes()
        lightdata[lofs:lofs + len(bts)] = bts
    print("luxels lit:", len(P), "in %.1fs" % (time.time() - t_start), flush=True)

print("geometry done %.1fs, %d faces, %d lights" % (_t.time() - _T0, len(dfaces), len(lights)), flush=True)
finalize_lighting()

# marksurfaces
for lf in dleafs:
    f = lf.get("faces", [])
    lf["first"], lf["num"] = len(marksurf), len(f)
    marksurf.extend(f)

# ------------------------------------------------------------------ pack lumps
def s16(v): return max(-32768, min(32767, int(v)))

L = [b""] * 15
L[0] = entities
L[1] = b"".join(struct.pack("<4fi", *n, d, t) for n, d, t in planes)
for nm in list(tex_names):
    if nm.startswith("+0"):
        for k in range(1, 10):
            fr = "+%d%s" % (k, nm[2:])
            if fr in wadtex and fr not in tex_lookup: tex_index(fr)
texblob = [wadtex[nm] for nm in tex_names]
hdr = 4 + 4 * len(texblob)
offs, pos = [], hdr
for t in texblob:
    offs.append(pos); pos += len(t)
L[2] = struct.pack("<i", len(texblob)) + b"".join(struct.pack("<i", o) for o in offs) + b"".join(texblob)
L[3] = b"".join(struct.pack("<3f", *v) for v in verts)
L[4] = b""
L[5] = b"".join(struct.pack("<ihh3h3hHH", n["plane"], *n["children"],
                            *[s16(math.floor(x)) for x in n["mins"]], *[s16(math.ceil(x)) for x in n["maxs"]],
                            n["firstface"], n["numfaces"]) for n in dnodes)
L[6] = b"".join(struct.pack("<8f2i", *s, *t, mi, fl) for s, t, mi, fl in texinfos)
L[7] = b"".join(struct.pack("<hhihh4Bi", f["plane"], f["side"], f["firstedge"], f["numedges"],
                            f["texinfo"], *((255, 255, 255, 255) if f.get("sky") else (0, 255, 255, 255)), f["lightofs"]) for f in dfaces)
L[8] = bytes(lightdata)
L[9] = b"".join(struct.pack("<i2h", p, *c) for p, c in dclip)
L[10] = b"".join(struct.pack("<ii3h3hHH4B", lf["contents"], -1,
                             *[s16(math.floor(x)) for x in lf["mins"]], *[s16(math.ceil(x)) for x in lf["maxs"]],
                             lf["first"], lf["num"], 0, 0, 0, 0) for lf in dleafs)
L[11] = b"".join(struct.pack("<H", m) for m in marksurf)
L[12] = b"".join(struct.pack("<2H", *e) for e in edges)
L[13] = b"".join(struct.pack("<i", s) for s in surfedges)
L[14] = b"".join(struct.pack("<9f4i3i", *m["mins"], *m["maxs"], 0, 0, 0, *m["heads"],
                             m["visleafs"], m["firstface"], m["numfaces"]) for m in models)

out = bytearray(struct.pack("<i", 30) + b"\0" * (15 * 8))
lumpinfo = []
for data in L:
    while len(out) % 4: out.append(0)
    lumpinfo.append((len(out), len(data)))
    out.extend(data)
struct.pack_into("<i30i", out, 0, 30, *[x for li in lumpinfo for x in li])
open(OUT, "wb").write(out)

print("planes", len(planes), "nodes", len(dnodes), "leafs", len(dleafs), "faces", len(dfaces),
      "clipnodes", len(dclip), "texinfo", len(texinfos), "verts", len(verts), "edges", len(edges),
      "light bytes", len(lightdata), "models", len(models))
print("extent float32 mismatches:", stats["extent_mismatch"])
print("bytes", len(out))
