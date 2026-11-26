"""Zone 'garage': the long driveway down the east side of the lot and the detached one-car garage at the back,
half of it converted to a home office. Ground z = 0; garage ceiling z = 144."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from _helpers import box, wall
import _textures as X
from tools.mapkit import textures as L

BOUNDS = (272, 256, -16, 480, 1856, 624)
TEXTURES = {
    "GARAGE_CONCRETE": L.T["CONCRETE"], "GARAGE_STUCCO": X.stucco(), "GARAGE_PLASTER": X.plaster,
    "GARAGE_ROOF": X.membrane, "GARAGE_TILE": X.rooftile, "GARAGE_FENCE": L.T["WOODFENCE"],
    "GARAGE_LAWN": X.lawn, "GARAGE_DESK": X.planks((70, 42, 24), (120, 80, 48)), "GARAGE_CRT": L.T["CRT"],
    "GARAGE_SHELF": L.T["METAL"], "GARAGE_BOX": X.cardboard, "GARAGE_DOOR": X.whitepaint,
}
GX0, GY0, GX1, GY1, GH = 272, 1472, 448, 1792, 144
ROUTES = [
    ("garage: up the driveway and into the garage", 0, [(360, 280), (360, 1440), (360, 1560), (360, 1660)]),
    ("garage: into the office and out the side door to the backyard", 0, [(360, 1620), (368, 1716), (360, 1630), (300, 1632), (230, 1632)]),
]
VIEWS = [
    ("driveway_to_garage", (360, 300, 90), (360, 1600, 60)),
    ("garage_office", (330, 1500, 90), (400, 1770, 40)),
]


def build(ctx):
    W, D = ctx.world, []
    W.append(box(272, 256, -16, 448, 1856, 0, "GARAGE_CONCRETE", scale=2))
    W.append(box(448, 256, -16, 480, 1856, 0, "GARAGE_LAWN", scale=2))
    W.append(box(448, 256, 0, 464, 1840, 96, "GARAGE_FENCE"))
    W.append(box(272, 1824, 0, 448, 1840, 96, "GARAGE_FENCE"))

    S, P = "GARAGE_STUCCO", "GARAGE_PLASTER"
    wall(ctx, GX0, GY0, GX1, GY0 + 16, 0, GH, S, n=P, gaps=[(304, 416, 0, 112)])            # roll-up door opening
    wall(ctx, GX0, GY1 - 16, GX1, GY1, 0, GH, S, s=P)
    wall(ctx, GX0, GY0 + 16, GX0 + 16, GY1 - 16, 0, GH, S, e=P, gaps=[(1600, 1664, 0, 112)])  # side door
    wall(ctx, GX1 - 16, GY0 + 16, GX1, GY1 - 16, 0, GH, S, w=P)
    W.append(box(GX0, GY0, GH, GX1, GY1, GH + 16, "GARAGE_ROOF", b=P, s=S, n=S, e=S, w=S))
    D.append(box(GX0 - 6, GY0 - 10, GH + 8, GX1 + 6, GY0 + 6, GH + 20, "GARAGE_TILE"))
    D.append(box(304, GY0 - 4, 104, 416, GY0 + 16, 112, "GARAGE_DOOR"))                   # the raised door's edge

    # office half: a partition with a doorway, desk with an old monitor
    D.append(box(288, 1680, 0, 336, 1696, 96, P))
    D.append(box(400, 1680, 0, 432, 1696, 96, P))
    D.append(box(352, 1736, 0, 432, 1776, 32, "GARAGE_DESK"))
    D.append(box(384, 1744, 32, 416, 1768, 56, "GARAGE_CRT"))
    # storage half: shelving and boxes
    D.append(box(408, 1504, 0, 432, 1656, 112, "GARAGE_SHELF"))
    D.append(box(296, 1500, 0, 344, 1540, 32, "GARAGE_BOX"))
    D.append(box(304, 1504, 32, 336, 1532, 56, "GARAGE_BOX"))
    ctx.brush_entity("func_detail", D, zhlt_detaillevel="1")
    ctx.point("light", 360, 1580, 128, _light="230 235 255 110")
    ctx.point("light", 360, 1730, 128, _light="255 236 200 90")

    ctx.point("info_player_deathmatch", 360, 900, 37, angles="0 90 0")
    ctx.point("weapon_rpg", 310, 1730, 10)                            # dead-end office: risky to fetch
    ctx.point("ammo_rpgclip", 330, 1580, 10)
    ctx.point("ammo_buckshot", 360, 600, 10)
