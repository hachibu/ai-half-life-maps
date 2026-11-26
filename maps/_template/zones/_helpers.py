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
