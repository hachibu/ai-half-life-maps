"""Read a compiled GoldSrc BSP (v30) into plain Python/numpy structures.

Used by every check and by the preview renderer, so they all agree on how the engine sees the map.
"""
import math
import re
import struct
from types import SimpleNamespace

import numpy as np

# Engine limits the checks enforce. Light samples: the GoldSrc lightmap atlas holds ~1,048,576
# samples, and packing wastes some, so we keep a safety margin.
LIMITS = [
    ("light samples", "lux", 800000),
    ("clipnodes", "clip", 32767),
    ("nodes", "nodes", 32767),
    ("leafs", "leafs", 8192),
    ("faces", "faces", 65535),
    ("marksurfaces", "marks", 65535),
    ("planes", "planes", 32767),
    ("texinfo", "texinfo", 8192),
    ("models", "models", 400),
    ("vertexes", "verts", 65535),
]


def load(path):
    data = open(path, "rb").read()
    assert struct.unpack_from("<i", data, 0)[0] == 30, "not a BSP v30 file"
    lumps = [struct.unpack_from("<ii", data, 4 + 8 * i) for i in range(15)]

    def lump(i):
        o, ln = lumps[i]
        return data[o:o + ln]

    def arr(i, fmt):
        b = lump(i)
        sz = struct.calcsize(fmt)
        assert len(b) % sz == 0, (i, len(b), sz)
        return [struct.unpack_from(fmt, b, k * sz) for k in range(len(b) // sz)]

    m = SimpleNamespace()
    m.planes = arr(1, "<4fi")
    m.verts = arr(3, "<3f")
    m.nodes = arr(5, "<ihh3h3hHH")
    m.texinfo = arr(6, "<8f2i")
    m.faces = arr(7, "<hhihh4Bi")
    m.clip = arr(9, "<i2h")
    m.leafs = arr(10, "<ii3h3hHH4B")
    m.edges = arr(12, "<2H")
    m.surf = [x[0] for x in arr(13, "<i")]
    m.marks = [x[0] for x in arr(11, "<H")]
    m.models = arr(14, "<9f4i3i")
    m.light = lump(8)
    m.ents = lump(0).decode(errors="replace")
    m.entities = [dict(re.findall(r'"([^"]*)" "([^"]*)"', b)) for b in re.findall(r"\{([^{}]*)\}", m.ents)]
    m.errors = []

    def chk(c, msg):
        if not c:
            m.errors.append(msg)

    for n in m.nodes:
        chk(0 <= n[0] < len(m.planes), "node plane")
        for c in n[1:3]:
            chk(c < len(m.nodes) if c >= 0 else (-1 - c) < len(m.leafs), "node child")
        chk(n[-2] + n[-1] <= len(m.faces), "node faces")
    for p, a, b in m.clip:
        chk(0 <= p < len(m.planes), "clip plane")
        for c in (a, b):
            chk(c < len(m.clip) if c >= 0 else c in (-1, -2), "clip child")
    for f in m.faces:
        chk(0 <= f[0] < len(m.planes) and 0 <= f[4] < len(m.texinfo), "face refs")
        chk(f[2] + f[3] <= len(m.surf) and f[3] >= 3, "face edges")
    for s in m.surf:
        chk(abs(s) < len(m.edges), "surfedge")
    for e in m.edges:
        chk(e[0] < len(m.verts) and e[1] < len(m.verts), "edge")
    for mk in m.marks:
        chk(mk < len(m.faces), "mark")
    for lf in m.leafs:
        chk(lf[8] + lf[9] <= len(m.marks), "leaf marks")
    for p in m.planes:
        if p[4] < 3:
            chk(abs(p[p[4]] - 1) < 1e-6, "axial plane must be positive")

    # embedded textures
    tb = lump(2)
    m.tex = []
    for i in range(struct.unpack_from("<i", tb, 0)[0]):
        o = struct.unpack_from("<i", tb, 4 + 4 * i)[0]
        name = tb[o:o + 16].split(b"\0")[0].decode()
        w, h, o0 = struct.unpack_from("<III", tb, o + 16)
        pix = np.frombuffer(tb, np.uint8, w * h, o + o0).reshape(h, w)
        palo = o + 40 + w * h * 85 // 64
        chk(struct.unpack_from("<H", tb, palo)[0] == 256, "texture palette")
        pal = np.frombuffer(tb, np.uint8, 768, palo + 2).reshape(256, 3)
        m.tex.append((name, w, h, pix, pal))
    names = {t[0] for t in m.tex}
    for nm in names:                                   # animated textures need every frame
        if nm.startswith("+0"):
            for k in range(1, 10):
                if "+%d%s" % (k, nm[2:]) not in names and any(n.startswith("+%d" % (k - 1)) for n in names):
                    pass
            chk("+1" + nm[2:] in names, "animated texture %s is missing frame +1" % nm)

    # face vertices + lightmap extents exactly as the engine computes them
    m.fverts, m.fext = [], []
    m.lux = 0
    for f in m.faces:
        vs = []
        for k in range(f[3]):
            s = m.surf[f[2] + k]
            vs.append(m.verts[m.edges[s][0]] if s >= 0 else m.verts[m.edges[-s][1]])
        vs = np.array(vs, np.float32)
        ti = m.texinfo[f[4]]
        if ti[9] & 1:                                  # TEX_SPECIAL (sky)
            chk(m.tex[ti[8]][0] == "sky", "sky texture must be named lowercase 'sky' (got %s)" % m.tex[ti[8]][0])
            m.fverts.append(vs)
            m.fext.append(None)
            continue
        ex = []
        for j in range(2):
            vec = np.array(ti[4 * j:4 * j + 3], np.float32)
            vals = vs @ vec + np.float32(ti[4 * j + 3])
            mn, mx = math.floor(vals.min() / 16), math.ceil(vals.max() / 16)
            ex.append((mn * 16, (mx - mn) * 16))
            chk((mx - mn) * 16 <= 256, "bad surface extents")
        w, h = ex[0][1] // 16 + 1, ex[1][1] // 16 + 1
        chk(w * h <= 324, "lightmap larger than 324 samples")
        chk(f[9] + w * h * 3 <= len(m.light), "lightmap out of range")
        m.lux += w * h
        m.fverts.append(vs)
        m.fext.append(ex)

    m.counts = {"lux": m.lux, "clip": len(m.clip), "nodes": len(m.nodes), "leafs": len(m.leafs),
                "faces": len(m.faces), "marks": len(m.marks), "planes": len(m.planes),
                "texinfo": len(m.texinfo), "models": len(m.models), "verts": len(m.verts)}
    for label, key, lim in LIMITS:
        chk(m.counts[key] < lim, "over engine limit: " + label)

    def hull0(head, p):
        n = head
        while n >= 0:
            nd = m.nodes[n]
            pl = m.planes[nd[0]]
            d = (p[pl[4]] if pl[4] < 3 else np.dot(pl[:3], p)) - pl[3]
            n = nd[1] if d >= 0 else nd[2]
        return m.leafs[-1 - n][0]

    def hullc(head, p):
        n = head
        while n >= 0:
            pn, a, b = m.clip[n]
            pl = m.planes[pn]
            d = (p[pl[4]] if pl[4] < 3 else np.dot(pl[:3], p)) - pl[3]
            n = a if d >= 0 else b
        return n

    m.hull0, m.hullc = hull0, hullc
    m.world_heads = m.models[0][9:13]
    m.solid = lambda p: hull0(m.world_heads[0], p) == -2          # point (hull 0)
    m.player_free = lambda p: hullc(m.world_heads[1], p) == -1    # standing player centre (hull 1)
    return m
