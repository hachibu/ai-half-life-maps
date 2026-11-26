"""Walk routes that must stay passable.  Run: make test MAP=battleaxecombatleague"""
import math
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from tools.mapkit.routes import run

T8 = math.tan(math.pi / 8)
def octv(a):
    b = a * T8
    return [(a, -b), (a, b), (b, a), (-b, a), (-a, b), (-a, -b), (-b, -a), (b, -a)]
def onoct(a, seg, t):
    v = octv(a); (px, py), (qx, qy) = v[seg], v[(seg + 1) % 8]
    return (px + (qx - px) * t, py + (qy - py) * t)

ROUTES = [
    ("red walkout: locker room -> cage", 24, [(0, -1870), (0, -1716), (0, -1300), (0, -400), (0, -300), (0, 0)]),
    ("blue walkout: locker room -> cage", 0, [(0, 1700), (0, 1476), (0, 900), (0, 400), (0, 300), (0, 0)]),
    ("floor -> lower aisle -> concourse", 0, [onoct(820, 0, 0.5), onoct(1900, 0, 0.5)]),
] + [
    ("concourse stairs -> balcony (corner %d)" % seg, 368,
     [onoct(1900, seg, 0.05), onoct(2008, seg, 0.06), onoct(2008, seg, 0.34), onoct(2040, seg, 0.34)]) for seg in (1, 3, 5, 7)
] + [
    ("balcony -> upper deck top row (side %d)" % seg, 608, [onoct(2030, seg, 0.5), onoct(2620, seg, 0.5)]) for seg in (0, 1, 2, 3, 4, 5, 7)
]

if __name__ == "__main__":
    run(ROUTES)
