"""Zone 'backyard': the private deck off the family room and primary bedroom, ringed by oaks, and the lawn behind
it. Deck top z = 16 (flush with the house floor), one 8-unit step down to the lawn at z = 0."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from _helpers import box, canopy
import _textures as X
from tools.mapkit import textures as L

BOUNDS = (-480, 1216, -16, 272, 1856, 624)
TEXTURES = {
    "YARD_LAWN": X.lawn, "YARD_DECK": X.planks((105, 72, 45), (168, 124, 82)), "YARD_BARK": X.bark,
    "YARD_LEAVES": X.leaves, "YARD_FENCE": L.T["WOODFENCE"], "YARD_GRAVEL": X.gravel,
    "YARD_TABLE": X.planks((70, 42, 24), (120, 80, 48)),
}
OAKS = ((200, 1330), (-330, 1600), (60, 1690), (-190, 1530))
ROUTES = [
    ("backyard: family slider across the deck and down to the lawn", 16, [(128, 1180), (128, 1240), (-100, 1300), (-100, 1460), (-100, 1700)]),
    ("backyard: primary slider to the deck", 16, [(-160, 1240), (-260, 1240), (-300, 1380)]),
    ("backyard: side yard to the deck and round the oaks", 0, [(-424, 1230), (-424, 1500), (-260, 1490), (-60, 1490), (-60, 1770), (160, 1770), (240, 1500)]),
]
VIEWS = [
    ("deck_under_the_oaks", (220, 1780, 140), (-120, 1300, 40)),
    ("back_of_house", (-40, 1800, 90), (0, 1216, 110)),
]


def build(ctx):
    W, D = ctx.world, []
    W.append(box(-480, 1216, -16, 272, 1856, 0, "YARD_LAWN", scale=2))
    # deck, its 8-unit step along the lawn side, and a railing on the west end
    W.append(box(-400, 1216, 0, 128, 1440, 16, "YARD_DECK", n="YARD_DECK"))
    D.append(box(-400, 1440, 0, 128, 1464, 8, "YARD_DECK"))
    D.append(box(-400, 1432, 16, -232, 1440, 52, "YARD_DECK"))
    D.append(box(-224, 1264, 16, -136, 1352, 46, "YARD_TABLE"))       # outdoor table (cover)
    # oaks: solid trunks, leafy canopies far above head height
    for x, y in OAKS:
        D.append(box(x - 20, y - 20, 0, x + 20, y + 20, 256, "YARD_BARK"))
        canopy(ctx, x, y, "YARD_LEAVES", (-480, 1216, 272, 1856))
    # fences: west side and back
    W.append(box(-464, 1216, 0, -448, 1840, 96, "YARD_FENCE"))
    W.append(box(-464, 1824, 0, 272, 1840, 96, "YARD_FENCE"))
    ctx.brush_entity("func_detail", D, zhlt_detaillevel="1")
    ctx.point("light", -100, 1330, 140, _light="255 220 170 120")

    for x, y, a in ((-280, 1700, 90), (140, 1770, 180)):
        ctx.point("info_player_deathmatch", x, y, 37, angles="0 %d 0" % a)
    ctx.point("info_player_deathmatch", -300, 1300, 53, angles="0 270 0")
    ctx.point("weapon_crossbow", 40, 1400, 26)                        # out on the open deck
    ctx.point("ammo_crossbow", -400, 1780, 10)
    ctx.point("weapon_satchel", 200, 1560, 10)
