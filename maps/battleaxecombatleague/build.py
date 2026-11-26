#!/usr/bin/env python3
"""BattleAxe Combat League - Fight Night. Indoor arena card for Half-Life DM.
Players spawn in the red / blue corner locker rooms and walk out to the octagon.
Look: enclosed arena (roof, rigging, glass suites), charcoal panels, red/black branding."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))     # repo root, for tools.*
sys.path.insert(0, str(Path(__file__).resolve().parent))         # this map's own modules
from tools.paths import build_dir
from tools.mapkit import chunks, detail_entity, split_detail, texlights
import math, os, sys, json
import numpy as np
from scipy.spatial import ConvexHull
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import assets as fn_textures
from tools.assets import skyfire

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = str(build_dir("battleaxecombatleague"))
TEX = fn_textures.write_wad(os.path.join(OUT, "battleaxecombatleague.wad"))
SCALE = {"ARENAFLOOR": 4, "STANDS": 2, "SEATS": 4, "STADWALL": 4, "CONCRETE": 2, "CROWDTOP": 4, "CEILTILE": 2, "ROOF": 4, "TRUSS": 4, "REDSTRIP": 4, "BLUESTRIP": 4}

# ------------------------------------------------------------------ brush helpers
Z = np.array([0, 0, 1.0])
def axes(n):
    if abs(n[2]) > 0.7: return np.array([1.0, 0, 0]), np.array([0, -1.0, 0])
    u = np.cross(Z, n); u /= np.linalg.norm(u)
    return u, np.cross(u, n)

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
    assert x1 > x0 and y1 > y0 and z1 > z0, (x0, y0, z0, x1, y1, z1)
    return brush([((0, 0, 1), z1, top or tex), ((0, 0, -1), -z0, bottom or tex), ((0, 1, 0), y1, n or tex),
                  ((0, -1, 0), -y0, s or tex), ((1, 0, 0), x1, e or tex), ((-1, 0, 0), -x0, w or tex)])

def area2(poly):
    return 0.5 * sum(poly[i][0] * poly[(i + 1) % len(poly)][1] - poly[(i + 1) % len(poly)][0] * poly[i][1] for i in range(len(poly)))

def prism(poly, z0, z1, side, top=None, bottom=None):
    """Convex CCW 2D polygon extruded. side: texture or f(n2d)->texture."""
    assert z1 > z0 and area2(poly) > 1
    pl = [((0, 0, 1), z1, top or (side if not callable(side) else "STANDS")),
          ((0, 0, -1), -z0, bottom or (side if not callable(side) else "STANDS"))]
    for i in range(len(poly)):
        a, b = np.array(poly[i], float), np.array(poly[(i + 1) % len(poly)], float)
        e = b - a
        if np.linalg.norm(e) < 0.01: continue
        n2 = np.array([e[1], -e[0]]) / np.linalg.norm(e)
        pl.append(((n2[0], n2[1], 0), n2 @ a, side(n2) if callable(side) else side))
    return brush(pl)

def clip2(poly, n, d):
    """Keep part of convex polygon with n.p <= d."""
    out = []
    for i in range(len(poly)):
        p, q = np.array(poly[i]), np.array(poly[(i + 1) % len(poly)])
        dp, dq = n @ p - d, n @ q - d
        if dp <= 1e-6: out.append(tuple(p))
        if (dp < -1e-6 and dq > 1e-6) or (dp > 1e-6 and dq < -1e-6):
            t = dp / (dp - dq); out.append(tuple(p + (q - p) * t))
    return out if len(out) >= 3 and abs(area2(out)) > 1 else None

def hull_brush(pts, tex):
    h = ConvexHull(np.array(pts, float))
    seen, pl = set(), []
    for eq in h.equations:
        n, off = eq[:3], eq[3]
        k = (tuple(np.round(n, 5)), round(-off, 3))
        if k in seen: continue
        seen.add(k); pl.append((n, -off, tex))
    return brush(pl)

def beam(p0, p1, th, tex):
    p0, p1 = np.array(p0, float), np.array(p1, float)
    a = p1 - p0; a /= np.linalg.norm(a)
    hlp = Z if abs(a[2]) < 0.9 else np.array([1.0, 0, 0])
    side = np.cross(a, hlp); side /= np.linalg.norm(side); up = np.cross(side, a)
    pl = [(a, a @ p1, tex), (-a, -(a @ p0), tex)]
    for v in (side, -side, up, -up): pl.append((v, v @ p0 + th / 2, tex))
    return brush(pl)

def fit(name, n, umin, umax, z0, z1):
    u, _ = axes(np.array(n, float))
    tw, th = TEX[name]
    su = (umax - umin) / tw; sv = (z1 - z0) / th
    return {"name": name, "su": su, "sv": sv, "ou": -umin / su, "ov": z1 / sv}

def sign(name, x0, y0, z0, x1, y1, z1, facing, back="METAL"):
    nrm = {"n": (0, 1, 0), "s": (0, -1, 0), "e": (1, 0, 0), "w": (-1, 0, 0)}[facing]
    u, _ = axes(np.array(nrm, float))
    us = [np.array([x, y, 0]) @ u for x in (x0, x1) for y in (y0, y1)]
    return box(x0, y0, z0, x1, y1, z1, back, **{facing: fit(name, nrm, min(us), max(us), z0, z1)})

# ------------------------------------------------------------------ octagon geometry
T8 = math.tan(math.pi / 8)
def octv(a):
    b = a * T8
    return [(a, -b), (a, b), (b, a), (-b, a), (-a, b), (-a, -b), (-b, -a), (b, -a)]
def segn(seg): return np.array([math.cos(seg * math.pi / 4), math.sin(seg * math.pi / 4)])
def onoct(a, seg, t):
    v = octv(a); p, q = np.array(v[seg]), np.array(v[(seg + 1) % 8])
    return p + (q - p) * t
def rq(a0, a1, seg, t0=0.0, t1=1.0):
    return [tuple(onoct(a0, seg, t0)), tuple(onoct(a1, seg, t0)), tuple(onoct(a1, seg, t1)), tuple(onoct(a0, seg, t1))]

S = 2800                 # shell half-size
SKYTOP = 1500            # roof deck height (indoor arena: no sky)
def ring_to_square(a, seg):
    """Piece between octagon apothem a and the square shell, for one segment."""
    rays = [(S, -S * T8), (S, S * T8), (S * T8, S), (-S * T8, S), (-S, S * T8), (-S, -S * T8), (-S * T8, -S), (S * T8, -S)]
    corners = {1: (S, S), 3: (-S, S), 5: (-S, -S), 7: (S, -S)}
    v = octv(a); j = (seg + 1) % 8
    poly = [v[seg], rays[seg]] + ([corners[seg]] if seg in corners else []) + [rays[j], v[j]]
    return poly

W, ENTS, BENTS, ITEMS, SPAWNS, CROWD = [], [], [], [], [], []
def point(cls, x, y, z, **kv):
    s = '{\n"classname" "%s"\n"origin" "%g %g %g"\n' % (cls, x, y, z)
    for k, v in kv.items(): s += '"%s" "%s"\n' % (k, v)
    ENTS.append(s + "}\n")
def light(x, y, z, rgb, i): point("light", x, y, z, _light="%d %d %d %d" % (*rgb, i))
def bent(cls, brushes, **kv):
    s = '{\n"classname" "%s"\n' % cls + "".join('"%s" "%s"\n' % kv_ for kv_ in kv.items())
    BENTS.append(s + "".join(brushes) + "}\n")

# ------------------------------------------------------------------ shell + floor
W.append(box(-S - 16, -S - 16, SKYTOP, S + 16, S + 16, SKYTOP + 16, "STADWALL", bottom="ROOF"))
W.append(box(-S - 16, -S - 16, -48, S + 16, S + 16, -32, "STADWALL"))
W.append(box(S, -S - 16, -32, S + 16, S + 16, SKYTOP, "STADWALL"))
W.append(box(-S - 16, -S - 16, -32, -S, S + 16, SKYTOP, "STADWALL"))
W.append(box(-S, S, -32, S, S + 16, SKYTOP, "STADWALL"))
W.append(box(-S, -S - 16, -32, S, -S, SKYTOP, "STADWALL"))
W.append(box(-S, -S, -32, S, S, 0, "ARENAFLOOR"))

# ------------------------------------------------------------------ lower bowl
A0, DEP, NT = 900, 48, 20           # first tier apothem, tier depth, tier count
lh = lambda k: 64 + 16 * k          # lower tier height (tier 0 = 64-high fascia)
CONC0, CONC1 = A0 + DEP * NT, 2040  # concourse 1860..2040 at lh(19)
CONC_Z = lh(NT - 1)
CROWDS = ["CROWD1", "CROWD2", "CROWD3"]
AISLES = [(0.04, 0.46), (0.54, 0.96)]
TUN_X, ROOM_X = 64, 256            # blue tunnel / blue locker room half widths (seg 2)
ROOM_K = (12, 19)                  # tiers 12..18 hold the blue locker room underneath
TUN_TOP, ROOM_TOP = 128, 176

def tier_side(seg, ribbon=None, zt=None):
    nseg = segn(seg)
    def f(n2):
        if ribbon and n2 @ nseg < -0.99:
            return {"name": ribbon, "su": 2, "sv": 2, "ov": zt / 2}
        return "STANDS"
    return f

def crowd_spec(seed, ztop):
    return {"name": CROWDS[seed % 3], "su": 1.25, "sv": 1.25, "ov": ztop / 1.25, "ou": (seed * 37) % 128}

def seg2_parts(poly, k):
    """Split a seg-2 polygon at the tunnel / room x-planes; returns [(poly, cavity_top or None)]."""
    if k < ROOM_K[0]:
        cuts, cav = [-TUN_X, TUN_X], TUN_TOP
    elif k < ROOM_K[1]:
        cuts, cav = [-ROOM_X, ROOM_X], ROOM_TOP
    else:
        return [(poly, None)]
    xa, xb = cuts
    out = []
    l = clip2(poly, np.array([1.0, 0]), xa)
    r = clip2(poly, np.array([-1.0, 0]), -xb)
    m = clip2(poly, np.array([1.0, 0]), xb); m = m and clip2(m, np.array([-1.0, 0]), -xa)
    if l: out.append((l, None))
    if r: out.append((r, None))
    if m: out.append((m, cav))
    return out

for seg in range(8):
    if seg == 6: continue                      # walkout stage side
    for k in range(NT):
        a0, a1 = A0 + DEP * k, A0 + DEP * (k + 1)
        h = lh(k)
        parts = [(rq(a0, a1, seg), None)] if seg != 2 else seg2_parts(rq(a0, a1, seg), k)
        for poly, cav in parts:
            side = tier_side(seg, "RIBBON1" if seg % 2 == 0 else "RIBBON3", h) if k == 0 else tier_side(seg)
            if cav is None:
                W.append(prism(poly, 0, h, side, top="SEATS"))
            elif h > cav + 16:
                W.append(prism(poly, cav, h, side, top="SEATS",
                               bottom="CEILTILE" if cav == ROOM_TOP else "TUNNELWALL"))
        # crowd
        for i, (t0, t1) in enumerate(AISLES):
            cpoly = rq(a0 + 6, a0 + 42, seg, t0, t1)
            cparts = [(cpoly, None)] if seg != 2 else seg2_parts(cpoly, k)
            for cp, cav in cparts:
                if cav is not None and h <= cav + 16: continue
                sp = crowd_spec(k * 3 + seg + i, h + 40)
                CROWD.append(prism(cp, h, h + 40, sp, top="CROWDTOP"))
    if seg != 2:   # aisle steps up to the fascia-high first tier
        for j in range(3):
            W.append(prism(rq(A0 - 48 + 16 * j, A0 - 32 + 16 * j, seg, 0.46, 0.54), 0, 16 * (j + 1), "STANDS", top="SEATS"))

# concourse (seg 2 keeps it solid; tunnel/room end before it)
for seg in range(8):
    if seg == 6: continue
    W.append(prism(rq(CONC0, CONC1, seg), 0, CONC_Z, "STADWALL", top="CONCRETE"))

# ------------------------------------------------------------------ upper deck + balcony + stairs
B0, B1 = 1980, 2076                 # balcony walkway (overhangs concourse 1980..2040)
BAL_Z = CONC_Z + 16 * 15            # 608: 15 steps of 16 up from the concourse
UND = 560                           # balcony underside
UA0, UNT = B1, 12
uh = lambda k: BAL_Z + 16 * (k + 1)
STAIR_SEGS = (1, 3, 5, 7)
ST0, ST1 = 0.10, 0.32               # stair run along the segment
HOLE_T0 = ST0 + (ST1 - ST0) * 3 / 15
for seg in range(8):
    if seg == 6: continue
    # balcony slab (overhang part) with stair hole
    if seg in STAIR_SEGS:
        W.append(prism(rq(B0, CONC1, seg, 0, HOLE_T0), UND, BAL_Z, "STADWALL", top="CONCRETE"))
        W.append(prism(rq(B0, CONC1, seg, ST1 + 0.01, 1), UND, BAL_Z, "STADWALL", top="CONCRETE"))
        for j in range(15):
            t0 = ST0 + (ST1 - ST0) * j / 15; t1 = ST0 + (ST1 - ST0) * (j + 1) / 15
            W.append(prism(rq(1976, CONC1, seg, t0, t1), CONC_Z, CONC_Z + 16 * (j + 1), "STADWALL", top="STEP"))
        pars = [(0, ST0 - 0.01), (ST1 + 0.02, 0.46), (0.54, 1)]
    else:
        W.append(prism(rq(B0, CONC1, seg), UND, BAL_Z, "STADWALL", top="CONCRETE"))
        pars = [(0, 0.46), (0.54, 1)]
    W.append(prism(rq(CONC1, B1, seg), 0, BAL_Z, "STADWALL", top="CONCRETE"))
    for t0, t1 in pars:   # balcony parapet with LED face
        sp = {"name": "RIBBON2", "su": 1, "sv": 1, "ov": (BAL_Z + 32) / 1}
        W.append(prism(rq(B0, B0 + 6, seg, t0, t1), BAL_Z, BAL_Z + 32,
                       lambda n2, sp=sp, seg=seg: sp if n2 @ segn(seg) < -0.99 else "STADWALL", top="CAGEPAD"))
    # upper tiers + crowd
    for k in range(UNT):
        a0, a1 = UA0 + DEP * k, UA0 + DEP * (k + 1)
        W.append(prism(rq(a0, a1, seg), 0, uh(k), "STANDS", top="SEATS"))
        for i, (t0, t1) in enumerate(AISLES):
            CROWD.append(prism(rq(a0 + 6, a0 + 42, seg, t0, t1), uh(k), uh(k) + 40, crowd_spec(k * 5 + seg + i + 1, uh(k) + 40), top="CROWDTOP"))
UTOP = UA0 + DEP * UNT              # 2652
SUITE_Z0, SUITE_Z1 = 900, 1020           # trim ledge must clear a player (72) on the top row (floor 800)
for seg in range(8):
    inward = lambda n2, seg=seg, t=None: t if n2 @ segn(seg) < -0.99 else "STADWALL"
    W.append(prism(ring_to_square(UTOP, seg), 0, SUITE_Z0, "STADWALL", top="CONCRETE"))
    W.append(prism(ring_to_square(UTOP, seg), SUITE_Z0, SUITE_Z1,
                   lambda n2, seg=seg: {"name": "SUITES", "su": 2, "sv": (SUITE_Z1 - SUITE_Z0) / 64,
                                        "ov": SUITE_Z1 / ((SUITE_Z1 - SUITE_Z0) / 64)} if n2 @ segn(seg) < -0.99 else "STADWALL"))
    W.append(prism(ring_to_square(UTOP, seg), SUITE_Z1, SKYTOP, "STADWALL"))
    W.append(prism(rq(UTOP - 10, UTOP, seg), SUITE_Z0 - 8, SUITE_Z0, "REDSTRIP"))

# ------------------------------------------------------------------ south: walkout stage, LED wall, red locker room
ST_A0, LED_A = 1300, 1716
W.append(prism(rq(ST_A0, LED_A, 6), 0, 24, "STAGE"))
bL = LED_A * T8
walk = lambda: fit("WALKOUT", (0, 1, 0), -bL, bL, 24, 424)
for x0, x1, z0, z1 in ((-bL, -48, 24, 424), (48, bL, 24, 424), (-48, 48, 136, 424)):
    W.append(box(x0, -LED_A - 16, z0, x1, -LED_A, z1, "METAL", n=walk()))
W.append(box(-bL, -LED_A - 16, 0, bL, -LED_A, 24, "STAGE"))
# mass behind the LED wall, carved for the red locker room (x +-320, y -2032..-1732, z 24..184)
RX, RY0, RY1, RZ0, RZ1 = 320, -2032, -LED_A - 16, 24, 184
mass = ring_to_square(LED_A + 16, 6)
L_ = clip2(mass, np.array([1.0, 0]), -RX); R_ = clip2(mass, np.array([-1.0, 0]), -RX)
W.append(prism(L_, 0, BAL_Z, "STADWALL", top="CONCRETE"))
W.append(prism(R_, 0, BAL_Z, "STADWALL", top="CONCRETE"))
W.append(box(-RX, -S, 0, RX, RY0, BAL_Z, "STADWALL", top="CONCRETE", n="LOCKER_R"))
W.append(box(-RX, RY0, 0, RX, RY1, RZ0, "STAGE"))
W.append(box(-RX, RY0, RZ1, RX, RY1, BAL_Z, "STADWALL", top="CONCRETE", bottom="CEILTILE"))
for seg_par in ((0, 0.46), (0.54, 1)):
    sp = {"name": "RIBBON1", "su": 1, "sv": 1, "ov": (BAL_Z + 32)}
    W.append(prism(rq(LED_A + 16, LED_A + 22, 6, *seg_par), BAL_Z, BAL_Z + 32,
                   lambda n2, sp=sp: sp if n2[1] > 0.99 else "STADWALL", top="CAGEPAD"))
# red locker room dressing
for x in range(-RX + 8, RX - 40, 48):
    if abs(x + 20) < 110: continue
    W.append(box(x, RY0, RZ0, x + 40, RY0 + 24, RZ0 + 96, "LOCKER_R"))
W.append(box(-RX, RY0 + 60, RZ0, -RX + 40, RY1 - 40, RZ0 + 96, "LOCKER_R"))
W.append(box(RX - 40, RY0 + 60, RZ0, RX, RY1 - 40, RZ0 + 96, "LOCKER_R"))
W.append(box(-200, -1930, RZ0, 200, -1910, RZ0 + 18, "COUCH"))
W.append(sign("CORNER_R", -112, RY0 + 2, RZ0 + 28, 104, RY0 + 6, RZ0 + 136, "n"))
light(0, -1880, 160, (255, 80, 70), 160); light(-200, -1800, 160, (255, 230, 210), 90); light(200, -1800, 160, (255, 230, 210), 90)
# runway + cage stairs
RUN = lambda name: {"name": name, "su": 112 / 128, "sv": 112 / 128, "ou": 56 / (112 / 128)}
W.append(box(-56, -ST_A0, 0, 56, -388, 16, "STAGE", top=RUN("WALKWAY_R")))
W.append(box(-56, 388, 0, 56, A0, 16, "STAGE", top=RUN("WALKWAY_B")))       # blue walkway
for x0 in (-60, 52):                                                           # red gate light bars
    W.append(box(x0, -LED_A, 24, x0 + 8, -LED_A + 8, 144, "REDSTRIP"))
W.append(box(-60, -LED_A, 136, 60, -LED_A + 8, 144, "REDSTRIP"))
for sgn in (1, -1):
    y_in = 340 * sgn
    for z, (ya, yb) in ((12, (388, 364)), (24, (364, 340))):
        y0, y1 = sorted((ya * sgn, yb * sgn))
        W.append(box(-40, y0, 0, 40, y1, z, "STAGE"))

# ------------------------------------------------------------------ north: blue locker room under the stands
BR_Y0, BR_Y1 = A0 + DEP * ROOM_K[0], A0 + DEP * (ROOM_K[1] - 1)  # 1476..1812
for x in range(-ROOM_X + 8, ROOM_X - 40, 48):
    if abs(x + 20) < 110: continue
    W.append(box(x, BR_Y1 - 24, 0, x + 40, BR_Y1, 96, "LOCKER_B"))
W.append(box(-ROOM_X, BR_Y0 + 40, 0, -ROOM_X + 24, BR_Y1 - 60, 96, "LOCKER_B"))
W.append(box(ROOM_X - 24, BR_Y0 + 40, 0, ROOM_X, BR_Y1 - 60, 96, "LOCKER_B"))
W.append(box(-160, 1640, 0, 160, 1660, 18, "COUCH"))
W.append(sign("CORNER_B", -112, BR_Y1 - 6, 28, 104, BR_Y1 - 2, 136, "s"))
for x0 in (-TUN_X - 8, TUN_X):
    W.append(box(x0, A0 - 8, 0, x0 + 8, A0, TUN_TOP, "BLUESTRIP"))
W.append(sign("CORNER_B", -96, A0 - 12, TUN_TOP + 24, 96, A0 - 8, TUN_TOP + 120, "s", back="CAGEPAD"))
for x in (-90, 82):
    W.append(box(x, A0 - 12, 0, x + 8, A0 - 8, TUN_TOP + 24, "METAL"))
light(0, 1600, 150, (80, 120, 255), 170); light(-180, 1720, 150, (220, 230, 255), 80); light(180, 1720, 150, (220, 230, 255), 80)
for y in (1000, 1200, 1400):
    W.append(box(-6, y - 6, TUN_TOP - 4, 6, y + 6, TUN_TOP, "METAL", bottom="LAMP"))
    light(0, y, 100, (120, 150, 255), 80)

# ------------------------------------------------------------------ the octagon
CA, FA, PA = 300, 304, 340        # canvas, fence, platform apothems
PZ = 36
W.append(prism(octv(CA), 0, PZ, "APRON", top=fit("CANVAS", (0, 0, 1), -307, 307, 0, 1) | {"su": 2.4, "sv": 2.4, "ou": 128, "ov": 128}))
def apron_side(n2, seg):
    """The apron banner fitted to exactly one octagon side, so it never wraps mid-word (it used to read "Batt|tleAxe")."""
    n3 = (float(n2[0]), float(n2[1]), 0.0)
    u, _ = axes(np.array(n3))
    us = [np.array([*onoct(PA, seg, t), 0.0]) @ u for t in (0.0, 1.0)]
    return fit("APRONSIDE", n3, min(us), max(us), 0, PZ)
for seg in range(8):
    W.append(prism(rq(CA, PA - 6, seg), 0, PZ, "APRON"))
    W.append(prism(rq(PA - 6, PA, seg), 0, PZ, lambda n2, seg=seg: apron_side(n2, seg) if n2 @ segn(seg) > 0.99 else "APRON",
                   top="REDSTRIP"))
fence, rail = [], []
for seg in range(8):
    spans = [(0, 0.36), (0.64, 1)] if seg in (2, 6) else [(0, 1)]
    for t0, t1 in spans:
        fence.append(prism(rq(CA, FA, seg, t0, t1), PZ, PZ + 80, {"name": "{CAGEFENCE", "su": 0.5, "sv": 0.5}))
        rail.append(prism(rq(CA - 2, FA + 2, seg, t0, t1), PZ + 80, PZ + 86, "CAGEPAD"))
bent("func_wall", fence, rendermode=4, renderamt=255)
W.extend(rail)
PH = 104
for (x, y) in octv(FA - 2):
    post = {}
    for f, nrm in (("n", (0, 1, 0)), ("s", (0, -1, 0)), ("e", (1, 0, 0)), ("w", (-1, 0, 0))):
        u, _ = axes(np.array(nrm, float))
        us = [np.array([x + dx, y + dy, 0]) @ u for dx in (-12, 12) for dy in (-12, 12)]
        post[f] = fit("CAGEPOST", nrm, min(us), max(us), PZ, PZ + PH)
    W.append(box(x - 12, y - 12, PZ, x + 12, y + 12, PZ + PH, "CAGEPAD", **post))

# ------------------------------------------------------------------ lighting rig, beams, jumbotron, towers
TZ = 560
for x0, y0, x1, y1 in ((-336, -336, 336, -320), (-336, 320, 336, 336), (-336, -320, -320, 320), (320, -320, 336, 320)):
    W.append(box(x0, y0, TZ, x1, y1, TZ + 16, "TRUSS"))
lamps = [(x, y) for x in (-328, 0, 328) for y in (-328, 0, 328) if (x, y) != (0, 0)]
beams = []
for (x, y) in lamps:
    W.append(box(x - 12, y - 12, TZ - 16, x + 12, y + 12, TZ, "METAL", bottom="LAMP"))
    light(x * 0.8, y * 0.8, TZ - 40, (255, 248, 235), 420)
    tx, ty = x * 0.35, y * 0.35
    top = [(x + dx, y + dy, TZ - 18) for dx in (-10, 10) for dy in (-10, 10)]
    bot = [(tx + dx, ty + dy, PZ + 2) for dx in (-46, 46) for dy in (-46, 46)]
    beams.append(hull_brush(top + bot, "LIGHTBEAM"))
for part in chunks(beams, 256):
    bent("func_illusionary", part, rendermode=5, renderamt=28)
light(0, 0, TZ - 60, (255, 255, 255), 700)
JZ0, JZ1 = 700, 860
W.append(box(-128, -128, JZ0, 128, 128, JZ1, "METAL", n=fit("JUMBO2", (0, 1, 0), -128, 128, JZ0, JZ1),
             s=fit("JUMBO1", (0, -1, 0), -128, 128, JZ0, JZ1), e=fit("JUMBO1", (1, 0, 0), -128, 128, JZ0, JZ1),
             w=fit("JUMBO2", (-1, 0, 0), -128, 128, JZ0, JZ1)))
W.append(box(-136, -136, JZ0 - 24, 136, 136, JZ0, "METAL", bottom=fit("AXELOGO", (0, 0, -1), -136, 136, -136, 136) | {"sv": 272 / 64, "ov": 136 / (272 / 64)}, n={"name": "RIBBON1", "su": 1.0625, "sv": 0.75, "ov": JZ0 / 0.75},
             s={"name": "RIBBON2", "su": 1.0625, "sv": 0.75, "ov": JZ0 / 0.75}, e={"name": "RIBBON3", "su": 1.0625, "sv": 0.75, "ov": JZ0 / 0.75},
             w={"name": "RIBBON1", "su": 1.0625, "sv": 0.75, "ov": JZ0 / 0.75}))
for sx in (-1, 1):
    for sy in (-1, 1):
        W.append(beam((sx * 328, sy * 328, TZ + 16), (sx * 120, sy * 120, JZ0 - 24), 4, "METAL"))
TW = 1900
for sx in (-1, 1):
    for sy in (-1, 1):
        cx, cy = sx * TW, sy * TW
        bx, by = cx - 60 * sx, cy - 60 * sy
        W.append(box(bx - 50, by - 50, 1300, bx + 50, by + 50, 1320, "METAL", bottom="LIGHTBANK",
                     **{("w" if sx > 0 else "e"): "LIGHTBANK", ("s" if sy > 0 else "n"): "LIGHTBANK"}))
        W.append(box(bx - 6, by - 6, 1320, bx + 6, by + 6, SKYTOP, "METAL"))
        W.append(beam((bx, by, 1320), (sx * 120, sy * 120, JZ1), 6, "METAL"))
        W.append(beam((sx * 120, sy * 120, JZ1), (sx * 120, sy * 120, SKYTOP), 3, "METAL"))
RZ = SKYTOP - 60                                       # roof rigging
for k in range(-4, 5):
    W.append(box(k * 600 - 10, -S + 200, RZ, k * 600 + 10, S - 200, RZ + 24, "TRUSS"))
    W.append(box(-S + 200, k * 600 - 10, RZ - 24, S - 200, k * 600 + 10, RZ, "TRUSS"))
for x in range(-2100, 2101, 600):
    for y in range(-2100, 2101, 600):
        if abs(x) < 400 and abs(y) < 400: continue
        W.append(box(x + 280 - 80, y + 300 - 10, SKYTOP - 8, x + 280 + 80, y + 300 + 10, SKYTOP, "METAL", bottom="LAMP"))
        light(cx - 200 * sx, cy - 200 * sy, 1240, (255, 250, 235), 4200)

# ------------------------------------------------------------------ floor: press row, broadcast desk, camera crane, floor seats
for seg in (0, 3, 4, 5, 7):
    W.append(prism(rq(430, 460, seg, 0.12, 0.88), 0, 30, "PRESS", top="TABLE"))
    for t in (0.25, 0.5, 0.75):
        W.append(prism(rq(446, 450, seg, t - 0.06, t + 0.06), 30, 46, lambda n2, seg=seg: {"name": "BROADCAST", "su": 0.6, "sv": 0.34, "ov": 46 / 0.34}
                       if n2 @ segn(seg) < -0.99 else "TVBLACK", top="TVBLACK"))
W.append(prism(rq(430, 490, 1, 0.15, 0.85), 0, 32, "PRESS", top="TABLE"))
for t in (0.3, 0.5, 0.7):
    W.append(prism(rq(470, 474, 1, t - 0.07, t + 0.07), 32, 56, lambda n2: {"name": "BROADCAST", "su": 0.7, "sv": 0.5, "ov": 56 / 0.5}
                   if n2 @ segn(1) < -0.99 else "TVBLACK", top="TVBLACK"))
cb = np.array([-370.0, 370.0])
W.append(box(cb[0] - 24, cb[1] - 24, 0, cb[0] + 24, cb[1] + 24, 64, "CAMERA"))
W.append(beam((cb[0] - 60, cb[1] + 60, 70), (cb[0] + 150, cb[1] - 150, 300), 10, "CAMERA"))
W.append(box(cb[0] + 140, cb[1] - 170, 270, cb[0] + 175, cb[1] - 135, 300, "CAMERA"))
W.append(box(cb[0] - 80, cb[1] + 50, 50, cb[0] - 50, cb[1] + 80, 90, "METAL"))
for seg in range(8):
    if seg == 6: continue
    for r in range(3):
        a0 = 560 + 64 * r
        for i, (t0, t1) in enumerate(AISLES):
            cp = rq(a0, a0 + 40, seg, t0, t1)
            parts = [cp]
            if seg == 2:
                parts = [p for p in (clip2(cp, np.array([1.0, 0]), -80), clip2(cp, np.array([-1.0, 0]), -80)) if p]
            for p in parts:
                CROWD.append(prism(p, 0, 40, crowd_spec(r + seg * 3 + i, 40), top="CROWDTOP"))

for part in chunks(CROWD, 384):        # the crowd: decoration only, no collision; in chunks so each has a small bbox
    bent("func_illusionary", part)

# ------------------------------------------------------------------ pyro + general lights
FIRE = "sprites/battleaxecombatleague_fire.spr"
def fire(x, y, z, sc):
    point("env_sprite", x, y, z, model=FIRE, rendermode=5, renderamt=255, rendercolor="255 255 255",
          framerate=16, scale=sc, spawnflags=1)
for x in (-560, -300, 300, 560):
    fire(x, -1330, 24, 2.4)
for x in (-90, 90):
    fire(x, -1290, 16, 1.4); fire(x, -900, 0, 1.2)
for x in (-100, 100):
    fire(x, 910, 0, 1.6)
light(0, -1450, 300, (255, 60, 40), 600); light(-450, -1450, 260, (255, 140, 40), 400); light(450, -1450, 260, (255, 140, 40), 400)
light(0, 960, 180, (80, 120, 255), 350)
for seg in range(8):
    p = onoct(1400, seg, 0.5); light(p[0], p[1], 1000, (240, 240, 255), 1100)
    p = onoct(2350, seg, 0.5); light(p[0], p[1], 1150, (240, 240, 255), 900)
    if seg != 6:
        p = onoct(1950, seg, 0.7); light(p[0], p[1], UND - 30, (255, 230, 200), 160)
    p = onoct(650, seg, 0.5); light(p[0], p[1], 420, (255, 245, 230), 300)

# ------------------------------------------------------------------ weapons + spawns (spawns ONLY in the locker rooms)
WEAPONS = ["weapon_shotgun", "weapon_9mmAR", "weapon_357", "weapon_crossbow", "weapon_rpg", "weapon_gauss",
           "weapon_egon", "weapon_hornetgun", "weapon_handgrenade", "weapon_satchel", "weapon_snark", "weapon_tripmine"]
AMMO = ["ammo_buckshot", "ammo_9mmAR", "ammo_357", "ammo_crossbow", "ammo_rpgclip", "ammo_gaussclip",
        "ammo_ARgrenades", "item_battery", "item_healthkit"]
wi = 0
for ring, n_, z in ((120, 8, PZ), (230, 12, PZ)):
    for i in range(n_):
        ang = 2 * math.pi * i / n_ + (0.2 if ring == 230 else 0)
        cls = WEAPONS[wi % len(WEAPONS)] if ring == 230 else AMMO[i % len(AMMO)]
        ITEMS.append((cls, ring * math.cos(ang), ring * math.sin(ang), z + 10)); wi += 1
for seg in range(8):
    if seg != 6:
        p = onoct(1950, seg, 0.85); ITEMS.append(("weapon_crossbow" if seg % 2 else "weapon_357", p[0], p[1], CONC_Z + 10))
        p = onoct(2040, seg, 0.6); ITEMS.append(("ammo_crossbow" if seg % 2 else "ammo_357", p[0], p[1], BAL_Z + 10))
    p = onoct(800, seg, 0.5)
    if seg not in (2, 6): ITEMS.append((AMMO[seg % len(AMMO)], p[0], p[1], 10))
for x in (-200, 200):
    ITEMS.append(("item_healthkit", x, -1500, 34))
for x in (-40, 40):
    ITEMS.append(("item_battery", x, 1100, 10))          # in the blue tunnel
for x, y, z in [(-200, -1830, RZ0), (-60, -1830, RZ0), (80, -1830, RZ0), (200, -1830, RZ0), (-120, -1870, RZ0), (120, -1870, RZ0),
                (-180, 1560, 0), (-60, 1560, 0), (60, 1560, 0), (180, 1560, 0), (-120, 1720, 0), (120, 1720, 0)]:
    face = 90 if y < 0 else 270
    SPAWNS.append((x, y, z + 37))
    point("info_player_deathmatch", x, y, z + 37, angles="0 %d 0" % face)
for cls, x, y, z in ITEMS:
    point(cls, round(x, 1), round(y, 1), z, angles="0 %d 0" % (int(abs(x * 3 + y)) % 8 * 45))
point("ambient_generic", 0, 0, 400, message="ambience/crowd1.wav", health=10, spawnflags=1)

STRUCT, DETAIL = split_detail(W)    # small brushes -> func_detail, so full VIS stays fast
world = ('{\n"classname" "worldspawn"\n"mapversion" "220"\n"wad" "battleaxecombatleague.wad"\n'
         '"message" "BattleAxe Combat League - Fight Night"\n' + "".join(STRUCT) + "}\n")
with open(os.path.join(OUT, "battleaxecombatleague.map"), "w", newline="\r\n") as f:
    f.write(world + detail_entity(DETAIL) + "".join(BENTS) + "".join(ENTS)
            + texlights(os.path.join(OUT, "battleaxecombatleague.wad"), fn_textures.EMISSIVE))
VALVE = os.path.join(OUT, "valve")
os.makedirs(os.path.join(VALVE, "gfx", "env"), exist_ok=True); os.makedirs(os.path.join(VALVE, "sprites"), exist_ok=True)
skyfire.write_sprite(os.path.join(VALVE, "sprites", "battleaxecombatleague_fire.spr"))
print("world brushes", len(STRUCT), "detail", len(DETAIL), "brush ents", len(BENTS), "items", len(ITEMS), "spawns", len(SPAWNS), "point ents", len(ENTS))
