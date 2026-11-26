#!/usr/bin/env python3
"""Prairie Loop Winter: JustinCon After Hours  (Half-Life 1 deathmatch).
A snowed-in prairie loop street; nine basements joined by a ring tunnel; the blue house's
second staircase reveals JustinCon, a 32-PC neon LAN hall where BattleAxe faces Crescent."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))     # repo root, for tools.*
sys.path.insert(0, str(Path(__file__).resolve().parent))         # this map's own modules
from tools.mapkit.paths import build_dir
from tools.mapkit import chunks, detail_entity, split_detail, texlights
import math, os, sys, json
import numpy as np
from scipy.spatial import ConvexHull
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import assets as A

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = str(build_dir("prairieloop"))
NAME = "prairieloop"
TEX = A.write_wad(os.path.join(OUT, NAME + ".wad"))
SCALE = {"SNOW": 4, "SNOWROAD": 3, "ROOFSNOW": 3, "TUNNELWALL": 2, "TUNNELFLOOR": 2, "HALLWALL": 2,
         "HALLCEIL": 2, "ARCADECARPET": 1, "ICE": 3, "CEILTILE": 2, "CEILING": 2, "WALLPAPER": 1}

# ------------------------------------------------------------------ brush helpers
Z = np.array([0, 0, 1.0])
def axes(n):
    if abs(n[2]) > 0.7: return np.array([1.0, 0, 0]), np.array([0, -1.0, 0])
    u = np.cross(Z, n); u /= np.linalg.norm(u); return u, np.cross(u, n)
def face_line(n, d, t):
    t = t if isinstance(t, dict) else {"name": t}
    n = np.array(n, float); n /= np.linalg.norm(n)
    e1 = np.cross(n, [0.0, 0.0, 1.0]) if abs(n[2]) < 0.9 else np.cross(n, [1.0, 0.0, 0.0])
    e1 /= np.linalg.norm(e1); e2 = np.cross(n, e1)
    p1 = n * d; p0, p2 = p1 + e1 * 256, p1 + e2 * 256
    u, v = axes(n); sc = SCALE.get(t["name"], 1)
    f = lambda p: " ".join("%.6f" % x for x in p)
    return "( %s ) ( %s ) ( %s ) %s [ %s %.4f ] [ %s %.4f ] 0 %g %g" % (
        f(p0), f(p1), f(p2), t["name"], f(u), t.get("ou", 0), f(v), t.get("ov", 0), t.get("su", sc), t.get("sv", sc))
def brush(planes): return "{\n" + "\n".join(face_line(n, d, t) for n, d, t in planes) + "\n}\n"
def box(x0, y0, z0, x1, y1, z1, tex, top=None, bottom=None, n=None, s=None, e=None, w=None):
    x0, x1 = min(x0, x1), max(x0, x1); y0, y1 = min(y0, y1), max(y0, y1); z0, z1 = min(z0, z1), max(z0, z1)
    assert x1 - x0 > 0.01 and y1 - y0 > 0.01 and z1 - z0 > 0.01, (x0, y0, z0, x1, y1, z1)
    return brush([((0, 0, 1), z1, top or tex), ((0, 0, -1), -z0, bottom or tex), ((0, 1, 0), y1, n or tex),
                  ((0, -1, 0), -y0, s or tex), ((1, 0, 0), x1, e or tex), ((-1, 0, 0), -x0, w or tex)])
def hull_brush(pts, tex, top=None):
    h = ConvexHull(np.array(pts, float)); seen, pl = set(), []
    for eq in h.equations:
        n, off = eq[:3], eq[3]; k = (tuple(np.round(n, 5)), round(-off, 3))
        if k in seen: continue
        seen.add(k); pl.append((n, -off, top if (top and n[2] > 0.3) else tex))
    return brush(pl)
def beam(p0, p1, th, tex):
    p0, p1 = np.array(p0, float), np.array(p1, float); a = p1 - p0; a /= np.linalg.norm(a)
    hlp = Z if abs(a[2]) < 0.9 else np.array([1.0, 0, 0])
    side = np.cross(a, hlp); side /= np.linalg.norm(side); up = np.cross(side, a)
    pl = [(a, a @ p1, tex), (-a, -(a @ p0), tex)]
    for v in (side, -side, up, -up): pl.append((v, v @ p0 + th / 2, tex))
    return brush(pl)
NRM = {"n": (0, 1, 0), "s": (0, -1, 0), "e": (1, 0, 0), "w": (-1, 0, 0)}
def fit_spec(name, facing, x0, y0, x1, y1, z0, z1):
    u, _ = axes(np.array(NRM[facing], float))
    us = [np.array([x, y, 0]) @ u for x in (x0, x1) for y in (y0, y1)]
    tw, th = TEX[name]; su = (max(us) - min(us)) / tw; sv = (z1 - z0) / th
    return {"name": name, "su": su, "sv": sv, "ou": -min(us) / su, "ov": z1 / sv}
def sign(name, x0, y0, z0, x1, y1, z1, facing, back="METAL"):
    x0, x1 = min(x0, x1), max(x0, x1); y0, y1 = min(y0, y1), max(y0, y1)
    return box(x0, y0, z0, x1, y1, z1, back, **{facing: fit_spec(name, facing, x0, y0, x1, y1, z0, z1)})
def wall_x(x0, x1, y0, y1, z0, z1, tex, openings=(), **kw):
    out, cur = [], x0
    for a, b, oz0, oz1 in sorted(openings):
        if a > cur: out.append(box(cur, y0, z0, a, y1, z1, tex, **kw))
        if oz0 > z0: out.append(box(a, y0, z0, b, y1, oz0, tex, **kw))
        if oz1 < z1: out.append(box(a, y0, oz1, b, y1, z1, tex, **kw))
        cur = b
    if cur < x1: out.append(box(cur, y0, z0, x1, y1, z1, tex, **kw))
    return out
def wall_y(y0, y1, x0, x1, z0, z1, tex, openings=(), **kw):
    out, cur = [], y0
    for a, b, oz0, oz1 in sorted(openings):
        if a > cur: out.append(box(x0, cur, z0, x1, a, z1, tex, **kw))
        if oz0 > z0: out.append(box(x0, a, z0, x1, b, oz0, tex, **kw))
        if oz1 < z1: out.append(box(x0, a, oz1, x1, b, z1, tex, **kw))
        cur = b
    if cur < y1: out.append(box(x0, cur, z0, x1, y1, z1, tex, **kw))
    return out

W, DECOR, ENTS, BENTS, ITEMS, SPAWNS = [], [], [], [], [], []
def point(cls, px_, py_, pz_, **kv):
    s = '{\n"classname" "%s"\n"origin" "%g %g %g"\n' % (cls, px_, py_, pz_)
    for k, v in kv.items(): s += '"%s" "%s"\n' % (k, v)
    ENTS.append(s + "}\n")
def light(x, y, z, rgb, i): point("light", round(x), round(y), round(z), _light="%d %d %d %d" % (*rgb, i))
def bent(cls, brushes, **kv):
    s = '{\n"classname" "%s"\n' % cls + "".join('"%s" "%s"\n' % kv_ for kv_ in kv.items())
    BENTS.append(s + "".join(brushes) + "}\n")
def item(cls, x, y, z): ITEMS.append((cls, float(x), float(y), float(z)))
def spawn(x, y, z, yaw): SPAWNS.append((float(x), float(y), float(z) + 37, yaw))

# ------------------------------------------------------------------ layout constants
XW, XE, YS, YN = -3072, 1920, -1472, 1472      # playable extent
PARK = (-896, -448, 896, 448)
ROAD = (-1152, -704, 1152, 704)                # outer edge of the loop road
WSTREET = (-3072, -128, -1152, 128)
EB = -512                                      # earth bottom
BF, BC = -176, -64                             # basement / tunnel floor and tunnel ceiling
HALL = (-832, -384, 832, 384); HF, HC = -464, -48
ROOF = 144
HXS = [(-1408, -896), (-640, -128), (128, 640), (896, 1408)]
BLUE = 2
SIDINGS = ["SIDING_CREAM", "SIDING_RED", "SIDING_BLUE", "SIDING_GRAY", "SIDING_GREEN", "SIDING_YELLOW",
           "SIDING_WHITE", "SIDING_BROWN", "SIDING_SAGE"]
HOUSES = []
for i, (x0, x1) in enumerate(HXS):
    HOUSES.append(dict(name="N%d" % i, cx=(x0 + x1) / 2, cy=1056, rot=0, siding=SIDINGS[i], blue=(i == BLUE)))
for i, (x0, x1) in enumerate(HXS):
    HOUSES.append(dict(name="S%d" % i, cx=(x0 + x1) / 2, cy=-1056, rot=2, siding=SIDINGS[4 + i] if i < 4 else "SIDING_GRAY", blue=False))
HOUSES.append(dict(name="E", cx=1504, cy=0, rot=1, siding="SIDING_SAGE", blue=False))
GARAGES = [(1280, 448, 1600, 704), (1280, -704, 1600, -448)]

def L2W(h, x, y):
    c, r = (h["cx"], h["cy"]), h["rot"]
    if r == 0: return c[0] + x, c[1] + y
    if r == 2: return c[0] - x, c[1] - y
    return c[0] + y, c[1] - x                    # rot 1: local front (-y) faces world -x
def footprint(h):
    a = L2W(h, -256, -224); b = L2W(h, 256, 224)
    return min(a[0], b[0]), min(a[1], b[1]), max(a[0], b[0]), max(a[1], b[1])
FMAP = {0: {"n": "n", "s": "s", "e": "e", "w": "w"}, 2: {"n": "s", "s": "n", "e": "w", "w": "e"},
        1: {"n": "e", "s": "w", "e": "s", "w": "n"}}

# ------------------------------------------------------------------ earth with holes (tunnels, hall, basements)
HOLES = []   # x0, y0, x1, y1, zl, zh, floor_tex, ceil_tex
for h in HOUSES:
    fx0, fy0, fx1, fy1 = footprint(h)
    HOLES.append((fx0, fy0, fx1, fy1, -192, 0, None, None))
ring = [(-1216, 512, 1216, 640), (-1216, -640, 1216, -512), (-1088, -512, -960, 512), (960, -512, 1088, 512)]
for r in ring: HOLES.append((*r, BF, BC, "TUNNELFLOOR", "TUNNELWALL"))
for i, h in enumerate(HOUSES):
    if h["rot"] == 1:
        HOLES.append((1088, -48, 1280, 48, BF, BC, "TUNNELFLOOR", "TUNNELWALL"))
    else:
        cx = h["cx"] + (-80 if h["blue"] else 0)
        y0, y1 = (640, 832) if h["rot"] == 0 else (-832, -640)
        HOLES.append((cx - 48, y0, cx + 48, y1, BF, BC, "TUNNELFLOOR", "TUNNELWALL"))
HOLES.append((-960, -48, -832, 48, BF, BC, "TUNNELFLOOR", "TUNNELWALL"))     # BattleAxe connector
HOLES.append((832, -48, 960, 48, BF, BC, "TUNNELFLOOR", "TUNNELWALL"))       # Crescent connector
HOLES.append((*HALL, HF, HC, "ARCADECARPET", "HALLCEIL"))
SRV = (-700, -560, -420, -384); SRV_Z = (HF, -336)
HOLES.append((*SRV, *SRV_Z, "TUNNELFLOOR", "TUNNELWALL"))
PX0, PX1 = 464, 560                                                          # blue house reveal passage
for j in range(1, 19):                                                       # 8-unit risers, 12-unit treads
    y0, y1 = 832 - 12 * j, 844 - 12 * j
    HOLES.append((PX0, y0, PX1, y1, BF - 8 * j, -192 if j >= 17 else -48, "STEP", "TUNNELWALL"))
HOLES.append((PX0, 384, PX1, 616, -320, -192, "TUNNELFLOOR", "TUNNELWALL"))

def surface_tex(x, y):
    in_outer = ROAD[0] <= x <= ROAD[2] and ROAD[1] <= y <= ROAD[3]
    in_park = PARK[0] <= x <= PARK[2] and PARK[1] <= y <= PARK[3]
    in_w = WSTREET[0] <= x <= WSTREET[2] and WSTREET[1] <= y <= WSTREET[3]
    return "SNOWROAD" if (in_outer and not in_park) or in_w else "SNOW"

xs = sorted({XW, XE, ROAD[0], ROAD[2], PARK[0], PARK[2]} | {v for hh in HOLES for v in (hh[0], hh[2])})
ys = sorted({YS, YN, ROAD[1], ROAD[3], PARK[1], PARK[3], WSTREET[1], WSTREET[3]} | {v for hh in HOLES for v in (hh[1], hh[3])})
def cell_sig(xa, xb, ya, yb):
    mx, my = (xa + xb) / 2, (ya + yb) / 2
    hs = [hh for hh in HOLES if hh[0] <= mx <= hh[2] and hh[1] <= my <= hh[3]]
    hs.sort(key=lambda hh: hh[4])
    solids, z, ftop = [], EB, None
    for hh in hs:
        if hh[4] > z: solids.append((z, hh[4], hh[6], ftop))
        ftop = hh[7]; z = max(z, hh[5])
    if z < 0: solids.append((z, 0, surface_tex(mx, my), ftop))
    return tuple(solids)
strips = []
for ya, yb in zip(ys, ys[1:]):
    runs, cur = [], None
    for xa, xb in zip(xs, xs[1:]):
        sg = cell_sig(xa, xb, ya, yb)
        if cur and cur[2] == sg and cur[1] == xa: cur[1] = xb
        else:
            if cur: runs.append(tuple(cur))
            cur = [xa, xb, sg]
    runs.append(tuple(cur)); strips.append((ya, yb, runs))
rects = {}
for ya, yb, runs in strips:
    for xa, xb, sg in runs:
        k = (xa, xb, sg)
        if k in rects and rects[k][1] == ya: rects[k][1] = yb
        else:
            if k in rects: W.extend(rects.pop(k)[2])
            rects[k] = [ya, yb, []]
        r = rects[k]
        r[2] = [box(xa, r[0], z0, xb, r[1], z1, "TUNNELWALL", top=tt or "TUNNELWALL", bottom=bt or "TUNNELWALL")
                for z0, z1, tt, bt in sg if z1 > z0]
for k, r in rects.items(): W.extend(r[2])
EARTH_N = len(W)

# ------------------------------------------------------------------ sky shell
SK0, SK1 = EB - 16, 1100
for b in [(XW - 32, YS - 32, SK1, XE + 32, YN + 32, SK1 + 16), (XW - 32, YS - 32, SK0 - 16, XE + 32, YN + 32, SK0),
          (XW - 32, YS - 32, SK0, XW, YN + 32, SK1), (XE, YS - 32, SK0, XE + 32, YN + 32, SK1),
          (XW, YS - 32, SK0, XE, YS, SK1), (XW, YN, SK0, XE, YN + 32, SK1)]:
    W.append(box(*b, "sky"))

# ------------------------------------------------------------------ houses
GLASS = []
def house(h):
    r = h["rot"]
    def B(x0, y0, z0, x1, y1, z1, tex, dest=W, **faces):
        a = L2W(h, x0, y0); b = L2W(h, x1, y1)
        dest.append(box(a[0], a[1], z0, b[0], b[1], z1, tex, **{FMAP[r].get(k, k): v for k, v in faces.items()}))
    def S(name, x0, y0, z0, x1, y1, z1, facing, dest=W, back="METAL"):
        a = L2W(h, x0, y0); b = L2W(h, x1, y1)
        dest.append(sign(name, a[0], a[1], z0, b[0], b[1], z1, FMAP[r][facing], back=back))
    def P(x, y): return L2W(h, x, y)
    sd = h["siding"]
    # basement shell
    B(-256, -224, -192, 256, 224, BF, "CONCRETE" if False else "TUNNELWALL", top="CARPET" if not h["blue"] else "ARCADECARPET")
    doors = []
    if r == 1: doors = [(-48, 48, BF, BC)]
    else:
        sx = -80 if h["blue"] else 0
        doors = [(sx - 48, sx + 48, BF, BC)]
        if r == 2: doors = [(-sx - 48, -sx + 48, BF, BC)]
    if h["blue"]: doors.append((PX0 - h["cx"], PX1 - h["cx"], BF, BC))
    for (a, b, z0, z1) in [(o[0], o[1], o[2], o[3]) for o in [(-256, 256, BF, 0)]]:
        pass
    # front basement wall (local y -224..-208) with doors, back wall, sides
    cur = -256
    for a, b, z0, z1 in sorted(doors):
        if a > cur: B(cur, -224, BF, a, -208, 0, "TUNNELWALL", n="BASEWALL")
        B(a, -224, z1, b, -208, 0, "TUNNELWALL", n="BASEWALL")
        cur = b
    B(cur, -224, BF, 256, -208, 0, "TUNNELWALL", n="BASEWALL")
    B(-256, 208, BF, 256, 224, 0, "TUNNELWALL", s="BASEWALL")
    B(-256, -208, BF, -240, 208, 0, "TUNNELWALL", e="BASEWALL")
    B(240, -208, BF, 256, 208, 0, "TUNNELWALL", w="BASEWALL")
    # ground slab with stair opening, stairs, rails
    B(-240, -208, -16, 240, 128, 0, "WOODFLOOR", bottom="CEILTILE")
    B(-240, 128, -16, -40, 208, 0, "WOODFLOOR", bottom="CEILTILE")
    B(200, 128, -16, 240, 208, 0, "WOODFLOOR", bottom="CEILTILE")
    for j in range(1, 22):                              # 8-unit risers down to the basement floor (BF = -176)
        B(200 - 12 * j, 128, BF, 212 - 12 * j, 208, -8 * j, "BASEWALL", top="STEP")
    B(-40, 124, 0, 200, 128, 36, "TRIM", dest=DECOR)
    # ground floor walls with door + windows
    GH = 128
    def wall_local(x0, x1, y0, y1, openings, inner):
        cur = x0
        for a, b, z0, z1 in sorted(openings):
            if a > cur: B(cur, y0, 0, a, y1, GH, sd, **{inner: "WALLPAPER"})
            if z0 > 0: B(a, y0, 0, b, y1, z0, sd, **{inner: "WALLPAPER"})
            B(a, y0, z1, b, y1, GH, sd, **{inner: "WALLPAPER"}); cur = b
        if cur < x1: B(cur, y0, 0, x1, y1, GH, sd, **{inner: "WALLPAPER"})
    wins = [(-200, -104, 40, 100), (104, 200, 40, 100)]
    wall_local(-256, 256, -224, -208, [(-40, 40, 0, 112)] + wins, "n")
    wall_local(-256, 256, 208, 224, [(-160, -64, 40, 100)], "s")
    def side_wall(x0, x1, inner):
        cur = -208
        for a, b, z0, z1 in [(-120, -24, 40, 100)]:
            B(x0, cur, 0, x1, a, GH, sd, **{inner: "WALLPAPER"})
            B(x0, a, 0, x1, b, z0, sd, **{inner: "WALLPAPER"}); B(x0, a, z1, x1, b, GH, sd, **{inner: "WALLPAPER"}); cur = b
        B(x0, cur, 0, x1, 208, GH, sd, **{inner: "WALLPAPER"})
    side_wall(-256, -240, "e"); side_wall(240, 256, "w")
    for (x0, y0, x1, y1, z0, z1) in [(-200, -218, -104, -214, 40, 100), (104, -218, 200, -214, 40, 100),
                                     (-160, 214, -64, 218, 40, 100), (-250, -120, -246, -24, 40, 100), (246, -120, 250, -24, 40, 100)]:
        a = L2W(h, x0, y0); b = L2W(h, x1, y1)
        GLASS.append(box(a[0], a[1], z0, b[0], b[1], z1, "GLASS"))
    # roof slab (walkable, snowy) + chimney for cover
    B(-256, -224, GH, 256, 224, ROOF, "TRIM", top="ROOFSNOW", bottom="CEILING")
    B(-190, 40, ROOF, -140, 90, 196, "BRICK", top="ROOFSNOW")
    # interior: partition, living room, kitchen
    B(40, -208, 0, 56, -20, GH, "WALLPAPER")
    B(-236, -150, 0, -200, -30, 36, "COUCH"); B(-240, -150, 0, -228, -30, 56, "COUCH")
    S("NEWSCAST", 30, -140, 36, 40, -80, 76, "w", back="TVBLACK")
    B(-140, -120, 0, -90, -60, 18, "STEP")
    B(200, -200, 0, 240, 60, 36, "TRIM", top="COUNTER")
    B(206, 70, 0, 240, 110, 80, "FRIDGE")
    # porch
    B(-64, -272, 0, 64, -224, 8, "TRIM", top="WOODFLOOR")
    B(56, -232, 92, 68, -224, 110, "METAL", dest=DECOR, s="LAMP")
    px, py = P(62, -250); light(px, py, 98, (255, 190, 120), 150)
    px, py = P(-100, -60); light(px, py, 100, (255, 225, 180), 120)
    px, py = P(130, 40); light(px, py, 100, (255, 235, 200), 100)
    # basement dressing
    if h["blue"]:
        # the small computer room
        for y0 in (-170, -90, -10):
            B(-240, y0, BF, -200, y0 + 70, BF + 30, "DESK")
            S("CRT_TERM" if y0 == -90 else "CRT_SAVER", -236, y0 + 20, BF + 34, -232, y0 + 50, BF + 58, "e", dest=DECOR, back="CRT")
            B(-240, y0 + 18, BF + 30, -236, y0 + 52, BF + 60, "CRT")
        S("POSTER98", -80, 204, BF + 40, -16, 208, BF + 136, "s", dest=DECOR)
        S("TS_JC", PX0 - h["cx"], -208, BC + 4, PX1 - h["cx"], -204, BC + 28, "n", dest=DECOR)
        px, py = P(-120, -60); light(px, py, -60, (120, 170, 255), 140)
        px, py = P(120, -150); light(px, py, -60, (255, 110, 210), 90)
        px, py = P(-120, -60); point("ambient_generic", px, py, -100, message="justincon/fans_loop.wav", health=6, spawnflags=2, pitch=100, pitchstart=100)
    else:
        B(-232, -150, BF, -196, -30, BF + 36, "COUCH")
        B(-60, -60, BF, 60, 20, BF + 30, "DESK")
        px, py = P(0, -60); light(px, py, -60, (255, 220, 180), 120)
    if h["name"] in ("S1", "N3"):
        S("EMB_AXE" if h["cx"] < 0 else "EMB_MOON", -32, -208, BC + 8, 32, -204, BC + 56, "n", dest=DECOR)

for h in HOUSES: house(h)
for gb in GLASS:
    bent("func_breakable", [gb], rendermode=2, renderamt=80, material=0, health=20)

# garages
(gx0, gy0, gx1, gy1) = GARAGES[0]
W.append(box(gx0, gy0, ROOF - 16, gx1, gy1, ROOF, "SIDING_GRAY", top="ROOFSNOW", bottom="CEILING"))
W.append(box(gx0, gy0, 0, gx1, gy0 + 16, ROOF - 16, "SIDING_GRAY", n="TUNNELWALL"))
W.append(box(gx0, gy1 - 16, 0, gx1, gy1, ROOF - 16, "SIDING_GRAY", s="TUNNELWALL"))
W.append(box(gx1 - 16, gy0 + 16, 0, gx1, gy1 - 16, ROOF - 16, "SIDING_GRAY", w="TUNNELWALL"))
W.extend(wall_y(gy0 + 16, gy1 - 16, gx0, gx0 + 16, 0, ROOF - 16, "SIDING_GRAY", [(496, 656, 0, 112)], e="TUNNELWALL"))
W.append(box(gx0 + 16, gy0 + 16, 0, gx1 - 16, gy1 - 16, 1, "TUNNELFLOOR"))
GARAGE_DOOR = box(gx0 + 2, 496, 0, gx0 + 14, 656, 112, "SIDING_GRAY", w=fit_spec("GARAGEDOOR", "w", gx0, 496, gx0 + 2, 656, 0, 112))
(sx0, sy0, sx1, sy1) = GARAGES[1]
W.append(box(sx0, sy0, 0, sx1, sy1, ROOF, "SIDING_GRAY", top="ROOFSNOW", w=fit_spec("GARAGEDOOR", "w", sx0, sy0, sx1, sy1, 0, ROOF)))

# ------------------------------------------------------------------ roofs: bridges, exterior stairs, snowbank
def bridge(x0, y0, x1, y1):
    W.append(box(x0, y0, ROOF - 16, x1, y1, ROOF, "PLANK", top="ROOFSNOW"))
    if x1 - x0 > y1 - y0:
        DECOR.append(box(x0, y0, ROOF, x1, y0 + 4, ROOF + 32, "PLANK")); DECOR.append(box(x0, y1 - 4, ROOF, x1, y1, ROOF + 32, "PLANK"))
        W.append(box(x0, y0, ROOF, x1, y0 + 4, ROOF + 32, "PLANK")); W.append(box(x0, y1 - 4, ROOF, x1, y1, ROOF + 32, "PLANK"))
    else:
        W.append(box(x0, y0, ROOF, x0 + 4, y1, ROOF + 32, "PLANK")); W.append(box(x1 - 4, y0, ROOF, x1, y1, ROOF + 32, "PLANK"))
for (a0, a1), (b0, b1) in zip(HXS, HXS[1:]):
    bridge(a1, 1024, b0, 1088); bridge(a1, -1088, b0, -1024)
bridge(1312, 704, 1376, 832); bridge(1408, 256, 1472, 448); bridge(1408, -448, 1472, -256); bridge(1312, -832, 1376, -704)
for j in range(1, 19):                                     # exterior stairs (8-unit risers): N0 west side
    W.append(box(-1472, 1000 + 12 * (j - 1), 0, -1408, 1000 + 12 * j, 8 * j, "PLANK", top="ROOFSNOW"))
for j in range(1, 19):                                     # exterior stairs: east house south side
    W.append(box(1700 - 12 * j, -320, 0, 1712 - 12 * j, -256, 8 * j, "PLANK", top="ROOFSNOW"))
W.append(hull_brush([(-1760, -1240, 0), (-1760, -1112, 0), (-1408, -1240, 0), (-1408, -1112, 0),
                     (-1408, -1240, ROOF), (-1408, -1112, ROOF)], "SNOW", top="SNOW"))     # climbable snowbank

# ------------------------------------------------------------------ outdoors: rink, snowman, trees, lamps, cars, fences
W.append(box(-384, -192, 0, 384, 192, 2, "ICE"))
for (x0, y0, x1, y1) in [(-392, -200, -32, -192), (32, -200, 392, -192), (-392, 192, -32, 200), (32, 192, 392, 200),
                         (-392, -192, -384, 192), (384, -192, 392, 192)]:
    W.append(box(x0, y0, 0, x1, y1, 28, "RINKBOARD"))
def sphere_pts(c, r, n=10, rings=5):
    pts = []
    for i in range(1, rings):
        ph = math.pi * i / rings
        for j in range(n):
            th = 2 * math.pi * (j + 0.5 * (i % 2)) / n
            pts.append((c[0] + r * math.sin(ph) * math.cos(th), c[1] + r * math.sin(ph) * math.sin(th), c[2] + r * math.cos(ph)))
    return pts + [(c[0], c[1], c[2] + r), (c[0], c[1], c[2] - r)]
SM = (232, 768)
for (zc, rr) in ((26, 28), (72, 21), (108, 15)):
    W.append(hull_brush(sphere_pts((SM[0], SM[1], zc), rr), "SNOWMAN"))
def cyl(cx, cy, r, z0, z1, tex, n=10):
    return hull_brush([(cx + r * math.cos(2 * math.pi * k / n), cy + r * math.sin(2 * math.pi * k / n), z) for k in range(n) for z in (z0, z1)], tex)
W.append(cyl(SM[0], SM[1], 19, 121, 124, "COAL")); W.append(cyl(SM[0], SM[1], 11, 124, 150, "COAL"))
DECOR.append(cyl(SM[0], SM[1], 12, 135, 139, "SCARF"))
DECOR.append(cyl(SM[0], SM[1], 17, 86, 92, "SCARF"))
DECOR.append(beam((SM[0], SM[1] - 13, 108), (SM[0], SM[1] - 30, 106), 4, "CARROT"))
for dx in (-5, 5): DECOR.append(box(SM[0] + dx - 2, SM[1] - 15, 113, SM[0] + dx + 2, SM[1] - 12, 117, "COAL"))
for dz in (60, 75, 90): DECOR.append(box(SM[0] - 2, SM[1] - 22, dz - 2, SM[0] + 2, SM[1] - 19, dz + 2, "COAL"))
DECOR.append(beam((SM[0] - 18, SM[1], 76), (SM[0] - 50, SM[1] - 6, 100), 3, "BARK"))
DECOR.append(beam((SM[0] + 18, SM[1], 76), (SM[0] + 48, SM[1] + 4, 96), 3, "BARK"))
def tree(x, y, s=1.0):
    W.append(box(x - 8, y - 8, 0, x + 8, y + 8, 40 * s, "BARK"))
    for z0, r, z1 in ((36, 70, 190), (120, 52, 260), (190, 34, 320)):
        W.append(hull_brush([(x + r * s * math.cos(2 * math.pi * k / 8), y + r * s * math.sin(2 * math.pi * k / 8), z0 * s) for k in range(8)] + [(x, y, z1 * s)], "PINE"))
for (x, y, s) in [(-700, -300, 1), (-640, 320, 1.1), (680, 300, 0.9), (720, -280, 1), (-560, 0, 0.8), (560, 40, 0.85),
                  (-1700, 1350, 1.2), (1700, 1350, 1.1), (-1700, -1350, 1.1), (1750, -1300, 1.2), (-2000, 520, 1.3),
                  (-2400, -560, 1.2), (-2700, 600, 1.4), (-1600, -600, 1)]:
    tree(x, y, s)
LAMPS = [(-768, 720), (0, 720), (768, 720), (-768, -720), (0, -720), (768, -720), (1168, 360), (1168, -360),
         (-1600, 150), (-2200, -150), (-2800, 150), (-1168, 500)]
for (x, y) in LAMPS:
    W.append(box(x - 4, y - 4, 0, x + 4, y + 4, 196, "METAL"))
    DECOR.append(box(x - 12, y - 12, 188, x + 12, y + 12, 198, "METAL", bottom="LAMP"))
    light(x, y, 176, (255, 200, 130), 300)
for (x, y) in [(-420, -260), (420, 260), (-420, 260), (420, -260)]:
    light(x, y, 160, (220, 230, 255), 160)
def car(x, y, paint, along_x=True):
    L, Wd = 200, 84
    if along_x:
        W.append(box(x - L / 2, y - Wd / 2, 14, x + L / 2, y + Wd / 2, 46, paint, top="SNOW"))
        W.append(box(x - 50, y - Wd / 2 + 6, 46, x + 50, y + Wd / 2 - 6, 82, "CARGLASS", top="SNOW"))
    else:
        W.append(box(x - Wd / 2, y - L / 2, 14, x + Wd / 2, y + L / 2, 46, paint, top="SNOW"))
        W.append(box(x - Wd / 2 + 6, y - 50, 46, x + Wd / 2 - 6, y + 50, 82, "CARGLASS", top="SNOW"))
    for dx in (-62, 62):
        for dy in (-Wd / 2, Wd / 2 - 12):
            (W.append(box(x + dx - 14, y + dy - 2, 0, x + dx + 14, y + dy + 14, 26, "TIRE")) if along_x else
             W.append(box(x + dy - 2, y + dx - 14, 0, x + dy + 14, y + dx + 14, 26, "TIRE")))
for (x, y, p, ax) in [(-480, 650, "CARRED", True), (560, -650, "CARBLUE", True), (-1080, 200, "CARSILVER", False),
                      (-2000, -80, "CARBLUE", True), (1080, -150, "CARRED", False)]:
    car(x, y, p, ax)
W.append(box(XW + 32, YN - 40, 0, XE - 32, YN - 32, 96, "WOODFENCE"))
W.append(box(XW + 32, YS + 32, 0, XE - 32, YS + 40, 96, "WOODFENCE"))
W.append(box(XE - 40, YS + 40, 0, XE - 32, YN - 40, 96, "WOODFENCE"))
W.append(box(-3072, 400, 0, -1840, 408, 72, "WOODFENCE")); W.append(box(-3072, -408, 0, -1840, -400, 72, "WOODFENCE"))
W.append(box(-3040, -128, 0, -3024, 128, 40, "RINKBOARD"))
W.append(box(-3020, -6, 0, -3014, 6, 70, "METAL")); W.append(sign("DEADEND", -3014, -40, 70, -3010, 40, 110, "e"))
W.append(box(-1180, 160, 0, -1174, 166, 110, "METAL")); W.append(sign("STREETSIGN", -1220, 166, 110, -1100, 170, 140, "n"))

# ------------------------------------------------------------------ ring tunnel dressing: faction colours
def tunnel_color(x):
    if x < -500: return ("STRIP_PINK", "STRIP_CORAL", (255, 80, 170), (255, 130, 100))
    if x > 500: return ("STRIP_CYAN", "STRIP_VIOLET", (60, 220, 255), (150, 100, 255))
    return ("STRIP_WHITE", "STRIP_WHITE", (220, 225, 255), (220, 225, 255))
for xa in range(-1216, 1216, 152):
    xb = xa + 152; s1, s2, c1, c2 = tunnel_color((xa + xb) / 2)
    for yy, (y0, y1) in (("n", (512, 640)), ("s", (-640, -512))):
        DECOR.append(box(xa, y0, BF + 6, xb, y0 + 2, BF + 10, s1)); DECOR.append(box(xa, y1 - 2, BF + 6, xb, y1, BF + 10, s2))
        DECOR.append(box(xa, y0, BC - 10, xb, y0 + 2, BC - 6, s2)); DECOR.append(box(xa, y1 - 2, BC - 10, xb, y1, BC - 6, s1))
for ya in range(-512, 512, 128):
    for side, (x0, x1), cols in (("w", (-1088, -960), ("STRIP_PINK", "STRIP_CORAL")), ("e", (960, 1088), ("STRIP_CYAN", "STRIP_VIOLET"))):
        DECOR.append(box(x0, ya, BF + 6, x0 + 2, ya + 128, BF + 10, cols[0])); DECOR.append(box(x1 - 2, ya, BF + 6, x1, ya + 128, BF + 10, cols[1]))
        DECOR.append(box(x0, ya, BC - 10, x0 + 2, ya + 128, BC - 6, cols[1])); DECOR.append(box(x1 - 2, ya, BC - 10, x1, ya + 128, BC - 6, cols[0]))
TL = [(-1024, 576), (-600, 576), (0, 576), (600, 576), (1024, 576), (-1024, -576), (-600, -576), (0, -576), (600, -576),
      (1024, -576), (-1024, -300), (-1024, 300), (1024, -300), (1024, 300), (-1024, 0), (1024, 0)]
for (x, y) in TL:
    s1, s2, c1, c2 = tunnel_color(x)
    DECOR.append(box(x - 6, y - 6, BC - 4, x + 6, y + 6, BC, "METAL", bottom="LAMP"))
    light(x, y, BC - 24, c1, 110)
for (x, y, f, name) in [(-1088, 0, "e", "EMB_AXE"), (1088, 0, "w", "EMB_MOON")]:
    pass
DECOR.append(sign("EMB_AXE", -1088, 120, BF + 30, -1086, 184, BF + 94, "e"))
DECOR.append(sign("EMB_AXE", -1088, -184, BF + 30, -1086, -120, BF + 94, "e"))
DECOR.append(sign("EMB_MOON", 1086, 120, BF + 30, 1088, 184, BF + 94, "w"))
DECOR.append(sign("EMB_MOON", 1086, -184, BF + 30, 1088, -120, BF + 94, "w"))
DECOR.append(sign("TS_AXE", -700, 638, BF + 70, -580, 640, BF + 100, "s"))
DECOR.append(sign("TS_MOON", 580, 638, BF + 70, 700, 640, BF + 100, "s"))
DECOR.append(sign("TS_AXE", -700, -640, BF + 70, -580, -638, BF + 100, "n"))
DECOR.append(sign("TS_MOON", 580, -640, BF + 70, 700, -638, BF + 100, "n"))
light(-896, 0, BC - 24, (255, 80, 170), 120); light(896, 0, BC - 24, (60, 220, 255), 120)
light(512, 740, -100, (180, 150, 255), 70); light(512, 500, -230, (255, 90, 200), 80)

# ------------------------------------------------------------------ JustinCon hall
X0, Y0, X1, Y1 = HALL
LIN = 8
W.extend(wall_y(Y0, Y1, X0, X0 + LIN, HF, HC, "HALLWALL", [(-48, 48, BF, BC)]))           # west lining (BattleAxe door)
W.extend(wall_y(Y0, Y1, X1 - LIN, X1, HF, HC, "HALLWALL", [(-48, 48, BF, BC)]))           # east lining (Crescent door)
W.extend(wall_x(X0 + LIN, X1 - LIN, Y1 - LIN, Y1, HF, HC, "HALLWALL", [(PX0, PX1, -320, -192)]))
SDOOR = (-620, -500)
W.extend(wall_x(X0 + LIN, X1 - LIN, Y0, Y0 + LIN, HF, HC, "HALLWALL", [(SDOOR[0], SDOOR[1], HF, -352)]))
IX0, IX1, IY0, IY1 = X0 + LIN, X1 - LIN, Y0 + LIN, Y1 - LIN
# BattleAxe entrance: landing + stairs down (north) along the west wall
W.append(box(IX0, -48, HF, IX0 + 64, 48, BF, "HALLWALL", top="STAGEFLOOR"))
for j in range(1, 35):                                     # 8-unit risers
    W.append(box(IX0, 48 + 8 * (j - 1), HF, IX0 + 64, 48 + 8 * j, BF - 8 * j, "HALLWALL", top="ARCADECARPET"))
# Crescent entrance: landing + stairs down (south) along the east wall
W.append(box(IX1 - 64, -48, HF, IX1, 48, BF, "HALLWALL", top="STAGEFLOOR"))
for j in range(1, 35):
    W.append(box(IX1 - 64, -48 - 8 * j, HF, IX1, -48 - 8 * (j - 1), BF - 8 * j, "HALLWALL", top="ARCADECARPET"))
# blue-house reveal: mezzanine + stairs down (west) along the north wall
W.append(box(PX0 - 32, 264, HF, PX1 + 32, IY1, -320, "HALLWALL", top="STAGEFLOOR"))
W.append(box(PX0 - 32, 260, -320, PX1 + 32, 264, -288, "HALLWALL"))
DECOR.append(box(PX0 - 32, 259, -288, PX1 + 32, 265, -284, "STRIP_PINK"))
for j in range(1, 17):                                     # 8-unit risers
    W.append(box(PX0 - 32 - 12 * j, 296, HF, PX0 - 32 - 12 * (j - 1), IY1, -320 - 8 * j, "HALLWALL", top="ARCADECARPET"))
# stage, backdrop, match board, neon sign
W.append(box(-320, IY0, HF, 320, -280, HF + 32, "HALLWALL", top="STAGEFLOOR"))
for x0, x1 in ((-140, 140), (200, 300)):                  # 8-unit steps up onto the stage, centre and teleport pad
    for i, z in enumerate((24, 16, 8)):
        W.append(box(x0, -280 + 12 * i, HF, x1, -268 + 12 * i, HF + z, "HALLWALL", top="STAGEFLOOR"))
DECOR.append(sign("STAGEBACK", -232, IY0, HF + 32, 232, IY0 + 2, -200, "n"))
DECOR.append(sign("MATCHBOARD", -168, IY0 + 2, -352, 168, IY0 + 6, -268, "n"))
DECOR.append(sign("NEON_JC", -240, IY0, -192, 240, IY0 + 6, -72, "n"))
# faction banners facing each other
DECOR.append(sign("BANNER_AXE", IX0, -300, -400, IX0 + 3, -140, -80, "e"))
DECOR.append(sign("BANNER_MOON", IX1 - 3, 140, -400, IX1, 300, -80, "w"))
DECOR.append(sign("EMB_AXE", IX0, -40, BC + 2, IX0 + 3, 40, -50, "e") if False else box(IX0, -48, BC, IX0 + 3, 48, BC + 8, "STRIP_PINK"))
DECOR.append(box(IX1 - 3, -48, BC, IX1, 48, BC + 8, "STRIP_CYAN"))
# glowing wall strips
for z in (HF + 14, -150):
    DECOR.append(box(IX0, IY0, z, IX0 + 2, IY1, z + 4, "STRIP_PINK" if z < -200 else "STRIP_CORAL"))
    DECOR.append(box(IX1 - 2, IY0, z, IX1, IY1, z + 4, "STRIP_CYAN" if z < -200 else "STRIP_VIOLET"))
    DECOR.append(box(IX0, IY1 - 2, z, 0, IY1, z + 4, "STRIP_CORAL")); DECOR.append(box(0, IY1 - 2, z, IX1, IY1, z + 4, "STRIP_VIOLET"))
    DECOR.append(box(IX0, IY0, z, -320, IY0 + 2, z + 4, "STRIP_PINK")); DECOR.append(box(320, IY0, z, IX1, IY0 + 2, z + 4, "STRIP_CYAN"))
# pods: 8 pods x 4 CRT workstations
TABLE_H = HF + 30
handle_slots = {}
west_handles = [h for i, h in enumerate(A.HANDLES) if i % 2 == 0]
east_handles = [h for i, h in enumerate(A.HANDLES) if i % 2 == 1]
fill = ["CRT_GAME", "CRT_SAVER", "CRT_TERM"]
PODS = [(x, y) for y in (144, -112) for x in (-480, -160, 160, 480)]
for pi, (px, py) in enumerate(PODS):
    side = "w" if px < 0 else "e"
    W.append(box(px - 56, py - 56, HF, px + 56, py + 56, TABLE_H, "DESK"))
    for k, f in enumerate(("n", "e", "s", "w")):
        dx, dy = {"n": (0, 1), "s": (0, -1), "e": (1, 0), "w": (-1, 0)}[f]
        # CRT body (solid) centred 20 units out from the pod centre, screen on its outer face
        cx, cy = px + dx * 20, py + dy * 20
        if dx == 0: bx0, bx1, by0, by1 = cx - 18, cx + 18, cy - 15, cy + 15
        else: bx0, bx1, by0, by1 = cx - 15, cx + 15, cy - 18, cy + 18
        W.append(box(bx0, by0, TABLE_H, bx1, by1, TABLE_H + 32, "CRT"))
        pool = west_handles if side == "w" else east_handles
        slot = pi * 4 + k
        hidx = [0, 5, 10, 15, 2, 7][:]  # placeholder
        tex = None
        order = (pi % 4) * 4 + k
        if order % 3 == 0 and len(handle_slots.get(side, [])) < 6:
            tex = A.handle_tex(pool[len(handle_slots.setdefault(side, []))]); handle_slots[side].append(tex)
        tex = tex or fill[(pi + k) % 3]
        # thin screen plate one unit in front of the body
        if f == "n": DECOR.append(sign(tex, bx0 + 3, by1, TABLE_H + 4, bx1 - 3, by1 + 1, TABLE_H + 28, "n", back="CRT"))
        if f == "s": DECOR.append(sign(tex, bx0 + 3, by0 - 1, TABLE_H + 4, bx1 - 3, by0, TABLE_H + 28, "s", back="CRT"))
        if f == "e": DECOR.append(sign(tex, bx1, by0 + 3, TABLE_H + 4, bx1 + 1, by1 - 3, TABLE_H + 28, "e", back="CRT"))
        if f == "w": DECOR.append(sign(tex, bx0 - 1, by0 + 3, TABLE_H + 4, bx0, by1 - 3, TABLE_H + 28, "w", back="CRT"))
        kx, ky = px + dx * 44, py + dy * 44
        DECOR.append(box(kx - (14 if dx == 0 else 5), ky - (5 if dx == 0 else 14), TABLE_H, kx + (14 if dx == 0 else 5), ky + (5 if dx == 0 else 14), TABLE_H + 2, "KEYBOARD"))
        chx, chy = px + dx * 86, py + dy * 86
        DECOR.append(box(chx - 12, chy - 12, HF, chx + 12, chy + 12, HF + 18, "CHAIR"))
        bx, by = px + dx * 98, py + dy * 98
        DECOR.append(box(bx - (12 if dx == 0 else 2), by - (2 if dx == 0 else 12), HF + 18, bx + (12 if dx == 0 else 2), by + (2 if dx == 0 else 12), HF + 50, "CHAIR"))
    light(px, py, -200, ((255, 90, 190) if px < 0 else (60, 220, 255)) if abs(px) > 300 else ((255, 140, 110) if px < 0 else (160, 110, 255)), 230)
assert sum(len(v) for v in handle_slots.values()) == 12, handle_slots
light(0, -320, -150, (255, 120, 220), 260); light(0, -200, -380, (120, 200, 255), 160)
light(0, 20, -120, (230, 220, 255), 200); light(-700, -250, -260, (255, 110, 90), 160); light(700, 250, -260, (150, 90, 255), 160)
light(PX0 + 48, 300, -250, (255, 255, 255), 90)
for (x, y) in [(-600, 150), (-250, 0), (250, 0), (600, -150), (0, 250), (0, -250)]:
    point("ambient_generic", x, y, -400, message="justincon/fans_loop.wav", health=7, spawnflags=2, pitch=100, pitchstart=100)

# ------------------------------------------------------------------ INTERACTIVE: "Restore the Power"
def ent(cls, brushes, **kv): bent(cls, brushes, **kv)
# breaker switches (3): north garage, south loop tunnel, far west end of the street
BRK = [box(gx1 - 32, 560, 40, gx1 - 16, 600, 90, "METAL", w="BREAKER"),
       box(-20, -640, -150, 20, -630, -110, "METAL", n="BREAKER"),
       box(-2956, 184, 50, -2944, 216, 90, "METAL", e="BREAKER")]
W.append(box(-2964, 196, 0, -2956, 204, 200, "METAL"))                    # utility pole for breaker 3
for i, br in enumerate(BRK):
    ent("func_button", [br], target="breaker_mm", wait=45, spawnflags=1, sounds=21, delay=0)
point("multi_manager", -100, -560, -120, targetname="breaker_mm", breaker_count=0, breaker_msg=0)
point("game_counter", -100, -560, -110, targetname="breaker_count", target="power_mm", health=3, frags=0, spawnflags=2)
point("game_text", -100, -560, -100, targetname="breaker_msg", message="A breaker was switched on.  (Flip all 3 to restore power)",
      **{"x": -1, "y": 0.7}, effect=0, color="255 220 90", color2="255 255 255", fadein=0.05, fadeout=0.5, holdtime=3, channel=2, spawnflags=1)
point("multi_manager", -100, -560, -90, targetname="power_mm", msg_power=0, boom=0.2, shake=0.2, alarm_snd=0.6,
      server_door=1.5, **{"alarm_snd#2": 16})
point("game_text", -100, -560, -80, targetname="msg_power", message="POWER RESTORED!  The JustinCon server room is open.",
      **{"x": -1, "y": 0.25}, effect=0, color="255 90 200", color2="60 230 255", fadein=0.05, fadeout=0.6, holdtime=5, channel=1, spawnflags=1)
# the transformer in the park blows when power comes back
W.append(box(-870, 380, 0, -800, 440, 72, "TRANSFORMER", top="METAL"))
point("env_explosion", -835, 410, 84, targetname="boom", iMagnitude=150, spawnflags=2)
point("env_shake", 0, 0, 0, targetname="shake", amplitude=8, frequency=30, duration=2.5, radius=5000, spawnflags=1)
for (x, y, z) in [(0, 0, -300), (-835, 410, 100), (-560, -470, -400)]:
    point("ambient_generic", x, y, z, targetname="alarm_snd", message="justincon/alarm.wav", health=9, spawnflags=4 | 16,
          pitch=100, pitchstart=100)
# server room behind a sliding door in the hall's south wall
ent("func_door", [box(SDOOR[0], Y0 + 1, HF, SDOOR[1], Y0 + 7, -352, "METAL", n=fit_spec("SERVERDOOR", "n", SDOOR[0], Y0, SDOOR[1], Y0 + 7, HF, -352),
                      s=fit_spec("SERVERDOOR", "s", SDOOR[0], Y0, SDOOR[1], Y0 + 7, HF, -352))],
    targetname="server_door", angles="0 -1 0", speed=110, lip=8, wait=14, movesnd=1, stopsnd=1)
DECOR.append(sign("+0MARQUEE", SDOOR[0] - 20, Y0 + 8, -344, SDOOR[1] + 20, Y0 + 10, -324, "n"))
for x0, face in ((SRV[0] + 4, "e"), (SRV[2] - 36, "w")):
    for y0 in (-544, -480):
        W.append(box(x0, y0, HF, x0 + 32, y0 + 56, -372, "METAL", **{face: "SERVERRACK"}))
ent("func_button", [box(-580, SRV[1], -420, -540, SRV[1] + 6, -380, "METAL", n=fit_spec("ARMORY", "n", -580, SRV[1], -540, SRV[1] + 6, -420, -380))], target="armory", wait=3, spawnflags=1, sounds=9)
point("game_player_equip", -560, -500, -440, targetname="armory", spawnflags=1, **{
    "weapon_crowbar": 1, "weapon_9mmhandgun": 1, "weapon_357": 1, "weapon_9mmAR": 1, "weapon_shotgun": 1, "weapon_crossbow": 1,
    "weapon_rpg": 1, "weapon_gauss": 1, "weapon_egon": 1, "weapon_hornetgun": 1, "weapon_handgrenade": 1, "weapon_satchel": 1,
    "weapon_tripmine": 1, "weapon_snark": 1, "ammo_buckshot": 2, "ammo_9mmAR": 2, "ammo_357": 2, "ammo_rpgclip": 2, "ammo_gaussclip": 2})
item("weapon_gauss", -600, -460, HF + 10); item("weapon_egon", -520, -460, HF + 10); item("item_battery", -560, -420, HF + 10)
light(-560, -470, -360, (255, 70, 70), 120); light(-560, -420, -380, (200, 200, 255), 60)
# stage marquee (animated) + teleport pad on the stage -> blue house roof
DECOR.append(sign("+0MARQUEE", -320, -280, HF, 320, -278, HF + 32, "n"))
DECOR.append(box(220, -360, HF + 32, 290, -300, HF + 33, "TELEPAD"))
ent("trigger_teleport", [box(222, -358, HF + 33, 288, -302, HF + 90, "AAATRIGGER")], target="tele_roof")
point("info_teleport_destination", 384, 1150, ROOF, targetname="tele_roof", angles="0 270 0")
light(255, -330, -380, (255, 120, 230), 90)
# breakable north-garage door + crates that drop weapons
ent("func_breakable", [GARAGE_DOOR], material=2, health=150, gibmodel="", explodemagnitude=0)
for (cx, cy, so) in ((1400, 520, 14), (1460, 640, 8), (1520, 520, 2)):
    ent("func_breakable", [box(cx - 20, cy - 20, 1, cx + 20, cy + 20, 41, "CRATE")], material=1, health=30, spawnobject=so)
light(1440, 576, 110, (255, 230, 190), 110)
# steam-vent updraft behind S2 house: jump in to ride up to the roof
W.append(box(344, -1356, 0, 424, -1284, 6, "VENT"))
ent("trigger_push", [box(352, -1348, 6, 416, -1286, 176, "AAATRIGGER")], angles="0 -1 0", speed=1100)
point("env_sprite", 384, -1316, 40, model="sprites/prairie_steam.spr", rendermode=5, renderamt=160, rendercolor="255 255 255",
      framerate=12, scale=2.5, spawnflags=1)
# rotating ceiling fans over the LAN hall
for fx in (-320, 320):
    c3 = (fx, 16, -110)
    W.append(box(fx - 3, 13, -104, fx + 3, 19, HC, "METAL"))
    blades = [box(fx - 112, 8, -112, fx + 112, 24, -110, "FANBLADE"), box(fx - 8, 16 - 104, -112, fx + 8, 16 + 104, -110, "FANBLADE"),
              box(fx - 10, 6, -116, fx + 10, 26, -104, "METAL"), box(fx - 2, 14, -112, fx + 2, 18, -108, "ORIGIN")]
    ent("func_rotating", blades, speed=80 if fx < 0 else -80, spawnflags=1 | 64, fanfriction=20)
# chargers: health in the BattleAxe tunnel, armour in the Crescent tunnel
ent("func_healthcharger", [box(-1088, -420, -150, -1084, -372, -100, "METAL", e="CHARGER_H")])
ent("func_recharge", [box(1084, -420, -150, 1088, -372, -100, "METAL", w="CHARGER_A")])

# ------------------------------------------------------------------ weather + wind
def clear_of_buildings(x, y, m):
    for h in HOUSES:
        fx0, fy0, fx1, fy1 = footprint(h)
        if fx0 - m < x < fx1 + m and fy0 - m < y < fy1 + m: return False
    for (x0, y0, x1, y1) in GARAGES:
        if x0 - m < x < x1 + m and y0 - m < y < y1 + m: return False
    return XW + 100 < x < XE - 100 and YS + 100 < y < YN - 100
# Continuous snowfall: a crossing grid of see-through (additive) animated sheets over the whole outdoor area,
# so snow is everywhere instead of in sprite patches. Over buildings a sheet starts at the roof, so it never
# snows indoors. Sheets are func_illusionary: no collision, no effect on the BSP.
SNOW_GAP, SNOW_TOP, SNOW_BOT = 256, 800, -16
SNOWTEX = {"name": "+0SNOWFALL", "su": 4, "sv": 4}
ROOFS = [footprint(h) for h in HOUSES] + GARAGES
def snow_spans(fixed, lo, hi, along_x):
    """Split the line (x = fixed if not along_x, else y = fixed) from lo..hi into (a, b, z_bottom) runs."""
    cuts = {lo, hi}
    for (x0, y0, x1, y1) in ROOFS:
        c0, c1, f0, f1 = (x0, x1, y0, y1) if along_x else (y0, y1, x0, x1)
        if f0 - 8 < fixed < f1 + 8:
            cuts |= {min(max(c0 - 8, lo), hi), min(max(c1 + 8, lo), hi)}
    cuts = sorted(cuts); out = []
    for a, b in zip(cuts, cuts[1:]):
        if b - a < 8: continue
        m = (a + b) / 2; px, py = (m, fixed) if along_x else (fixed, m)
        over = any(x0 - 8 < px < x1 + 8 and y0 - 8 < py < y1 + 8 for (x0, y0, x1, y1) in ROOFS)
        z0 = ROOF + 2 if over else SNOW_BOT
        if out and out[-1][2] == z0: out[-1] = (out[-1][0], b, z0)
        else: out.append((a, b, z0))
    return out
SNOW = []
for x in range(XW + SNOW_GAP // 2, XE, SNOW_GAP):
    for a, b, z0 in snow_spans(x, YS, YN, False):
        SNOW.append(box(x - 1, a, z0, x + 1, b, SNOW_TOP, SNOWTEX))
for y in range(YS + SNOW_GAP // 2, YN, SNOW_GAP):
    for a, b, z0 in snow_spans(y, XW, XE, True):
        SNOW.append(box(a, y - 1, z0, b, y + 1, SNOW_TOP, SNOWTEX))
for part in chunks(SNOW, 512):     # falling-snow sheets, in chunks so each has a small bbox
    bent("func_illusionary", part, rendermode=5, renderamt=200)
snow_n = len(SNOW)
for i, (x, y, fr) in enumerate([(-2850, 0, 7), (-2650, -160, 9), (-2950, 200, 11), (-2500, 120, 8)]):
    point("env_sprite", x, y, 380, model="sprites/prairie_burst.spr", rendermode=5, renderamt=210, rendercolor="255 255 255",
          framerate=fr, scale=6, spawnflags=1)
for (x, y) in [(-1700, 0), (-2400, 0), (-1000, 900), (0, 880), (1000, 900), (-1000, -900), (0, -880), (1000, -900),
               (1700, 600), (1700, -600), (-1300, 1300), (1300, 1300)]:
    point("ambient_generic", x, y, 120, message="justincon/wind_loop.wav", health=8, spawnflags=4, pitch=100, pitchstart=100)
# night sky fill: blue-grey storm light from above
for (x, y) in [(-2500, 0), (-1200, 900), (-1200, -900), (300, 1000), (300, -1000), (1500, 0), (0, 0)]:
    light(x, y, 950, (140, 155, 185), 1700)

# ------------------------------------------------------------------ weapons + spawns
WEP = ["weapon_shotgun", "weapon_9mmAR", "weapon_357", "weapon_crossbow", "weapon_rpg", "weapon_gauss", "weapon_egon",
       "weapon_hornetgun", "weapon_handgrenade", "weapon_satchel", "weapon_snark", "weapon_tripmine"]
AMM = ["ammo_buckshot", "ammo_9mmAR", "ammo_357", "ammo_crossbow", "ammo_rpgclip", "ammo_gaussclip", "ammo_ARgrenades",
       "item_battery", "item_healthkit"]
wi = [0]
def W_(x, y, z): item(WEP[wi[0] % len(WEP)], x, y, z); wi[0] += 1
def A_(x, y, z): item(AMM[wi[0] % len(AMM)], x, y, z); wi[0] += 1
for (px, py) in PODS:                                  # hall: between pods
    W_(px, py + (128 if py > 0 else -128) * 0 + 128 * (1 if py < 0 else -1), HF + 10)
for (x, y) in [(-650, 0), (650, 0), (0, 10), (-320, 10), (320, 10), (0, -220)]: A_(x, y, HF + 10)
item("weapon_gauss", 0, -330, HF + 42); item("weapon_egon", 0, 330, HF + 10)
for (x, y) in TL[::2]: W_(x, y + (20 if y > 0 else -20) * 0, BF + 10)
for h in HOUSES:
    x, y = L2W(h, -120, -160); W_(x, y, 10)
    x, y = L2W(h, 100, -100); A_(x, y, BF + 10)
for (x, y) in [(-2000, 0), (-1400, -300), (0, 620), (0, -620), (-1100, 0), (500, 0), (-500, 0), (1150, 0)]: W_(x, y, 10)
# roof weapons out on the bridges and open roofs, away from the roof spawns (strong weapons in risky spots)
for (x, y) in [(-768, 1056), (-384, 1150), (768, -1056), (-1152, -1150), (1440, 352), (1440, 576)]: W_(x, y, ROOF + 10)
item("weapon_crossbow", 384, 1200, ROOF + 10); item("weapon_rpg", -384, -1200, ROOF + 10)
for h in HOUSES:
    x, y = L2W(h, 140, -60); spawn(x, y, 0, 0)          # kitchen side, across the room from the house's weapon
for (x, y) in [(-700, 576), (300, 576), (-1024, -200), (1024, 200), (-300, -576), (250, -576)]: spawn(x, y, BF, 0)
for (x, y) in [(-300, -240), (650, 200), (-720, -160), (720, 160), (-560, 330), (-650, 300)]: spawn(x, y, HF, 0)
for (x, y) in [(-1700, 60), (-2400, 60), (-760, -120), (600, -300), (1600, -100)]: spawn(x, y, 0, 0)
for (x, y) in [(-1152, 1150), (1152, -1150), (1504, -100)]: spawn(x, y, ROOF, 0)

for cls, x, y, z in ITEMS:
    point(cls, round(x, 1), round(y, 1), z, angles="0 %d 0" % (int(abs(x * 3 + y)) % 8 * 45))
for x, y, z, yaw in SPAWNS:
    point("info_player_deathmatch", round(x, 1), round(y, 1), z, angles="0 %d 0" % (int(math.degrees(math.atan2(-y, -x))) % 360))
for part in chunks(DECOR, 512):
    bent("func_illusionary", part)

STRUCT, DETAIL = split_detail(W)    # small brushes -> func_detail, so full VIS stays fast
world = ('{\n"classname" "worldspawn"\n"mapversion" "220"\n"wad" "%s.wad"\n"skyname" "%s"\n'
         '"message" "Prairie Loop Winter: JustinCon After Hours"\n' % (NAME, NAME) + "".join(STRUCT) + "}\n")
with open(os.path.join(OUT, NAME + ".map"), "w", newline="\r\n") as f:
    f.write(world + detail_entity(DETAIL) + "".join(BENTS) + "".join(ENTS)
            + texlights(os.path.join(OUT, NAME + ".wad"), A.EMISSIVE))
# ---- generated assets in the game's valve/ layout (sky, sprites, sounds)
VALVE = os.path.join(OUT, "valve")
for sub in ("gfx/env", "sprites", "sound/justincon"):
    os.makedirs(os.path.join(VALVE, sub), exist_ok=True)
A.write_skybox(os.path.join(VALVE, "gfx", "env", NAME))
A.write_spr(os.path.join(VALVE, "sprites", "prairie_burst.spr"), A.burst_frames(), A.SNOW_PAL)
A.write_spr(os.path.join(VALVE, "sprites", "prairie_steam.spr"), A.steam_frames(), A.SNOW_PAL)
A.make_sounds(os.path.join(VALVE, "sound", "justincon"))
A.make_alarm(os.path.join(VALVE, "sound", "justincon", "alarm.wav"))
print("world brushes %d (detail %d, earth %d)  decor %d  brush ents %d  items %d  spawns %d  snow sheets %d  point ents %d" % (
    len(STRUCT), len(DETAIL), EARTH_N, len(DECOR), len(BENTS), len(ITEMS), len(SPAWNS), snow_n, len(ENTS)))
