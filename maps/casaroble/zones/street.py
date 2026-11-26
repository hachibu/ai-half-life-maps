"""Zone 'street': the quiet residential street, sidewalk and the front yard: low-water gravel garden with agave
planters, a concrete walk up to a tiled entry porch, and the front of the driveway. Ground is z = 0; the road
sits 6 units lower behind the curb."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from _helpers import box
import _textures as X
from tools.mapkit import textures as L

BOUNDS = (-480, -400, -16, 480, 256, 624)
TEXTURES = {
    "STREET_ASPHALT": X.asphalt, "STREET_CONCRETE": L.T["CONCRETE"], "STREET_GRAVEL": X.gravel,
    "STREET_LAWN": X.lawn, "STREET_STUCCO": X.stucco((200, 182, 150), (232, 216, 188)), "STREET_AGAVE": X.agave,
    "STREET_FENCE": L.T["WOODFENCE"], "STREET_TILE": X.rooftile, "STREET_TERRA": X.terracotta,
    "STREET_CARBODY": L.T["CARSILVER"], "STREET_CARGLASS": L.T["CARGLASS"], "STREET_TIRE": L.T["TIRE"],
    "STREET_METAL": L.T["METAL"], "STREET_HEDGE": X.hedge,
}
ROUTES = [
    ("street: along the sidewalk", 0, [(-420, -50), (420, -50)]),
    ("street: across the road and back over the curb", -6, [(-420, -160), (420, -160), (420, -50)]),
    ("street: sidewalk up the front walk and onto the porch", 0, [(-144, -50), (-144, 150), (-144, 236)]),
    ("street: front yard to the side yard gate", 0, [(-300, 160), (-424, 200), (-424, 250)]),
    ("street: sidewalk up the driveway", 0, [(360, -50), (360, 250)]),
]
VIEWS = [
    ("front_from_street", (60, -330, 120), (-80, 400, 110)),
    ("front_porch", (-60, 120, 70), (-150, 256, 90)),
]


def build(ctx):
    W, D = ctx.world, []
    # road (z -6), sidewalk and curb, then the lot
    W.append(box(-480, -400, -16, 480, -112, -6, "STREET_ASPHALT", scale=2))
    W.append(box(-480, -112, -16, 480, 0, 0, "STREET_CONCRETE", scale=2))
    for x0, x1, t in ((-480, -448, "STREET_LAWN"), (-448, -176, "STREET_GRAVEL"), (-176, -112, "STREET_CONCRETE"),
                      (-112, 272, "STREET_GRAVEL"), (272, 448, "STREET_CONCRETE"), (448, 480, "STREET_LAWN")):
        W.append(box(x0, 0, -16, x1, 256, 0, t, scale=2))
    # porch: two 8-unit steps up to the house floor (16), tiled, under a clay-tile roof on stucco posts
    D.append(box(-176, 176, 0, -112, 208, 8, "STREET_TERRA"))
    W.append(box(-208, 208, 0, -80, 256, 16, "STREET_TERRA", s="STREET_STUCCO", e="STREET_STUCCO", w="STREET_STUCCO"))
    for x0 in (-224, -80):
        D.append(box(x0, 192, 0, x0 + 16, 208, 160, "STREET_STUCCO"))
    D.append(box(-232, 184, 160, -56, 256, 176, "STREET_TILE", b="STREET_STUCCO"))
    # low stucco planters with agave
    for x0, y0, x1, y1 in ((-400, 64, -240, 112), (16, 64, 208, 112)):
        D.append(box(x0, y0, 0, x1, y1, 24, "STREET_STUCCO", t="STREET_GRAVEL"))
        for i, x in enumerate(range(x0 + 16, x1 - 16, 48)):
            D.append(box(x, y0 + 12, 24, x + 24, y1 - 12, 40 + 8 * (i % 2), "STREET_AGAVE"))
    # side fences and a hedge between the lot and the neighbours
    for x0 in (-464, 448):
        W.append(box(x0, 0, 0, x0 + 16, 256, 64, "STREET_FENCE"))
        D.append(box(x0 + (-16 if x0 < 0 else 16), 0, 0, x0 + (0 if x0 < 0 else 32), 256, 112, "STREET_HEDGE"))
    # a parked car (cover on the street) and a street lamp
    cx0, cy0 = -300, -300
    for dx, dy in ((24, 0), (176, 0), (24, 88), (176, 88)):
        D.append(box(cx0 + dx, cy0 + dy - 8, -6, cx0 + dx + 40, cy0 + dy + 8, 20, "STREET_TIRE"))
    D.append(box(cx0, cy0 + 4, 8, cx0 + 240, cy0 + 84, 44, "STREET_CARBODY"))
    D.append(box(cx0 + 56, cy0 + 12, 44, cx0 + 176, cy0 + 76, 72, "STREET_CARGLASS", t="STREET_CARBODY"))
    D.append(box(392, -100, 0, 404, -88, 200, "STREET_METAL"))
    D.append(box(360, -108, 200, 404, -80, 208, "STREET_METAL"))
    ctx.brush_entity("func_detail", D, zhlt_detaillevel="1")
    ctx.point("light", 376, -94, 190, _light="255 214 150 160")
    ctx.point("light", -144, 220, 140, _light="255 214 160 90")

    for x, y, a in ((100, -50, 90), (-320, 180, 0), (300, -250, 180)):
        ctx.point("info_player_deathmatch", x, y, 37 if y > -112 else 31, angles="0 %d 0" % a)
    ctx.point("weapon_handgrenade", 120, 180, 10)
    ctx.point("ammo_9mmclip", -180, -200, 4)
