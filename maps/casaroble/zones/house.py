"""Zone 'house': a one-storey 1920s Spanish bungalow. Stucco walls, flat roof behind a parapet with a clay-tile
coping, copper downspouts. Inside: living room with a wood-burning fireplace, dining room, kitchen with a breakfast
bar and a skylight, three bedrooms (the primary has its own bath and a sliding door to the deck), a hall bath and a
family room that slides open onto the deck. House floor is z = 16, ceiling z = 176.

Plan (x east, y north; the street is south):
    y 272-568   living (x -384..-56)          | dining (x -40..256)
    y 584-792   bedroom 2 | hall bath          | kitchen (x -40..256, y 584..888), skylight
    y 808-888   hall (x -384..-40) ------------> kitchen
    y 904-1200  bedroom 3 | primary + ensuite  | family room (x 48..256)
"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from _helpers import box, wall, FLOOR, CEIL, DOOR
import _textures as X
from tools.mapkit import textures as L

BOUNDS = (-480, 256, -16, 272, 1216, 624)
TEXTURES = {
    "HOUSE_STUCCO": X.stucco(), "HOUSE_PLASTER": X.plaster, "HOUSE_ROOFTILE": X.rooftile, "HOUSE_TERRA": X.terracotta,
    "HOUSE_COPPER": X.copper, "HOUSE_WOOD": L.T["WOODFLOOR"], "HOUSE_CEIL": L.T["CEILING"], "HOUSE_ROOF": X.membrane,
    "HOUSE_BRICK": L.T["BRICK"], "HOUSE_GLASS": L.T["GLASS"], "HOUSE_COUNTER": L.T["COUNTER"],
    "HOUSE_FRIDGE": L.T["FRIDGE"], "HOUSE_METAL": L.T["METAL"], "HOUSE_TABLE": X.planks((70, 42, 24), (120, 80, 48)),
    "HOUSE_COUCH": L.T["COUCH"], "HOUSE_RUG": L.T["CARPET"], "HOUSE_TV": L.T["TVBLACK"], "HOUSE_WHITE": X.whitepaint,
    "HOUSE_BED": X.bedspread((60, 92, 120), (110, 148, 175)), "HOUSE_BED2": X.bedspread((150, 110, 60), (205, 165, 105)),
    "HOUSE_SIDEYARD": X.gravel, "HOUSE_FENCE": L.T["WOODFENCE"],
}
ROUTES = [
    ("house: porch through the front door, living and dining to the kitchen", FLOOR,
     [(-144, 236), (-144, 300), (-100, 420), (0, 500), (0, 640), (0, 760)]),
    ("house: kitchen down the hall to bedroom 2 and the hall bath", FLOOR,
     [(0, 848), (-288, 848), (-288, 760), (-288, 848), (-144, 848), (-144, 720)]),
    ("house: hall to bedroom 3", FLOOR, [(-200, 848), (-288, 848), (-288, 960), (-300, 1040)]),
    ("house: hall into the primary bedroom, its bath and out the slider", FLOOR,
     [(-48, 848), (-48, 940), (-84, 964), (-168, 964), (-84, 964), (-84, 1100), (-160, 1180), (-160, 1240)]),
    ("house: kitchen into the family room and out the slider", FLOOR, [(100, 780), (176, 960), (176, 1100), (128, 1180), (128, 1240)]),
    ("house: side yard along the west wall", 0, [(-424, 280), (-424, 1200)]),
]
VIEWS = [
    ("living_fireplace", (-90, 530, 88), (-384, 410, 56)),
    ("kitchen_skylight", (-20, 600, 80), (150, 760, 150)),
    ("family_room_to_deck", (220, 920, 84), (120, 1220, 60)),
    ("primary_bedroom", (-24, 930, 90), (-170, 1190, 60)),
    ("hall", (-30, 848, 70), (-384, 848, 60)),
]

HX0, HY0, HX1, HY1 = -400, 256, 272, 1216      # outside of the exterior walls
TOP = 192                                      # roof slab top
WIN = (64, 144)                                # window sill and head


def build(ctx):
    W, D = ctx.world, []                      # D: furniture and trim -> one func_detail
    S, P = "HOUSE_STUCCO", "HOUSE_PLASTER"

    # ground under the house zone (side yard strip and under the slab), and the west fence
    W.append(box(-464, 256, 0, -448, 1216, 96, "HOUSE_FENCE"))
    W.append(box(-480, 256, -16, -400, 1216, 0, "HOUSE_SIDEYARD", scale=2))
    W.append(box(-400, 256, -16, 272, 1216, 0, "HOUSE_SIDEYARD", scale=4))

    # floors (z 0..16): hardwood except tile in the kitchen and baths
    for x0, y0, x1, y1, t in [(-400, 256, 272, 584, "HOUSE_WOOD"), (-400, 584, -208, 808, "HOUSE_WOOD"),
                              (-208, 584, -40, 808, "HOUSE_TERRA"), (-40, 584, 272, 904, "HOUSE_TERRA"),
                              (-400, 808, -40, 904, "HOUSE_WOOD"), (-400, 904, -208, 1216, "HOUSE_WOOD"),
                              (-208, 904, -112, 1040, "HOUSE_TERRA"), (-208, 1040, -112, 1216, "HOUSE_WOOD"),
                              (-112, 904, 272, 1216, "HOUSE_WOOD")]:
        W.append(box(x0, y0, 0, x1, y1, FLOOR, t, s=S, n=S, e=S, w=S))

    # exterior walls (stucco outside, plaster inside), with breakable windows and the two sliders
    g = "HOUSE_GLASS"
    wall(ctx, HX0, HY0, HX1, HY0 + 16, FLOOR, TOP, S, glass=g, n=P,
         gaps=[(-352, -224, *WIN), (-176, -112, FLOOR, DOOR), (16, 208, *WIN)])           # front
    wall(ctx, HX0, HY1 - 16, HX1, HY1, FLOOR, TOP, S, glass=g, s=P,
         gaps=[(-352, -272, *WIN), (-192, -64, FLOOR, DOOR), (96, 224, FLOOR, DOOR)])      # back: sliders
    wall(ctx, HX0, HY0 + 16, HX0 + 16, HY1 - 16, FLOOR, TOP, S, glass=g, e=P,
         gaps=[(512, 560, *WIN), (640, 736, *WIN), (960, 1088, *WIN)])                     # west
    wall(ctx, HX1 - 16, HY0 + 16, HX1, HY1 - 16, FLOOR, TOP, S, glass=g, w=P,
         gaps=[(320, 512, *WIN), (640, 736, 72, 144), (960, 1104, *WIN)])                  # east
    W.append(box(HX0, HY0, 0, HX1, HY0 + 16, FLOOR, S))                                    # (slab edge is stucco)
    # sliders: the east half of each stays shut (fixed glass), the west half is open
    for x0, x1 in ((-128, -64), (160, 224)):
        ctx.brush_entity("func_wall", [box(x0, HY1 - 10, FLOOR, x1, HY1 - 6, DOOR, g)], rendermode="2", renderamt="100")
    for x0, x1 in ((-192, -64), (96, 224)):                                                # door frames
        D.append(box(x0, HY1 - 18, DOOR, x1, HY1 - 14, DOOR + 8, "HOUSE_WHITE"))

    # interior walls (plaster), doors are open 64-wide gaps
    dz = (FLOOR, DOOR)
    for x0, y0, x1, y1, gaps in [
        (-384, 568, -40, 584, []),                                        # living | bedroom 2, bath
        (-56, 272, -40, 568, [(336, 528, FLOOR, 152)]),                   # living | dining: wide opening
        (-224, 584, -208, 792, []),                                       # bedroom 2 | bath
        (-56, 584, -40, 792, []),                                         # bath | kitchen
        (-384, 792, -40, 808, [(-320, -256, *dz), (-176, -112, *dz)]),    # hall wall: bedroom 2, bath
        (-384, 888, 32, 904, [(-320, -256, *dz), (-80, -16, *dz)]),       # hall wall: bedroom 3, primary
        (32, 888, 256, 904, [(64, 224, FLOOR, 152)]),                     # kitchen | family: wide opening
        (-224, 904, -208, 1200, []),                                      # bedroom 3 | primary
        (32, 904, 48, 1200, []),                                          # primary | family
        (-128, 904, -112, 1040, [(936, 992, *dz)]),                       # ensuite east wall + door
        (-208, 1024, -128, 1040, []),                                     # ensuite north wall
    ]:
        wall(ctx, x0, y0, x1, y1, FLOOR, CEIL, P, gaps=gaps)
    D.append(box(-40, 568, 144, 256, 584, CEIL, P))                       # soffit over the breakfast bar

    # roof slab with a hole for the kitchen skylight; parapet with clay-tile coping
    sk = (96, 672, 192, 768)
    for x0, y0, x1, y1 in [(HX0, HY0, HX1, sk[1]), (HX0, sk[3], HX1, HY1), (HX0, sk[1], sk[0], sk[3]), (sk[2], sk[1], HX1, sk[3])]:
        W.append(box(x0, y0, CEIL, x1, y1, TOP, "HOUSE_CEIL", t="HOUSE_ROOF", s=S, n=S, e=S, w=S))
    for x0, y0, x1, y1 in [(sk[0] - 8, sk[1] - 8, sk[2] + 8, sk[1]), (sk[0] - 8, sk[3], sk[2] + 8, sk[3] + 8),
                           (sk[0] - 8, sk[1], sk[0], sk[3]), (sk[2], sk[1], sk[2] + 8, sk[3])]:
        W.append(box(x0, y0, CEIL, x1, y1, 232, P, t=S))                  # light well
    W.append(box(sk[0] - 8, sk[1] - 8, 232, sk[2] + 8, sk[3] + 8, 240, "sky"))
    for b in [(HX0, HY0, TOP, HX1, HY0 + 16, 224), (HX0, HY1 - 16, TOP, HX1, HY1, 224),
              (HX0, HY0 + 16, TOP, HX0 + 16, HY1 - 16, 224), (HX1 - 16, HY0 + 16, TOP, HX1, HY1 - 16, 224)]:
        W.append(box(*b, S))
    for b in [(HX0 - 6, HY0 - 6, 224, HX1 + 6, HY0 + 18, 234), (HX0 - 6, HY1 - 18, 224, HX1 + 6, HY1 + 6, 234),
              (HX0 - 6, HY0 + 18, 224, HX0 + 18, HY1 - 18, 234), (HX1 - 18, HY0 + 18, 224, HX1 + 6, HY1 - 18, 234)]:
        D.append(box(*b, "HOUSE_ROOFTILE"))
    # tile eyebrows over the big front windows, a stepped gable over the entry, copper downspouts
    for x0, x1 in ((-368, -208), (0, 224)):
        D.append(box(x0, HY0 - 20, 150, x1, HY0, 160, "HOUSE_ROOFTILE"))
    D.append(box(-208, HY0 - 4, TOP, -80, HY0 + 16, 248, S))
    D.append(box(-216, HY0 - 10, 248, -72, HY0 + 18, 256, "HOUSE_ROOFTILE"))
    for x, y in ((HX0, HY0 - 8), (HX1 - 8, HY0 - 8), (HX0, HY1), (HX1 - 8, HY1)):     # on the front/back faces
        D.append(box(x, y, 0, x + 8, y + 8, 228, "HOUSE_COPPER"))
    # chimney
    W.append(box(HX0, 384, TOP, HX0 + 48, 448, 296, S))
    D.append(box(HX0 - 4, 380, 296, HX0 + 52, 452, 304, "HOUSE_COPPER"))

    # living room: fireplace (open firebox), couch, coffee table, rug
    W.append(box(-384, 336, FLOOR, -352, 384, CEIL, P))
    W.append(box(-384, 448, FLOOR, -352, 496, CEIL, P))
    W.append(box(-384, 384, 96, -352, 448, CEIL, P))
    D.append(box(-352, 352, FLOOR, -320, 480, FLOOR + 4, "HOUSE_BRICK"))
    D.append(box(-388, 384, FLOOR, -384, 448, 96, "HOUSE_BRICK"))
    ctx.point("env_sprite", -372, 416, FLOOR + 2, model="sprites/casaroble_fire.spr", rendermode="5",
              renderamt="255", framerate="10", scale="0.45", spawnflags="1")
    ctx.point("light", -360, 416, 44, _light="255 140 50 140")
    D.append(box(-280, 496, FLOOR, -136, 544, 44, "HOUSE_COUCH"))
    D.append(box(-280, 544, FLOOR, -136, 560, 80, "HOUSE_COUCH"))
    D.append(box(-248, 400, FLOOR, -168, 448, 36, "HOUSE_TABLE"))
    D.append(box(-304, 360, FLOOR, -112, 480, FLOOR + 1, "HOUSE_RUG"))

    # dining: table and a sideboard
    D.append(box(48, 352, FLOOR, 176, 464, 48, "HOUSE_TABLE"))
    D.append(box(-36, 288, FLOOR, 0, 400, 56, "HOUSE_TABLE"))

    # kitchen: counters on the east wall with the range, fridge, breakfast bar facing the dining room
    D.append(box(208, 600, FLOOR, 256, 680, 56, "HOUSE_COUNTER"))
    D.append(box(208, 680, FLOOR, 256, 728, 58, "HOUSE_METAL"))
    D.append(box(208, 728, FLOOR, 256, 832, 56, "HOUSE_COUNTER"))
    D.append(box(200, 832, FLOOR, 256, 888, 144, "HOUSE_FRIDGE"))
    D.append(box(64, 568, FLOOR, 256, 600, 64, "HOUSE_COUNTER"))

    # bedroom 2, bedroom 3, primary (bed head against the east wall)
    D.append(box(-376, 592, FLOOR, -280, 720, 44, "HOUSE_BED2"))
    D.append(box(-376, 1072, FLOOR, -280, 1192, 44, "HOUSE_BED2"))
    D.append(box(-56, 976, FLOOR, 24, 1088, 44, "HOUSE_BED"))
    D.append(box(8, 976, 44, 24, 1088, 80, "HOUSE_TABLE"))
    # hall bath: tub and vanity; ensuite: vanity; hall: stacked washer/dryer
    D.append(box(-200, 592, FLOOR, -72, 640, 48, "HOUSE_WHITE"))
    D.append(box(-200, 690, FLOOR, -176, 760, 56, "HOUSE_WHITE"))
    D.append(box(-200, 1000, FLOOR, -144, 1024, 56, "HOUSE_WHITE"))
    D.append(box(-384, 816, FLOOR, -344, 880, 144, "HOUSE_WHITE"))

    # family room: sofa facing a TV on the east wall, rug
    D.append(box(80, 960, FLOOR, 128, 1104, 44, "HOUSE_COUCH"))
    D.append(box(64, 960, FLOOR, 80, 1104, 80, "HOUSE_COUCH"))
    D.append(box(248, 992, 64, 256, 1072, 112, "HOUSE_TV"))
    D.append(box(144, 960, FLOOR, 224, 1104, FLOOR + 1, "HOUSE_RUG"))

    ctx.brush_entity("func_detail", D, zhlt_detaillevel="1")

    for x, y in ((-216, 420), (108, 400), (108, 736), (-304, 688), (-132, 700), (-200, 848),
                 (-304, 1050), (-30, 1120), (-168, 964), (152, 1050)):
        ctx.point("light", x, y, 150, _light="255 228 190 120")

    z = FLOOR + 37
    for x, y, a in ((-300, 320, 0), (200, 320, 180), (0, 700, 90), (-250, 760, 0), (-280, 960, 0),
                    (-170, 1120, 0), (200, 950, 90)):
        ctx.point("info_player_deathmatch", x, y, z, angles="0 %d 0" % a)
    zi = FLOOR + 10
    ctx.point("weapon_shotgun", 110, 520, zi)            # strong weapons sit away from spawns (PR #67 references)
    ctx.point("weapon_9mmAR", 80, 720, zi)
    ctx.point("ammo_9mmAR", 150, 1150, zi)
    ctx.point("weapon_357", -40, 940, zi)
    ctx.point("item_healthkit", -130, 720, zi)
    ctx.point("item_healthkit", -190, 930, zi)
    ctx.point("item_battery", -250, 620, zi)
    ctx.point("ammo_buckshot", -330, 1000, zi)
