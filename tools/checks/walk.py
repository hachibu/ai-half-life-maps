"""Walk a player-sized box along routes in the compiled world.

A route is (name, floor_z, [(x, y), ...]). The walker starts standing on floor_z, can step up at most
18 units (the engine's step height) and can drop any distance. Doors and breakables are entities, so
they are ignored: a route through a closed door passes if the space behind it is open.

Maps call run() from tests/routes.py:   from tools.checks.walk import run;  run(ROUTES)
"""
import sys

import numpy as np

from tools.bsp.reader import load

STEP = 18


class Walker:
    def __init__(self, bsp):
        self.m = load(bsp)

    def free(self, x, y, z):
        return self.m.player_free((x, y, z))

    def walk(self, floor_z, pts, step=4):
        z = floor_z + 36
        if not self.free(pts[0][0], pts[0][1], z):
            return False, "start blocked at (%.0f, %.0f)" % pts[0][:2], floor_z
        for a, b in zip(pts, pts[1:]):
            a, b = np.array(a, float), np.array(b, float)
            n = max(1, int(np.linalg.norm(b - a) / step))
            for i in range(1, n + 1):
                p = a + (b - a) * i / n
                for up in range(0, STEP + 1):
                    if self.free(p[0], p[1], z + up):
                        break
                else:
                    return False, "blocked at (%.0f, %.0f), floor %.0f" % (p[0], p[1], z - 36), z - 36
                z += up
                while self.free(p[0], p[1], z - 1) and z > -4000:
                    z -= 1
        return True, "", z - 36


def run(routes, bsp=None):
    bsp = bsp or sys.argv[1]
    w = Walker(bsp)
    bad = 0
    for name, floor_z, pts in routes:
        ok, why, end = w.walk(floor_z, pts)
        print("%-48s %s" % (name, ("OK   floor %5.0f -> %5.0f" % (floor_z, end)) if ok else "FAIL " + why))
        bad += not ok
    print("routes: %d  failed: %d" % (len(routes), bad))
    sys.exit(1 if bad else 0)
