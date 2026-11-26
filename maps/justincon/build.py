#!/usr/bin/env python3
"""Justin Con / Bunker Con / Respawn Con: three suburban houses at night, each with a basement
LAN party, joined by tunnels. A giant wooden effigy burns in the shared back yard."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))     # repo root, for tools.*
sys.path.insert(0, str(Path(__file__).resolve().parent))         # this map's own modules
from tools.paths import build_dir
import math, os, sys, json
import numpy as np
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import tools.assets.library as textures
from tools.assets import skyfire

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = str(build_dir("justincon"))
TEX_SIZES = textures.write_wad(os.path.join(OUT, "justincon.wad"))

SCALE = {"GRASS": 2, "ASPHALT": 2, "CONCRETE": 1, "SIDEWALK": 1, "HEDGE": 1, "DIRT": 1.5,
         "CARPET": 1, "SHINGLE": 1, "NEIGHBOR": 1.5}

# ------------------------------------------------------------------ brush helpers
Z = np.array([0, 0, 1.0])

def axes(n):
    if abs(n[2]) > 0.7:
        return np.array([1.0, 0, 0]), np.array([0, -1.0, 0])
    u = np.cross(Z, n); u /= np.linalg.norm(u)
    return u, np.cross(u, n)

def face_line(n, d, t):
    t = t if isinstance(t, dict) else {"name": t}
    n = np.array(n, float); n /= np.linalg.norm(n)
    e1 = np.cross(n, [0.0, 0.0, 1.0]) if abs(n[2]) < 0.9 else np.cross(n, [1.0, 0.0, 0.0])
    e1 /= np.linalg.norm(e1); e2 = np.cross(n, e1)
    p1 = n * d
    p0, p2 = p1 + e1 * 256, p1 + e2 * 256
    u, v = axes(n)
    sc = SCALE.get(t["name"], 1)
    su, sv = t.get("su", sc), t.get("sv", sc)
    f = lambda p: " ".join("%.6f" % x for x in p)
    return "( %s ) ( %s ) ( %s ) %s [ %s %.4f ] [ %s %.4f ] 0 %g %g" % (
        f(p0), f(p1), f(p2), t["name"], f(u), t.get("ou", 0), f(v), t.get("ov", 0), su, sv)

def brush(planes):
    return "{\n" + "\n".join(face_line(n, d, t) for n, d, t in planes) + "\n}\n"

def box(x0, y0, z0, x1, y1, z1, tex, top=None, bottom=None, n=None, s=None, e=None, w=None):
    assert x1 > x0 and y1 > y0 and z1 > z0, (x0, y0, z0, x1, y1, z1)
    return brush([((0, 0, 1), z1, top or tex), ((0, 0, -1), -z0, bottom or tex),
                  ((0, 1, 0), y1, n or tex), ((0, -1, 0), -y0, s or tex),
                  ((1, 0, 0), x1, e or tex), ((-1, 0, 0), -x0, w or tex)])

def gable_x(x0, x1, y0, y1, z0, z1, slope_tex, end_tex, bottom_tex):
    yc = (y0 + y1) / 2
    hn = np.array([0, -(z1 - z0), (yc - y0)], float); hn /= np.linalg.norm(hn)
    sn = np.array([0, (z1 - z0), (y1 - yc)], float); sn /= np.linalg.norm(sn)
    return brush([(hn, hn @ np.array([0, y0, z0]), slope_tex), (sn, sn @ np.array([0, y1, z0]), slope_tex),
                  ((0, 0, -1), -z0, bottom_tex), ((1, 0, 0), x1, end_tex), ((-1, 0, 0), -x0, end_tex)])

def beam(p0, p1, th, tex):
    """Square beam from p0 to p1 (in a vertical plane containing the y axis normal)."""
    p0, p1 = np.array(p0, float), np.array(p1, float)
    a = p1 - p0; L = np.linalg.norm(a); a /= L
    side = np.cross(a, [0, 1.0, 0]); side /= np.linalg.norm(side)
    up = np.cross(side, a)
    h = th / 2
    pl = [(a, a @ p1, tex), (-a, -(a @ p0), tex)]
    for v in (side, -side, up, -up):
        pl.append((v, v @ p0 + h, tex))
    return brush(pl)

def octa(c, rx, rz, tex):
    c = np.array(c, float); pl = []
    for sx in (1, -1):
        for sy in (1, -1):
            for sz in (1, -1):
                n = np.array([sx / rx, sy / rx, sz / rz]); nn = np.linalg.norm(n)
                pl.append((n / nn, (n @ c + 1) / nn, tex))
    return brush(pl)

def sign(name, x0, y0, z0, x1, y1, z1, facing, back="METAL"):
    nrm = {"n": (0, 1, 0), "s": (0, -1, 0), "e": (1, 0, 0), "w": (-1, 0, 0)}[facing]
    u, v = axes(np.array(nrm, float))
    corners = [np.array([x, y, z]) for x in (x0, x1) for y in (y0, y1) for z in (z0, z1)]
    us = [c @ u for c in corners]
    tw, th = TEX_SIZES[name]
    su = (max(us) - min(us)) / tw; sv = (z1 - z0) / th
    t = {"name": name, "su": su, "sv": sv, "ou": -min(us) / su, "ov": z1 / sv}
    return box(x0, y0, z0, x1, y1, z1, back, **{facing: t})

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

W, ENTS, BRUSH_ENTS, ITEMS, SPAWNS = [], [], [], [], []

def point(cls, x, y, z, **kv):
    s = '{\n"classname" "%s"\n"origin" "%g %g %g"\n' % (cls, x, y, z)
    for k, v in kv.items(): s += '"%s" "%s"\n' % (k, v)
    ENTS.append(s + "}\n")

def light(x, y, z, rgb, i):
    point("light", x, y, z, _light="%d %d %d %d" % (*rgb, i))

WEAPONS = ["weapon_shotgun", "weapon_9mmAR", "weapon_rpg", "weapon_gauss", "weapon_egon",
           "weapon_crossbow", "weapon_357", "weapon_hornetgun", "weapon_handgrenade",
           "weapon_satchel", "weapon_tripmine", "weapon_snark", "weapon_9mmhandgun", "weapon_crowbar"]
AMMO = ["ammo_buckshot", "ammo_9mmAR", "ammo_rpgclip", "ammo_gaussclip", "ammo_crossbow",
        "ammo_357", "ammo_ARgrenades", "ammo_9mmbox", "item_battery", "item_healthkit"]
_wi, _ai = [0], [0]
def weapon(x, y, z):
    ITEMS.append((WEAPONS[_wi[0] % len(WEAPONS)], x, y, z)); _wi[0] += 1
def ammo(x, y, z):
    ITEMS.append((AMMO[_ai[0] % len(AMMO)], x, y, z)); _ai[0] += 1

# ------------------------------------------------------------------ world constants
SX, SY = 2700, 1600           # shell half extents
SKYTOP, BOT = 960, -208
HX, HY, WT = 448, 320, 16
BF = -176                     # basement floor
BT = BF + 36                  # LAN table height
IX0, IX1, IY0, IY1 = -HX + WT, HX - WT, -HY + WT, HY - WT
CH = (-304, -192); CH_END = 560
TD = (-296, -212)             # tunnel door span (y)
TUN_TOP = BF + 112
HOUSES = [
    dict(cx=-1250, siding="SIDING_G", banner="BANNER_B", poster="POSTER_B", lawn="LAWN_B", reg="REG_B",
         tv="BRACKET", layout="u", doors="e", to_e="TS_JUSTIN", to_w=None,
         glow=[(60, 255, 100), (200, 255, 210), (60, 255, 100), (120, 200, 255)]),
    dict(cx=0, siding="SIDING", banner="BANNER", poster="POSTER", lawn="LAWNSIGN", reg="REGSIGN",
         tv="SCREENTV", layout="rows", doors="we", to_e="TS_RESPAWN", to_w="TS_BUNKER",
         glow=[(255, 60, 200), (60, 200, 255), (120, 255, 120), (255, 140, 40)]),
    dict(cx=1250, siding="SIDING_Y", banner="BANNER_R", poster="POSTER_R", lawn="LAWN_R", reg="REG_R",
         tv="SCREEN_R1", layout="pods", doors="w", to_e=None, to_w="TS_JUSTIN",
         glow=[(255, 140, 30), (0, 220, 200), (255, 140, 30), (0, 220, 200)]),
]

# ------------------------------------------------------------------ sky shell
W.append(box(-SX - 16, -SY - 16, SKYTOP, SX + 16, SY + 16, SKYTOP + 16, "sky"))
W.append(box(-SX - 16, -SY - 16, BOT - 16, SX + 16, SY + 16, BOT, "sky"))
W.append(box(SX, -SY - 16, BOT, SX + 16, SY + 16, SKYTOP, "sky"))
W.append(box(-SX - 16, -SY - 16, BOT, -SX, SY + 16, SKYTOP, "sky"))
W.append(box(-SX, SY, BOT, SX, SY + 16, SKYTOP, "sky"))
W.append(box(-SX, -SY - 16, BOT, SX, -SY, SKYTOP, "sky"))

# ------------------------------------------------------------------ ground with holes
HOLES = []   # (x0, y0, x1, y1, zl, zh, floor_tex, ceil_tex)
for h in HOUSES:
    c = h["cx"]
    HOLES.append((c - HX, -HY, c + HX, HY, BOT, 0, None, None))
    HOLES.append((c + CH[0], HY, c + CH[1], CH_END, BF, 0, "CONCRETE", None))
for a, b in zip(HOUSES, HOUSES[1:]):
    HOLES.append((a["cx"] + HX, TD[0], b["cx"] - HX, TD[1], BF, TUN_TOP, "DIRT", "DIRT"))
xs = sorted({-SX, SX} | {v for hh in HOLES for v in (hh[0], hh[2])})
ys = sorted({-SY, SY, -1000} | {v for hh in HOLES for v in (hh[1], hh[3])})
for xa, xb in zip(xs, xs[1:]):
    for ya, yb in zip(ys, ys[1:]):
        hole = next((hh for hh in HOLES if hh[0] <= xa and xb <= hh[2] and hh[1] <= ya and yb <= hh[3]), None)
        top = "ASPHALT" if yb <= -1000 else "GRASS"
        if hole is None:
            W.append(box(xa, ya, BOT, xb, yb, 0, "DIRT", top=top))
        else:
            _, _, _, _, zl, zh, ft, ct = hole
            if zl > BOT: W.append(box(xa, ya, BOT, xb, yb, zl, "DIRT", top=ft))
            if zh < 0: W.append(box(xa, ya, zh, xb, yb, 0, "DIRT", top=top, bottom=ct))

# ------------------------------------------------------------------ street + neighbourhood
W.append(box(-SX, -1000, 0, SX, -936, 8, "SIDEWALK"))
for x in range(-2500, 2500, 200):
    W.append(box(x, -1302, 0, x + 100, -1298, 1, "STREETLINE"))
W.append(box(-SX, -1440, 0, -SX + 120, -1000, 96, "HEDGE"))
W.append(box(SX - 120, -1440, 0, SX, -1000, 96, "HEDGE"))
W.append(box(-SX, SY - 40, 0, SX, SY - 32, 96, "WOODFENCE"))
W.append(box(-SX + 32, -936, 0, -SX + 40, SY - 40, 96, "WOODFENCE"))
W.append(box(SX - 40, -936, 0, SX - 32, SY - 40, 96, "WOODFENCE"))
for x0 in range(-2600, 2150, 650):
    x1 = x0 + 500
    W.append(box(x0, -SY, 0, x1, -1440, 224, "NEIGHBOR"))
    W.append(gable_x(x0 - 16, x1 + 16, -SY, -1424, 224, 360, "SHINGLE", "NEIGHBOR", "TRIM"))
    for wx in (x0 + 60, x0 + 380):
        for wz in (40, 140):
            W.append(box(wx, -1440, wz, wx + 64, -1436, wz + 64, "TRIM", n="NWINDOW"))
    W.append(box(x0 + 220, -1440, 0, x0 + 280, -1436, 100, "TRIM", n="BRICK"))
    light(x0 + 250, -1400, 110, (255, 200, 120), 90)
for x in (-2000, -1000, 0, 1000, 2000):
    W.append(box(x - 4, -976, 8, x + 4, -968, 200, "METAL"))
    W.append(box(x - 4, -1000, 196, x + 4, -968, 204, "METAL"))
    W.append(box(x - 10, -1020, 188, x + 10, -996, 196, "METAL", bottom="LAMP"))
    light(x, -1008, 176, (255, 190, 110), 320)

def car(x, y, paint, along_x=True):
    L, Wd = 200, 84
    if along_x:
        W.append(box(x - L / 2, y - Wd / 2, 14, x + L / 2, y + Wd / 2, 46, paint))
        W.append(box(x - 50, y - Wd / 2 + 6, 46, x + 50, y + Wd / 2 - 6, 76, "CARGLASS", top=paint))
        for dx in (-62, 62):
            for dy in (-Wd / 2, Wd / 2 - 12):
                W.append(box(x + dx - 14, y + dy - 2, 0, x + dx + 14, y + dy + 14, 26, "TIRE"))
    else:
        W.append(box(x - Wd / 2, y - L / 2, 14, x + Wd / 2, y + L / 2, 46, paint))
        W.append(box(x - Wd / 2 + 6, y - 50, 46, x + Wd / 2 - 6, y + 50, 76, "CARGLASS", top=paint))
        for dy in (-62, 62):
            for dx in (-Wd / 2, Wd / 2 - 12):
                W.append(box(x + dx - 2, y + dy - 14, 0, x + dx + 14, y + dy + 14, 26, "TIRE"))
CARS = [(-2150, -1100, "CARBLUE"), (-1700, -1100, "CARRED"), (-700, -1100, "CARRED"), (-350, -1100, "CARBLUE"),
        (650, -1100, "CARSILVER"), (1600, -1100, "CARBLUE"), (2050, -1100, "CARSILVER")]
for x, y, p in CARS: car(x, y, p)

# ------------------------------------------------------------------ houses
def house(h):
    c = h["cx"]
    B = lambda x0, y0, z0, x1, y1, z1, tex, **kw: W.append(box(c + x0, y0, z0, c + x1, y1, z1, tex, **kw))
    Sg = lambda name, x0, y0, z0, x1, y1, z1, facing, **kw: W.append(sign(name, c + x0, y0, z0, c + x1, y1, z1, facing, **kw))
    L = lambda x, y, z, rgb, i: light(c + x, y, z, rgb, i)
    Wp = lambda x, y, z: weapon(c + x, y, z)
    Am = lambda x, y, z: ammo(c + x, y, z)
    It = lambda cls, x, y, z: ITEMS.append((cls, c + x, y, z))
    Sp = lambda x, y, z: SPAWNS.append((c + x, y, z + 37))
    sh = lambda lst: [box(c + 0, 0, 0, 1, 1, 1, "X")] and None

    # basement floor slab + foundation (basement walls) with bulkhead + tunnel doors
    B(-HX, -HY, BOT, HX, HY, BF, "CONCRETE", top="CARPET")
    B(-HX, -HY, BF, HX, -HY + WT, 0, "CONCRETE", n="BASEWALL")
    W.extend(wall_x(c - HX, c + HX, HY - WT, HY, BF, 0, "CONCRETE", [(c + CH[0], c + CH[1], BF, BF + 112)], s="BASEWALL"))
    door = [(TD[0], TD[1], BF, TUN_TOP)]
    W.extend(wall_y(-HY + WT, HY - WT, c - HX, c - HX + WT, BF, 0, "CONCRETE", door if "w" in h["doors"] else [], e="BASEWALL"))
    W.extend(wall_y(-HY + WT, HY - WT, c + HX - WT, c + HX, BF, 0, "CONCRETE", door if "e" in h["doors"] else [], w="BASEWALL"))
    if h["to_w"]: Sg(h["to_w"], IX0, TD[0], TUN_TOP + 4, IX0 + 3, TD[1], TUN_TOP + 28, "e")
    if h["to_e"]: Sg(h["to_e"], IX1 - 3, TD[0], TUN_TOP + 4, IX1, TD[1], TUN_TOP + 28, "w")
    # ground floor slab with stair opening, stairs, rails
    SX0, SX1, SY0 = 104, 368, 208
    B(IX0, IY0, -16, SX0, IY1, 0, "WOODFLOOR", bottom="CEILTILE")
    B(SX1, IY0, -16, IX1, IY1, 0, "WOODFLOOR", bottom="CEILTILE")
    B(SX0, IY0, -16, SX1, SY0, 0, "WOODFLOOR", bottom="CEILTILE")
    for k in range(1, 11):
        B(SX1 - 24 * k, SY0, BF, SX1 - 24 * (k - 1), IY1, -16 * k, "BASEWALL", top="STEP")
    B(SX0, SY0 - 4, 0, SX1, SY0, 36, "TRIM")
    B(SX0 - 4, SY0, 0, SX0, IY1, 36, "TRIM")
    # bulkhead steps
    for k in range(1, 11):
        B(CH[0], HY + 24 * (k - 1), BF, CH[1], HY + 24 * k, BF + 16 * k, "CONCRETE")
    L(-248, 440, 60, (255, 200, 140), 90)
    # exterior walls + glass
    GH = 128
    win = lambda a, b: (c + a, c + b, 40, 104)
    so = [(c - 48, c + 48, 0, 112), win(-352, -224), win(192, 320)]
    no = [(c - 96, c + 32, 0, 112), win(-352, -224)]
    sd = h["siding"]
    W.extend(wall_x(c - HX, c + HX, -HY, -HY + WT, 0, GH, sd, so, n="WALLPAPER"))
    W.extend(wall_x(c - HX, c + HX, HY - WT, HY, 0, GH, sd, no, s="WALLPAPER"))
    W.extend(wall_y(-HY + WT, HY - WT, c - HX, c - HX + WT, 0, GH, sd, [(-64, 64, 40, 104)], e="WALLPAPER"))
    W.extend(wall_y(-HY + WT, HY - WT, c + HX - WT, c + HX, 0, GH, sd, [(-200, -100, 40, 104)], w="WALLPAPER"))
    g = []
    for a, b in ((-352, -224), (192, 320)): g.append(box(c + a, -HY + 6, 40, c + b, -HY + 10, 104, "GLASS"))
    g.append(box(c - 352, HY - 10, 40, c - 224, HY - 6, 104, "GLASS"))
    g.append(box(c - HX + 6, -64, 40, c - HX + 10, 64, 104, "GLASS"))
    g.append(box(c + HX - 10, -200, 40, c + HX - 6, -100, 104, "GLASS"))
    for gb in g:
        BRUSH_ENTS.append('{\n"classname" "func_breakable"\n"rendermode" "2"\n"renderamt" "90"\n"material" "0"\n"health" "20"\n' + gb + "}\n")
    B(-HX, -HY, GH, HX, HY, GH + 16, "TRIM", bottom="CEILING")
    W.append(gable_x(c - HX - 16, c + HX + 16, -HY - 32, HY + 32, GH + 16, GH + 208, "SHINGLE", sd, "TRIM"))
    B(-300, 100, GH + 16, -236, 164, 400, "BRICK")
    B(-96, -384, 0, 96, -HY, 8, "CONCRETE")
    L(0, -350, 110, (255, 210, 150), 140)
    # ground floor interior
    W.extend(wall_y(IY0, -40, c + 96, c + 112, 0, GH, "WALLPAPER", [(-220, -140, 0, 112)]))
    B(-420, -160, 0, -340, 40, 20, "COUCH"); B(-432, -160, 0, -404, 40, 44, "COUCH")
    B(-420, -176, 0, -340, -160, 30, "COUCH"); B(-420, 40, 0, -340, 56, 30, "COUCH")
    B(-280, -100, 0, -200, -20, 18, "STEP")
    B(40, -110, 0, 90, -10, 24, "STEP")
    W.append(sign(h["tv"], c + 64, -104, 24, c + 80, -16, 74, "w", back="TVBLACK"))
    L(-200, -60, 92, (255, 220, 170), 110); L(20, -60, 60, (120, 140, 255), 70)
    B(150, -250, 0, 330, -214, 34, "TABLE")
    Sg(h["reg"], 176, IY0, 70, 304, IY0 + 4, 102, "n")
    L(260, -150, 92, (255, 230, 190), 100)
    B(400, -40, 0, IX1, 120, 36, "TRIM", top="COUNTER")
    B(180, 60, 0, 300, 140, 36, "TRIM", top="COUNTER")
    B(380, IY0, 0, IX1, -260, 84, "FRIDGE")
    L(260, 100, 92, (255, 240, 210), 110)
    # yard bits: lawn sign, mailbox, driveway
    B(-326, -702, 0, -322, -698, 40, "METAL")
    Sg(h["lawn"], -404, -704, 40, -244, -700, 120, "s", back="TRIM")
    B(-124, -962, 8, -118, -956, 48, "METAL"); B(-134, -968, 48, -108, -950, 62, "METAL")
    B(200, -936, 0, 400, -HY, 2, "CONCRETE")
    car(c + 300, -720, ["CARRED", "CARSILVER", "CARBLUE"][HOUSES.index(h)], along_x=False)

    # ---- basement common
    banner_z = (-128, -32) if h["layout"] != "u" else (-112, -32)
    Sg(h["banner"], IX0, -192, banner_z[0], IX0 + 4, 192, banner_z[1], "e")
    Sg(h["poster"], -160, IY1 - 4, -136, 0, IY1, -36, "s")
    B(-420, 236, BF, -376, IY1, -80, "METAL", s="SERVERRACK")
    B(160, IY0, BT - 4, 340, IY0 + 40, BT, "TABLE")
    B(160, IY0, BF, 164, IY0 + 40, BT - 4, "METAL"); B(336, IY0, BF, 340, IY0 + 40, BT - 4, "METAL")
    for k in range(4):
        B(170, IY0 + 6, BT + 3 * k, 198, IY0 + 34, BT + 3 * k + 3, "PIZZABOX")
        B(204, IY0 + 6, BT + 3 * k, 232, IY0 + 34, BT + 3 * k + 3, "PIZZABOX")
    for k in range(6):
        B(250 + 12 * k, IY0 + 8, BT, 258 + 12 * k, IY0 + 16, BT + 12, "ENERGY" if k % 2 else "SODA")
    for x in (180, 220, 300, 330):
        It("weapon_handgrenade", x, IY0 + 32, BT + 20)
    glow = h["glow"]
    L(240, 0, -84, glow[0], 110); L(240, -240, -84, (255, 200, 120), 80)
    L(-380, 0, -90, glow[1], 70); L(250, 256, -84, (255, 240, 220), 70)
    # ground floor items + spawns
    for x, y in [(-380, -100), (-380, 20), (-240, -60)]: Wp(x, y, 32)
    for x in (170, 250, 320): Wp(x, -232, 44)
    for x, y in [(200, 80), (280, 120)]: Wp(x, y, 46)
    for y in (-20, 60): Am(416, y, 46)
    for x in (-300, -140, 20): Wp(x, -240, 10)
    Wp(-248, HY + 24 * 5 - 12, BF + 16 * 5 + 10)
    for x, y in [(-250, 150), (-120, -220), (240, -60), (340, 170)]: Sp(x, y, 0)

    lay = h["layout"]
    if lay in ("rows", "u"):
        for zs in (BF + 56, BF + 96):
            B(IX1 - 16, -150, zs - 4, IX1, 150, zs, "METAL")
            for y in range(-130, 150, 70): Wp(IX1 - 8, y, zs + 10)

    if lay == "rows":            # Justin Con: classic rows of folding tables
        ROWS = [(-250, -210), (-120, -80), (10, 50), (140, 180)]
        SEATS = [-330, -270, -210, -150, -90, -30, 30]
        screens = ["SCREEN1", "SCREEN2", "SCREEN3"]
        for r, (y0, y1) in enumerate(ROWS):
            B(-360, y0, BT - 4, 60, y1, BT, "TABLE")
            B(-360, y0, BF, -356, y1, BT - 4, "METAL"); B(56, y0, BF, 60, y1, BT - 4, "METAL")
            for i, s_ in enumerate(SEATS):
                B(s_ - 14, y1 - 10, BT, s_ + 14, y1 - 6, BT + 24, "TVBLACK", s=screens[(r + i) % 3])
                B(s_ + 17, y1 - 30, BT, s_ + 25, y1 - 4, BT + 34, "PCCASE")
                B(s_ - 10, y0 - 34, BF, s_ + 10, y0 - 14, BF + 18, "COUCH")
                B(s_ - 10, y0 - 38, BF, s_ + 10, y0 - 34, BF + 46, "COUCH")
                if (r + i) % 2 == 0:
                    B(s_ - 4, y0 + 22, BT, s_, y0 + 26, BT + 8, "ENERGY" if i % 3 else "SODA")
                Wp(s_ + 30 if s_ < 30 else s_ - 2, y0 + 12, BT + 10)
            L(-150, (y0 + y1) / 2 - 20, -84, glow[r], 75)
        for i in range(4):
            for j in range(3):
                (Wp if (i + j) % 3 else Am)(170 + i * 40, -150 + j * 40, BF + 10)
        for x, y in [(-300, -170), (-60, -170), (-300, -40), (-60, -40), (-300, 90), (240, 60)]: Sp(x, y, BF)

    elif lay == "u":             # Bunker Con: tournament U, dual monitors, bracket screen
        for x0, x1 in ((-300, 100),):
            B(x0, IY0, BT - 4, x1, IY0 + 40, BT, "TABLE")
            B(x0, IY0, BF, x0 + 4, IY0 + 40, BT - 4, "METAL"); B(x1 - 4, IY0, BF, x1, IY0 + 40, BT - 4, "METAL")
        for i, s_ in enumerate((-260, -200, -140, -80, -20, 40)):
            B(s_ - 27, IY0 + 4, BT, s_ - 3, IY0 + 8, BT + 20, "TVBLACK", n="SCREEN1" if i % 2 else "SCREEN3")
            B(s_ + 3, IY0 + 4, BT, s_ + 27, IY0 + 8, BT + 20, "TVBLACK", n="SCREEN2")
            B(s_ - 10, IY0 + 54, BF, s_ + 10, IY0 + 74, BF + 18, "COUCH")
            B(s_ - 10, IY0 + 74, BF, s_ + 10, IY0 + 78, BF + 46, "COUCH")
            Wp(s_ + 30, IY0 + 30, BT + 10)
        B(IX0, -180, BT - 4, IX0 + 40, 180, BT, "TABLE")
        B(IX0, -180, BF, IX0 + 40, -176, BT - 4, "METAL"); B(IX0, 176, BF, IX0 + 40, 180, BT - 4, "METAL")
        for i, s_ in enumerate((-150, -90, -30, 30, 90, 150)):
            B(IX0 + 4, s_ - 27, BT, IX0 + 8, s_ - 3, BT + 20, "TVBLACK", e="SCREEN2" if i % 2 else "SCREEN1")
            B(IX0 + 4, s_ + 3, BT, IX0 + 8, s_ + 27, BT + 20, "TVBLACK", e="SCREEN3")
            B(IX0 + 54, s_ - 10, BF, IX0 + 74, s_ + 10, BF + 18, "COUCH")
            B(IX0 + 74, s_ - 10, BF, IX0 + 78, s_ + 10, BF + 46, "COUCH")
            Wp(IX0 + 30, s_ + 30 if s_ < 150 else s_ - 30, BT + 10)
        # tournament stage: bracket screen + bench
        B(-120, 60, BF, 120, 72, BF + 20, "METAL")
        W.append(sign("BRACKET", c - 120, 60, BF + 20, c + 120, 64, BF + 140, "s", back="TVBLACK"))
        B(-150, -60, BF, 150, -36, BF + 18, "STEP")
        for x in range(-120, 121, 60): (Wp if x % 120 else Am)(x, -10, BF + 10)
        for x in (-100, 0, 100): Wp(x, 30, BF + 10)
        L(0, -20, -60, glow[0], 120); L(-300, -100, -84, glow[2], 80)
        for x, y in [(-250, -110), (-100, -150), (60, -150), (-250, 110), (0, 160), (250, 60)]: Sp(x, y, BF)

    else:                        # Respawn Con: retro pods of CRTs, arcade cabinets
        for px, py in ((-300, -170), (-300, 90), (-60, -170), (-60, 90)):
            B(px - 48, py - 48, BT - 4, px + 48, py + 48, BT, "TABLE")
            B(px - 4, py - 4, BF, px + 4, py + 4, BT - 4, "METAL")
            crt = [("n", (px - 14, py + 4, px + 14, py + 30)), ("s", (px - 14, py - 30, px + 14, py - 4)),
                   ("e", (px + 4, py - 14, px + 30, py + 14)), ("w", (px - 30, py - 14, px - 4, py + 14))]
            for k, (f, (a0, b0, a1, b1)) in enumerate(crt):
                W.append(box(c + a0, b0, BT, c + a1, b1, BT + 26, "CRT", **{f: "SCREEN_R1" if k % 2 else "SCREEN_R2"}))
            for dx, dy in ((0, 66), (0, -66), (66, 0), (-66, 0)):
                B(px + dx - 10, py + dy - 10, BF, px + dx + 10, py + dy + 10, BF + 18, "COUCH")
            for dx, dy in ((36, 36), (-36, 36), (36, -36), (-36, -36)):
                Wp(px + dx, py + dy, BT + 10)
            L(px, py, -84, glow[(px + py) % 2], 75)
        for y0 in (-110, -25, 60):
            B(IX1 - 40, y0, BF, IX1, y0 + 50, BF + 84, "ARCADE")
            W.append(sign("SCREEN_R1" if y0 % 2 else "SCREEN_R2", c + IX1 - 42, y0 + 6, BF + 46, c + IX1 - 40, y0 + 44, BF + 76, "w", back="ARCADE"))
            Wp(IX1 - 60, y0 + 25, BF + 10)
        for i in range(4):
            for j in range(3):
                (Wp if (i + j) % 2 else Am)(170 + i * 40, -150 + j * 40, BF + 10)
        for x, y in [(-180, -40), (-180, -250), (60, -40), (250, 120), (-380, -100), (200, -100)]: Sp(x, y, BF)

for h in HOUSES:
    house(h)

# ------------------------------------------------------------------ tunnels
for a, b in zip(HOUSES, HOUSES[1:]):
    x0, x1 = a["cx"] + HX, b["cx"] - HX
    for x in range(int(x0) + 48, int(x1) - 20, 96):
        W.append(box(x, TD[0], BF, x + 8, TD[0] + 6, TUN_TOP, "STEP"))
        W.append(box(x, TD[1] - 6, BF, x + 8, TD[1], TUN_TOP, "STEP"))
        W.append(box(x, TD[0], TUN_TOP - 8, x + 8, TD[1], TUN_TOP, "STEP"))
    W.append(box(x0, -244, BF, x1, -240, BF + 2, "CABLEB"))
    W.append(box(x0, -258, BF, x1, -254, BF + 2, "CABLEY"))
    for x in (x0 + 90, (x0 + x1) / 2, x1 - 90):
        W.append(box(x - 6, -258, TUN_TOP - 4, x + 6, -250, TUN_TOP, "METAL", bottom="LAMP"))
        light(x, -254, TUN_TOP - 20, (255, 200, 130), 70)
    for x in (x0 + 60, (x0 + x1) / 2 - 30, (x0 + x1) / 2 + 30, x1 - 60):
        weapon(x, -270, BF + 10)

# ------------------------------------------------------------------ the burning effigy (back yard)
EX, EY = 0, 1150
for k in range(4):
    z0 = 16 * k
    if k % 2 == 0:
        for dy in (-78, 78):
            W.append(box(EX - 110, EY + dy - 14, z0, EX + 110, EY + dy + 14, z0 + 16, "LOGS"))
    else:
        for dx in (-78, 78):
            W.append(box(EX + dx - 14, EY - 110, z0, EX + dx + 14, EY + 110, z0 + 16, "LOGS"))
W.append(box(EX - 64, EY - 64, 0, EX + 64, EY + 64, 64, "CHAR", top="LOGS"))
P = lambda x, z: (EX + x, EY, z)
for sx in (-1, 1):
    W.append(beam(P(sx * 56, 64), P(sx * 18, 262), 20, "CHAR"))          # legs
    W.append(beam(P(sx * 70, 404), P(sx * 150, 560), 18, "CHAR"))        # raised arms
    W.append(beam(P(sx * 150, 560), P(sx * 168, 600), 12, "CHAR"))       # hands
W.append(box(EX - 44, EY - 12, 252, EX + 44, EY + 12, 272, "CHAR"))   # hips
W.append(box(EX - 20, EY - 12, 272, EX + 20, EY + 12, 404, "CHAR"))   # torso
W.append(beam(P(-44, 272), P(0, 380), 10, "CHAR")); W.append(beam(P(44, 272), P(0, 380), 10, "CHAR"))
W.append(box(EX - 80, EY - 12, 398, EX + 80, EY + 12, 416, "CHAR"))   # shoulders
W.append(box(EX - 7, EY - 7, 416, EX + 7, EY + 7, 440, "CHAR"))       # neck
W.append(octa((EX, EY, 486), 40, 50, "CHAR"))                          # head
# fire
FIRE = "sprites/justincon_fire.spr"
def fire(x, y, z, sc):
    point("env_sprite", x, y, z, model=FIRE, rendermode=5, renderamt=255, rendercolor="255 255 255",
          framerate=14 + (int(x + y + z) % 5), scale=sc, spawnflags=1)
for dx, dy in ((-80, -80), (80, -80), (-80, 80), (80, 80), (0, -100), (0, 100), (-100, 0), (100, 0)):
    fire(EX + dx, EY + dy, 40, 2.2)
for sx in (-1, 1):
    fire(EX + sx * 40, EY - 16, 140, 1.4); fire(EX + sx * 40, EY + 16, 140, 1.4)
    fire(EX + sx * 110, EY - 14, 470, 1.0); fire(EX + sx * 160, EY + 14, 560, 0.8)
fire(EX, EY - 20, 290, 1.6); fire(EX, EY + 20, 300, 1.6)
fire(EX, EY - 10, 470, 1.4); fire(EX, EY + 10, 500, 1.2)
light(EX, EY - 110, 110, (255, 130, 40), 900)
light(EX, EY + 110, 110, (255, 130, 40), 900)
light(EX, EY - 60, 420, (255, 150, 60), 800)
light(EX, EY + 60, 420, (255, 150, 60), 800)
point("ambient_generic", EX, EY, 100, message="ambience/burning1.wav", health=10, spawnflags=4, pitch=100, pitchstart=100)
BRUSH_ENTS.append('{\n"classname" "trigger_hurt"\n"dmg" "15"\n"damagetype" "8"\n' +
                  box(EX - 124, EY - 124, 0, EX + 124, EY + 124, 120, "AAATRIGGER") + "}\n")

# moonlight (moon is towards +y in the skybox)
for x, y, z, i in ((-2200, 1300, 900, 1700), (2200, 1300, 900, 1700), (0, 1400, 900, 1500),
                   (-1500, -1300, 800, 900), (1500, -1300, 800, 900), (0, -1300, 800, 700)):
    light(x, y, z, (110, 130, 210), i)

# ------------------------------------------------------------------ yard + street items, yard spawns
EXCL = [(h["cx"] - HX - 48, -HY - 96, h["cx"] + HX + 48, HY + 48) for h in HOUSES]
EXCL += [(h["cx"] + CH[0] - 24, HY, h["cx"] + CH[1] + 24, CH_END + 24) for h in HOUSES]
EXCL += [(h["cx"] - 420, -730, h["cx"] - 230, -680) for h in HOUSES]           # lawn signs
EXCL += [(h["cx"] + 240, -840, h["cx"] + 360, -600) for h in HOUSES]           # driveway cars
EXCL += [(x - 120, y - 60, x + 120, y + 60) for x, y, _ in CARS]
EXCL += [(EX - 170, EY - 170, EX + 170, EY + 170)]
EXCL += [(x - 30, -1010, x + 30, -950) for x in (-2000, -1000, 0, 1000, 2000)]
EXCL += [(h["cx"] - 150, -980, h["cx"] - 90, -940) for h in HOUSES]           # mailboxes
clear = lambda x, y: not any(a <= x <= b and c_ <= y <= d for a, c_, b, d in EXCL)
k = 0
for x in range(-2400, 2401, 400):
    for y in (-1200, -760, 800, 1350):
        if clear(x, y):
            (weapon if k % 3 else ammo)(x, y, 10); k += 1
for x in range(-2250, 2251, 600):
    for y in (0,):
        if clear(x, y): weapon(x, y, 10)
for x in (-170, 170):
    ITEMS.append(("item_longjump", x, -400 if False else -620, 10))
for x, y in [(0, -560), (-1250, -560), (1250, -560), (-2300, 300), (2300, 300), (-600, 800), (600, 800),
             (-1900, 1100), (1900, 1100), (-400, 1400), (400, 1400), (-1000, -1180), (1100, -1180)]:
    assert clear(x, y), (x, y)
    SPAWNS.append((x, y, 37))

for cls, x, y, z in ITEMS:
    point(cls, x, y, z, angles="0 %d 0" % ((abs(int(x) * 7 + int(y) * 3) % 8) * 45))
for x, y, z in SPAWNS:
    point("info_player_deathmatch", x, y, z, angles="0 %d 0" % (int(math.degrees(math.atan2(-y, -x))) % 360))
point("game_player_equip", 0, 0, 300, **{
    "weapon_crowbar": 1, "weapon_9mmhandgun": 1, "weapon_357": 1, "weapon_9mmAR": 1,
    "weapon_shotgun": 1, "weapon_crossbow": 1, "weapon_rpg": 1, "weapon_gauss": 1, "weapon_egon": 1,
    "weapon_hornetgun": 1, "weapon_handgrenade": 1, "weapon_satchel": 1, "weapon_tripmine": 1,
    "weapon_snark": 1, "ammo_9mmclip": 2, "ammo_357": 2, "ammo_ARgrenades": 2, "ammo_buckshot": 3,
    "ammo_crossbow": 2, "ammo_rpgclip": 2, "ammo_gaussclip": 3, "item_battery": 2})

world = ('{\n"classname" "worldspawn"\n"mapversion" "220"\n"wad" "justincon.wad"\n"skyname" "justincon"\n'
         '"message" "Justin Con / Bunker Con / Respawn Con"\n"_emissive" "%s"\n' %
         ",".join("%s:%d" % kv for kv in textures.EMISSIVE.items()) + "".join(W) + "}\n")
with open(os.path.join(OUT, "justincon.map"), "w", newline="\r\n") as f:
    f.write(world + "".join(BRUSH_ENTS) + "".join(ENTS))
VALVE = os.path.join(OUT, "valve")
os.makedirs(os.path.join(VALVE, "gfx", "env"), exist_ok=True); os.makedirs(os.path.join(VALVE, "sprites"), exist_ok=True)
skyfire.write_skybox(os.path.join(VALVE, "gfx", "env", "justincon"))
skyfire.write_sprite(os.path.join(VALVE, "sprites", "justincon_fire.spr"))
print("world brushes:", len(W), "brush ents:", len(BRUSH_ENTS), "items:", len(ITEMS), "spawns:", len(SPAWNS),
      "point ents:", len(ENTS))
