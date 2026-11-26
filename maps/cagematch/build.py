#!/usr/bin/env python3
"""Generates cagematch.map (Valve 220 format) and cagematch.wad (WAD3) for Half-Life DM."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))     # repo root, for tools.*
sys.path.insert(0, str(Path(__file__).resolve().parent))         # this map's own modules
from tools.paths import build_dir
import math, struct, os
import numpy as np

OUT = str(build_dir("cagematch"))
os.makedirs(OUT, exist_ok=True)

# ---------------------------------------------------------------- textures (WAD3)
rng = np.random.default_rng(7)

def noise(w, h, scale=1.0):
    n = rng.random((h, w))
    # cheap smoothing for a concrete feel
    for _ in range(2):
        n = (n + np.roll(n, 1, 0) + np.roll(n, 1, 1) + np.roll(n, -1, 0) + np.roll(n, -1, 1)) / 5
    n = (n - n.min()) / (n.max() - n.min() + 1e-9)
    return n

def gradient_palette(c0, c1):
    return [tuple(int(c0[k] + (c1[k] - c0[k]) * i / 255) for k in range(3)) for i in range(256)]

def tex_concrete():
    w = h = 128
    n = noise(w, h) * 0.6 + rng.random((h, w)) * 0.4
    idx = (40 + n * 170).astype(np.uint8)
    idx[::64, :] = 25; idx[:, ::64] = 25  # slab seams
    return idx, gradient_palette((20, 20, 22), (170, 168, 160))

def tex_mat():
    w = h = 128
    n = noise(w, h)
    idx = (60 + n * 60).astype(np.uint8)
    yy, xx = np.mgrid[0:h, 0:w]
    stripe = ((xx + yy) // 16) % 2 == 0
    idx[stripe] = np.clip(idx[stripe].astype(int) + 20, 0, 255)
    pal = gradient_palette((10, 10, 12), (90, 20, 20))
    return idx, pal

def tex_wall():
    w = h = 128
    n = noise(w, h)
    idx = (70 + n * 90).astype(np.uint8)
    idx[::32, :] = 30
    for r in range(0, 128, 32):
        off = 0 if (r // 32) % 2 == 0 else 32
        idx[r:r + 32, off::64] = 30
    return idx, gradient_palette((15, 18, 25), (110, 120, 140))

def tex_metal():
    w = h = 64
    n = noise(w, h)
    idx = (90 + n * 60).astype(np.uint8)
    idx[0, :] = 220; idx[:, 0] = 220; idx[-1, :] = 20; idx[:, -1] = 20
    for (x, y) in [(6, 6), (57, 6), (6, 57), (57, 57)]:
        idx[y - 1:y + 2, x - 1:x + 2] = 240
    return idx, gradient_palette((10, 10, 10), (190, 190, 200))

def tex_hazard():
    w = h = 64
    yy, xx = np.mgrid[0:h, 0:w]
    idx = np.where(((xx + yy) // 16) % 2 == 0, 230, 20).astype(np.uint8)
    idx = (idx.astype(int) + (rng.random((h, w)) * 20).astype(int)).clip(0, 255).astype(np.uint8)
    pal = [(int(i * 0.25), int(i * 0.2), 0) if i < 128 else (min(255, i + 10), int(i * 0.8), 0) for i in range(256)]
    return idx, pal

def tex_pad():
    w = h = 64
    n = noise(w, h)
    idx = (30 + n * 40).astype(np.uint8)
    idx[:, 28:36] = 200  # red stripe
    pal = [(int(i * 0.4), int(i * 0.4), int(i * 0.45)) if i < 128 else (200, 25, 25) for i in range(256)]
    return idx, pal

def tex_ceiling():
    w = h = 128
    n = noise(w, h)
    idx = (30 + n * 40).astype(np.uint8)
    idx[::32, :] = 110; idx[:, ::32] = 110
    return idx, gradient_palette((5, 5, 8), (120, 120, 130))

def tex_fence():
    """Chain-link. Palette index 255 = pure blue = transparent with rendermode Solid."""
    w = h = 64
    yy, xx = np.mgrid[0:h, 0:w]
    idx = np.full((h, w), 255, np.uint8)
    d1 = (xx + yy) % 16
    d2 = (xx - yy) % 16
    wire = (d1 < 2) | (d2 < 2)
    shade = (140 + 80 * ((xx + yy) % 16) / 16).astype(np.uint8)
    idx[wire] = shade[wire]
    pal = gradient_palette((40, 40, 45), (220, 222, 230))
    pal[255] = (0, 0, 255)
    return idx, pal

def tex_trigger():
    w = h = 16
    idx = np.full((h, w), 1, np.uint8)
    pal = [(0, 0, 0)] * 256
    pal[1] = (255, 140, 0)
    return idx, pal

TEXTURES = {
    "CAGE_CONC": tex_concrete,
    "CAGE_MAT": tex_mat,
    "CAGE_WALL": tex_wall,
    "CAGE_METAL": tex_metal,
    "CAGE_HAZARD": tex_hazard,
    "CAGE_PAD": tex_pad,
    "CAGE_CEIL": tex_ceiling,
    "{CAGE_FENCE": tex_fence,
    "AAATRIGGER": tex_trigger,
}

def build_miptex(name, idx, pal):
    h, w = idx.shape
    assert w % 16 == 0 and h % 16 == 0
    mips = [idx]
    for _ in range(3):
        mips.append(mips[-1][::2, ::2])
    header_size = 16 + 4 * 2 + 4 * 4
    offsets, pos = [], header_size
    for m in mips:
        offsets.append(pos); pos += m.size
    data = struct.pack("<16sII4I", name.encode().ljust(16, b"\0")[:16], w, h, *offsets)
    for m in mips:
        data += np.ascontiguousarray(m).tobytes()
    data += struct.pack("<H", 256) + bytes(c for rgb in pal for c in rgb) + b"\0\0"
    return data

def write_wad(path):
    lumps = [(n, build_miptex(n, *f())) for n, f in TEXTURES.items()]
    body, entries = b"", []
    pos = 12
    for name, data in lumps:
        entries.append(struct.pack("<IIIbbH16s", pos, len(data), len(data), 0x43, 0, 0,
                                   name.encode().ljust(16, b"\0")[:16]))
        body += data; pos += len(data)
    with open(path, "wb") as f:
        f.write(b"WAD3" + struct.pack("<II", len(lumps), pos))
        f.write(body)
        f.write(b"".join(entries))

# ---------------------------------------------------------------- geometry
def octagon(apothem):
    """Integer vertices of a regular octagon with flat sides facing the axes, CCW from +x/-y."""
    a = apothem
    b = round(apothem * math.tan(math.pi / 8))
    return [(a, -b), (a, b), (b, a), (-b, a), (-a, b), (-a, -b), (-b, -a), (b, -a)]

ROOM = 768          # half-size of the arena (inner face of outer walls)
WALL = 16
CEIL_Z = 448
BALC_Z = 192        # balcony floor height (fall from here = no fall damage)
SLAB = 16
FENCE_IN, FENCE_OUT = 384, 392
HOLE = 352          # balcony overhangs the fence by 32 units
OUTER_RAYS = [(ROOM, -318), (ROOM, 318), (318, ROOM), (-318, ROOM),
              (-ROOM, 318), (-ROOM, -318), (-318, -ROOM), (318, -ROOM)]
SQUARE_CORNERS = {1: (ROOM, ROOM), 3: (-ROOM, ROOM), 5: (-ROOM, -ROOM), 7: (ROOM, -ROOM)}

def sub(a, b): return tuple(a[i] - b[i] for i in range(3))
def cross(a, b): return (a[1]*b[2]-a[2]*b[1], a[2]*b[0]-a[0]*b[2], a[0]*b[1]-a[1]*b[0])
def dot(a, b): return sum(a[i]*b[i] for i in range(3))
def norm(v):
    l = math.sqrt(dot(v, v)); return tuple(x / l for x in v)

def tex_axes(n, edge=None):
    ax, ay, az = abs(n[0]), abs(n[1]), abs(n[2])
    if az >= ax and az >= ay:
        return (1, 0, 0), (0, -1, 0)
    if edge is not None:
        e = norm((edge[0], edge[1], 0))
        return e, (0, 0, -1)
    if ax >= ay:
        return (0, 1, 0), (0, 0, -1)
    return (1, 0, 0), (0, 0, -1)

def fmt(v):
    return " ".join(("%d" % x) if float(x).is_integer() else ("%.6f" % x) for x in v)

def face(p0, p1, p2, tex, scale=1.0, edge=None):
    n = norm(cross(sub(p0, p1), sub(p2, p1)))
    u, v = tex_axes(n, edge)
    return "( %s ) ( %s ) ( %s ) %s [ %s 0 ] [ %s 0 ] 0 %g %g" % (
        fmt(p0), fmt(p1), fmt(p2), tex, fmt(u), fmt(v), scale, scale)

def prism(poly, z0, z1, tex, top=None, bottom=None, side=None, scale=1.0):
    """Convex prism from a CCW (viewed from +z) 2D polygon. side may be a function(i)->tex."""
    top = top or tex; bottom = bottom or tex
    P = poly
    faces = []
    A, B, C = P[0], P[1], P[2]
    faces.append(face((B[0], B[1], z1), (A[0], A[1], z1), (C[0], C[1], z1), top, scale))
    faces.append(face((C[0], C[1], z0), (A[0], A[1], z0), (B[0], B[1], z0), bottom, scale))
    for i in range(len(P)):
        a, b = P[i], P[(i + 1) % len(P)]
        t = side(i) if callable(side) else (side or tex)
        faces.append(face((b[0], b[1], z0), (a[0], a[1], z0), (a[0], a[1], z1), t, scale,
                          edge=(b[0] - a[0], b[1] - a[1])))
    check_convex(P)
    return "{\n" + "\n".join(faces) + "\n}\n"

def box(x0, y0, z0, x1, y1, z1, tex, **kw):
    return prism([(x0, y0), (x1, y0), (x1, y1), (x0, y1)], z0, z1, tex, **kw)

def check_convex(P):
    s = None
    for i in range(len(P)):
        a, b, c = P[i], P[(i + 1) % len(P)], P[(i + 2) % len(P)]
        z = (b[0]-a[0])*(c[1]-b[1]) - (b[1]-a[1])*(c[0]-b[0])
        assert z > 0, ("non-convex or CW polygon", P)

def ring_pieces(inner_apothem):
    """8 convex pieces filling the space between an octagon and the room square."""
    v = octagon(inner_apothem)
    pieces = []
    for i in range(8):
        j = (i + 1) % 8
        poly = [v[i], OUTER_RAYS[i]]
        if i in SQUARE_CORNERS:
            poly.append(SQUARE_CORNERS[i])
        poly += [OUTER_RAYS[j], v[j]]
        pieces.append(poly)
    return pieces

def fence_segments():
    vi, vo = octagon(FENCE_IN), octagon(FENCE_OUT)
    return [[vi[i], vo[i], vo[(i + 1) % 8], vi[(i + 1) % 8]] for i in range(8)]

# ---------------------------------------------------------------- build the map
world = []
R, W = ROOM, WALL
# outer shell
world.append(box(-R - W, -R - W, CEIL_Z, R + W, R + W, CEIL_Z + W, "CAGE_CEIL"))
world.append(box(R, -R - W, -W, R + W, R + W, CEIL_Z, "CAGE_WALL"))
world.append(box(-R - W, -R - W, -W, -R, R + W, CEIL_Z, "CAGE_WALL"))
world.append(box(-R, R, -W, R, R + W, CEIL_Z, "CAGE_WALL"))
world.append(box(-R, -R - W, -W, R, -R, CEIL_Z, "CAGE_WALL"))
# floor: octagonal mat inside the cage + concrete outside
world.append(prism(octagon(FENCE_IN), -W, 0, "CAGE_MAT", side="CAGE_CONC"))
for p in ring_pieces(FENCE_IN):
    world.append(prism(p, -W, 0, "CAGE_CONC"))
# balcony slab ring with a hazard lip facing the pit
for p in ring_pieces(HOLE):
    # side index 4 is the edge v[j]->v[i] (the inner hole edge) for 5-gons, index 3 for 4-gons
    inner = len(p) - 1
    world.append(prism(p, BALC_Z - SLAB, BALC_Z, "CAGE_CONC", bottom="CAGE_METAL",
                       side=lambda i, inner=inner: "CAGE_HAZARD" if i == inner else "CAGE_METAL"))
# padded corner posts at each fence vertex
for (x, y) in octagon(388):
    world.append(box(x - 8, y - 8, 0, x + 8, y + 8, BALC_Z - SLAB, "CAGE_PAD"))

ents = []
ents.append('{\n"classname" "worldspawn"\n"mapversion" "220"\n"wad" "cagematch.wad"\n'
            '"message" "Cage Match"\n"skyname" "desert"\n"sounds" "1"\n' + "".join(world) + "}\n")

# chain-link fence (see-through, solid to players and bullets)
fence = "".join(prism(s, 0, BALC_Z - SLAB, "{CAGE_FENCE", scale=0.5) for s in fence_segments())
ents.append('{\n"classname" "func_wall"\n"rendermode" "4"\n"renderamt" "255"\n' + fence + "}\n")

# downward push in the drop hole: stops anyone gauss/grenade-jumping back out of the pit
push = prism(octagon(HOLE), 136, BALC_Z + 64, "AAATRIGGER")
ents.append('{\n"classname" "trigger_push"\n"angles" "0 -2 0"\n"speed" "1000"\n' + push + "}\n")

def point(classname, x, y, z, **kv):
    s = '{\n"classname" "%s"\n"origin" "%d %d %d"\n' % (classname, x, y, z)
    for k, v in kv.items():
        s += '"%s" "%s"\n' % (k, v)
    return s + "}\n"

def facing_center(x, y):
    return "0 %d 0" % (round(math.degrees(math.atan2(-y, -x))) % 360)

# pit spawns: fire the loadout equip on spawn
for (x, y) in [(220, 0), (-220, 0), (0, 220), (0, -220)]:
    ents.append(point("info_player_deathmatch", x, y, 37, angles=facing_center(x, y), target="pit_loadout"))

# balcony spawns: unarmed
for k in range(12):
    a = math.radians(k * 30 + 15)
    x, y = round(590 * math.cos(a)), round(590 * math.sin(a))
    x = max(-R + 40, min(R - 40, x)); y = max(-R + 40, min(R - 40, y))
    ents.append(point("info_player_deathmatch", x, y, BALC_Z + 37, angles=facing_center(x, y)))

# Loadout. "Use only" (spawnflag 1) means it never auto-equips everyone; because a
# game_player_equip exists, HLDM also stops handing out the default crowbar + 9mm.
loadout = {
    "weapon_crowbar": 1, "weapon_9mmhandgun": 1, "weapon_357": 1, "weapon_9mmAR": 1,
    "weapon_shotgun": 1, "weapon_crossbow": 1, "weapon_rpg": 1, "weapon_gauss": 1,
    "weapon_egon": 1, "weapon_hornetgun": 1, "weapon_handgrenade": 1, "weapon_satchel": 1,
    "weapon_tripmine": 1, "weapon_snark": 1,
    "ammo_9mmclip": 3, "ammo_357": 2, "ammo_ARgrenades": 2, "ammo_buckshot": 3,
    "ammo_crossbow": 2, "ammo_rpgclip": 2, "ammo_gaussclip": 3, "item_battery": 2,
}
ents.append(point("game_player_equip", 0, 0, 300, targetname="pit_loadout", spawnflags=1, **loadout))

# loose weapons/ammo on the pit floor (respawn as normal in HLDM)
floor_items = ["weapon_shotgun", "weapon_9mmAR", "weapon_rpg", "weapon_gauss", "weapon_egon",
               "weapon_crossbow", "weapon_357", "weapon_hornetgun", "weapon_handgrenade",
               "weapon_satchel", "weapon_tripmine", "weapon_snark"]
for k, cls in enumerate(floor_items):
    a = math.radians(k * 30)
    ents.append(point(cls, round(300 * math.cos(a)), round(300 * math.sin(a)), 16,
                      angles="0 %d 0" % ((k * 30 + 90) % 360)))
ammo = ["ammo_buckshot", "ammo_9mmAR", "ammo_rpgclip", "ammo_gaussclip",
        "ammo_crossbow", "ammo_357", "ammo_ARgrenades", "item_healthkit"]
for k, cls in enumerate(ammo):
    a = math.radians(k * 45 + 22.5)
    ents.append(point(cls, round(120 * math.cos(a)), round(120 * math.sin(a)), 16))
ents.append(point("item_battery", 0, 0, 16))

# lighting: bright over the cage, moodier balcony, dim red under the stands
ents.append(point("light", 0, 0, 400, _light="255 250 235 600"))
for (x, y) in octagon(200):
    ents.append(point("light", x, y, 150, _light="255 245 225 180"))
for k in range(8):
    a = math.radians(k * 45 + 22.5)
    ents.append(point("light", round(580 * math.cos(a)), round(580 * math.sin(a)), 400,
                      _light="200 210 255 250"))
for k in range(8):
    a = math.radians(k * 45)
    ents.append(point("light", round(600 * math.cos(a)), round(600 * math.sin(a)), 120,
                      _light="255 60 40 120"))

with open(os.path.join(OUT, "cagematch.map"), "w", newline="\r\n") as f:
    f.write("".join(ents))
write_wad(os.path.join(OUT, "cagematch.wad"))
print("ok", len(world), "world brushes,", len(ents), "entities")
