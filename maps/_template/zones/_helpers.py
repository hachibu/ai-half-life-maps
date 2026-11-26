"""Shared helpers for this map's zones (files starting with _ are not zones). Keep it small and stable."""


def box(x0, y0, z0, x1, y1, z1, tex):
    """Axis-aligned brush in Valve 220 format."""
    def face(p0, p1, p2, u, v):
        f3 = lambda p: "%g %g %g" % p
        return "( %s ) ( %s ) ( %s ) %s [ %s 0 ] [ %s 0 ] 0 1 1" % (f3(p0), f3(p1), f3(p2), tex, u, v)
    return "{\n" + "\n".join([
        face((x0, y0, z1), (x0, y1, z1), (x1, y0, z1), "1 0 0", "0 -1 0"), face((x0, y0, z0), (x1, y0, z0), (x0, y1, z0), "1 0 0", "0 -1 0"),
        face((x0, y1, z0), (x1, y1, z0), (x0, y1, z1), "1 0 0", "0 0 -1"), face((x0, y0, z0), (x0, y0, z1), (x1, y0, z0), "1 0 0", "0 0 -1"),
        face((x1, y0, z0), (x1, y0, z1), (x1, y1, z0), "0 1 0", "0 0 -1"), face((x0, y0, z0), (x0, y1, z0), (x0, y0, z1), "0 1 0", "0 0 -1")]) + "\n}\n"


def stairs(x0, y0, x1, y1, z0, z1, tex, rise=8, axis="+y"):
    """Brushes for a straight staircase filling (x0, y0)-(x1, y1), climbing from floor z0 to z1 in the direction
    `axis` ("+x", "-x", "+y" or "-y"). Risers are `rise` units (8 looks best in GoldSrc; never more than 16, the
    walker and players step at most 18). The last step's top is z1, so it meets the floor above flush."""
    n = max(1, -(-(z1 - z0) // rise))
    lo, hi = (x0, x1) if axis[1] == "x" else (y0, y1)
    out = []
    for i in range(n):
        a, b = lo + (hi - lo) * i / n, lo + (hi - lo) * (i + 1) / n
        if axis[0] == "-":
            a, b = hi - (b - lo), hi - (a - lo)
        top = min(z1, z0 + rise * (i + 1))
        out.append(box(a, y0, z0, b, y1, top, tex) if axis[1] == "x" else box(x0, a, z0, x1, b, top, tex))
    return out
