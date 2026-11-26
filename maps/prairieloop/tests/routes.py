"""Walk routes that must stay passable.  Run: make test MAP=prairieloop"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from tools.mapkit.routes import run


def N(cx, x, y): return (cx + x, 1056 + y)          # north-row house: local -> world
def S_(cx, x, y): return (cx - x, -1056 - y)        # south-row house: local -> world
def house_down(conv, cx):
    return [conv(cx, -20, -170), conv(cx, -20, 40), conv(cx, 120, 60), conv(cx, 222, 168)]
def basement_out(conv, cx, sx=0):
    return [conv(cx, -100, 168), conv(cx, -100, -100), conv(cx, sx, -150), conv(cx, sx, -260)]


cxN = cxS = -384
cb = 384   # the blue house
ROUTES = [
    # in through one house, under the street, up through another
    ("street -> N1 house -> stairs down", 0, [(-300, 760), N(cxN, 0, -240)] + house_down(N, cxN)),
    ("N1 basement -> ring tunnel -> S1 basement", -176, basement_out(N, cxN) + [(cxN, 576), (-1024, 576), (-1024, -576), (cxS, -576),
                                                                              S_(cxS, 0, -260), S_(cxS, 0, -150), S_(cxS, -100, -100), S_(cxS, -100, 168)]),
    ("S1 basement -> stairs up -> street", -176, [S_(cxS, -80, 168), S_(cxS, 222, 168), S_(cxS, 120, 60), S_(cxS, -20, 40),
                                                  S_(cxS, -20, -170), S_(cxS, 0, -240), (-300, -760)]),
    ("corner house N0 -> ring tunnel", -176, basement_out(N, -1152) + [(-1152, 576), (-1024, 576)]),
    ("corner house S3 -> ring tunnel", -176, basement_out(S_, 1152) + [(1152, -576), (1024, -576)]),
    # back over the roofs
    ("snowbank -> S0 roof", 0, [(-1800, -1176), (-1700, -1176), (-1420, -1176), (-1300, -1100)]),
    ("roof loop: S row -> garages -> east -> N row", 144, [(-1300, -1056), (-1152, -1056), (1152, -1056), (1344, -1056), (1344, -900),
                                                          (1344, -760), (1344, -600), (1440, -576), (1440, -350), (1440, 0), (1440, 350),
                                                          (1440, 576), (1344, 576), (1344, 760), (1344, 900), (1344, 1056), (-1152, 1056), (-1300, 1056)]),
    ("N0 roof -> exterior stairs -> ground", 144, [(-1350, 1180), (-1440, 1210), (-1440, 1100), (-1440, 980), (-1440, 900)]),
    ("east house exterior stairs up", 0, [(1720, -288), (1690, -288), (1490, -288), (1490, -220)]),
    # the blue house reveal and the hall
    ("street -> blue house -> computer room", 0, [(300, 740), N(cb, 0, -240)] + house_down(N, cb) + [N(cb, -100, 168), N(cb, -100, 0)]),
    ("computer room -> reveal stairs -> balcony", -176, [N(cb, -100, 0), N(cb, 128, -150), (512, 840), (512, 700), (512, 500), (512, 330)]),
    ("balcony -> stairs -> hall floor", -320, [(512, 330), (420, 336), (250, 336), (200, 300), (0, 200)]),
    ("hall -> BattleAxe stairs -> ring tunnel", -464, [(0, 20), (-700, 20), (-700, 345), (-792, 345), (-792, 60), (-792, 0), (-900, 0), (-1024, 0)]),
    ("hall -> Crescent stairs -> ring tunnel", -464, [(0, 20), (700, -20), (700, -345), (792, -345), (792, -60), (792, 0), (900, 0), (1024, 0)]),
    ("ring tunnel -> east house basement", -176, [(1024, 0), (1150, 0), (1270, 0), (1340, 0)]),
    # interactive spots (doors/breakables are entities: the space behind them must be open)
    ("hall -> server room", -464, [(-560, -200), (-560, -360), (-560, -420), (-560, -500)]),
    ("road -> north garage -> breaker 1", 0, [(1200, 576), (1270, 576), (1330, 576), (1540, 580)]),
    ("south tunnel -> breaker 2", -176, [(-300, -576), (0, -576), (0, -610)]),
    ("west street -> breaker 3", 0, [(-2500, 60), (-2900, 60), (-2915, 200)]),
    ("back yard -> steam vent", 0, [(384, -1400), (384, -1316)]),
    ("tunnel -> health charger", -176, [(-1024, -200), (-1040, -396)]),
    ("tunnel -> armor charger", -176, [(1024, -200), (1040, -396)]),
    ("hall -> stage -> teleport pad", -464, [(250, -200), (250, -268), (255, -330)]),
    ("hall -> centre stage", -464, [(0, -200), (0, -268), (0, -320)]),
]

if __name__ == "__main__":
    run(ROUTES)
