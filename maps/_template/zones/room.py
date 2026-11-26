"""Zone 'room': a lit room with four spawns, a crate you climb by 8-unit stairs and two weapons. Copy this file to
add a zone.

Things new maps get wrong (see AGENTS.md: Lessons and References):
- Stairs: use stairs() from _helpers (8-unit risers); every walkable area needs a route in ROUTES.
- Strong weapons (RPG, gauss, egon, crossbow) go in exposed or dead-end spots, well away from spawns.
- Outdoors: one zone must register the lowercase sky texture, e.g. TEXTURES = {"sky": L.T["sky"]}.
- Small brushes (furniture, trim, steps) go in a func_detail entity so full VIS stays fast.
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from _helpers import box, stairs
from tools.mapkit import textures as L

BOUNDS = (-528, -528, -16, 528, 528, 272)          # this zone's space; other zones must not overlap it
TEXTURES = {"ROOM_FLOOR": L.T["WOODFLOOR"], "ROOM_WALL": L.T["WALLPAPER"],
            "ROOM_CEIL": L.T["CEILING"], "ROOM_CRATE": L.T["CONCRETE"]}   # zone-prefixed names
ROUTES = [("room: corner to corner around the crate", 0, [(-300, -300), (-300, 300), (300, 300)]),
          ("room: up the stairs onto the crate", 0, [(0, -200), (0, -100), (0, 0)])]
VIEWS = [("room", (-400, -400, 120), (100, 100, 40))]


def build(ctx):
    R, H = 512, 256
    for b in [(-R - 16, -R - 16, -16, R + 16, R + 16, 0, "ROOM_FLOOR"), (-R - 16, -R - 16, H, R + 16, R + 16, H + 16, "ROOM_CEIL"),
              (R, -R, 0, R + 16, R, H, "ROOM_WALL"), (-R - 16, -R, 0, -R, R, H, "ROOM_WALL"),
              (-R, R, 0, R, R + 16, H, "ROOM_WALL"), (-R, -R - 16, 0, R, -R, H, "ROOM_WALL"),
              (-64, -64, 0, 64, 64, 48, "ROOM_CRATE")]:
        ctx.brush(box(*b))
    # 8-unit stairs up the crate's south side, as func_detail (small brushes don't cut VIS)
    ctx.brush_entity("func_detail", stairs(-32, -136, 32, -64, 0, 48, "ROOM_CRATE"), zhlt_detaillevel="1")
    for x, y in ((-300, -300), (300, -300), (-300, 300), (300, 300)):
        ctx.point("info_player_deathmatch", x, y, 37, angles="0 0 0")      # floor + 37
    ctx.point("weapon_shotgun", 0, 0, 58)                                    # on the crate, in the open
    ctx.point("weapon_9mmAR", 0, 200, 10)
    ctx.point("light", -200, 0, 200, _light="255 240 220 300")
    ctx.point("light", 200, 0, 200, _light="255 240 220 300")
