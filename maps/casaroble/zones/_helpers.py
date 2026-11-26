"""Shared helpers for casaroble's zones (files starting with _ are not zones). Keep it small and stable."""

FLOOR = 16      # house, deck: floor top. Ground (yards, driveway, garage) is z = 0.
CEIL = 176      # house interior ceiling
DOOR = 128      # top of door openings in the house (floor 16 + 112)


def box(x0, y0, z0, x1, y1, z1, tex, scale=1, **faces):
    """Axis-aligned brush in Valve 220 format. `faces` overrides the texture per side: t(op), b(ottom),
    n(orth, +y), s(outh, -y), e(ast, +x), w(est, -x). `scale` > 1 means fewer light samples (big plain surfaces)."""
    def face(side, p0, p1, p2, u, v):
        f3 = lambda p: "%g %g %g" % p
        return "( %s ) ( %s ) ( %s ) %s [ %s 0 ] [ %s 0 ] 0 %g %g" % (
            f3(p0), f3(p1), f3(p2), faces.get(side, tex), u, v, scale, scale)
    return "{\n" + "\n".join([
        face("t", (x0, y0, z1), (x0, y1, z1), (x1, y0, z1), "1 0 0", "0 -1 0"),
        face("b", (x0, y0, z0), (x1, y0, z0), (x0, y1, z0), "1 0 0", "0 -1 0"),
        face("n", (x0, y1, z0), (x1, y1, z0), (x0, y1, z1), "1 0 0", "0 0 -1"),
        face("s", (x0, y0, z0), (x0, y0, z1), (x1, y0, z0), "1 0 0", "0 0 -1"),
        face("e", (x1, y0, z0), (x1, y0, z1), (x1, y1, z0), "0 1 0", "0 0 -1"),
        face("w", (x0, y0, z0), (x0, y1, z0), (x0, y0, z1), "0 1 0", "0 0 -1")]) + "\n}\n"


def wall(ctx, x0, y0, x1, y1, z0, z1, tex, gaps=(), glass=None, detail=None, **faces):
    """A wall from z0 to z1 with openings. The wall runs along its longer side; each gap is (a, b, gz0, gz1) along
    that side. A gap whose bottom is above z0 is a window: with `glass`, it gets a breakable pane."""
    along_x = (x1 - x0) >= (y1 - y0)
    lo, hi = (x0, x1) if along_x else (y0, y1)

    def piece(a, b, za, zb):
        if b <= a or zb <= za:
            return
        bx = (a, y0, za, b, y1, zb) if along_x else (x0, a, za, x1, b, zb)
        (detail if detail is not None else ctx.world).append(box(*bx, tex, **faces))

    cur = lo
    for a, b, gz0, gz1 in sorted(gaps):
        piece(cur, a, z0, z1)
        piece(a, b, z0, gz0)
        piece(a, b, gz1, z1)
        if glass and gz0 > z0:
            m = ((y0 + y1) / 2) if along_x else ((x0 + x1) / 2)
            pane = (a, m - 2, gz0, b, m + 2, gz1) if along_x else (m - 2, a, gz0, m + 2, b, gz1)
            ctx.brush_entity("func_breakable", [box(*pane, glass)], material="0", health="15",
                             rendermode="2", renderamt="110")
        cur = b
    piece(cur, hi, z0, z1)


def canopy(ctx, cx, cy, tex, clamp):
    """An oak canopy: overlapping leafy blocks high above the ground, as one func_illusionary (players never touch
    it). `clamp` = (x0, y0, x1, y1) keeps it inside the zone."""
    def c(x0, y0, x1, y1):
        return max(x0, clamp[0]), max(y0, clamp[1]), min(x1, clamp[2]), min(y1, clamp[3])
    parts = []
    for dx, dy, r, z0, z1 in ((0, 0, 176, 224, 304), (-56, 40, 120, 304, 352), (64, -48, 112, 296, 344),
                              (-96, -88, 88, 208, 256), (104, 96, 80, 216, 264)):
        x0, y0, x1, y1 = c(cx + dx - r, cy + dy - r, cx + dx + r, cy + dy + r)
        if x1 > x0 and y1 > y0:
            parts.append(box(x0, y0, z0, x1, y1, z1, tex))
    ctx.brush_entity("func_illusionary", parts)
