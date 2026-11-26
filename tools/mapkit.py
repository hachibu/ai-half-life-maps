"""Zone-based map assembly, designed so several agents can work on one map without merge conflicts.

A map is a folder of independent zone files: maps/<map>/zones/*.py, discovered automatically (sorted by
file name). Adding a zone never edits a shared file. Each zone module may define:

    BOUNDS   = (x0, y0, z0, x1, y1, z1)    # space this zone may build in; zones must not overlap
    TEXTURES = {"NAME": fn -> PIL.Image}   # textures this zone adds (names must be unique across zones)
    ROUTES   = [(name, floor_z, [(x, y), ...]), ...]
    VIEWS    = [(name, eye, target), ...]
    def build(ctx): ...                    # add brushes / entities through ctx

The map's build.py is three lines (see maps/_template/build.py):

    from tools.mapkit import build_map
    build_map(__file__, worldspawn={...})

Conflict rules enforced here: duplicate texture names, duplicate targetnames across zones, and
overlapping BOUNDS all fail the build with a message naming both zones.
"""
import importlib.util
import struct
from pathlib import Path

from tools.assets import library as L
from tools.paths import build_dir


def load_zones(map_dir):
    zones = []
    for p in sorted((Path(map_dir) / "zones").glob("*.py")):
        if p.name.startswith("_"):
            continue
        spec = importlib.util.spec_from_file_location("zone_" + p.stem, p)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        mod.ZONE = p.stem
        zones.append(mod)
    return zones


class Ctx:
    """What a zone's build(ctx) uses. Brush text is Valve 220 (see tools/assets + existing maps for helpers)."""

    def __init__(self, zone):
        self.zone = zone
        self.world = []        # brush strings added to worldspawn
        self.entities = []     # full entity strings (point or brush entities)
        self.targetnames = set()

    def name(self, short):
        """Zone-prefixed targetname, e.g. ctx.name('door') -> 'hall_door'. Use for every targetname."""
        n = "%s_%s" % (self.zone, short)
        self.targetnames.add(n)
        return n

    def brush(self, text):
        self.world.append(text)

    def point(self, cls, x, y, z, **kv):
        if "targetname" in kv:
            self.targetnames.add(kv["targetname"])
        self.entities.append('{\n"classname" "%s"\n"origin" "%g %g %g"\n' % (cls, x, y, z) +
                             "".join('"%s" "%s"\n' % (k, v) for k, v in kv.items()) + "}\n")

    def brush_entity(self, cls, brushes, **kv):
        if "targetname" in kv:
            self.targetnames.add(kv["targetname"])
        self.entities.append('{\n"classname" "%s"\n' % cls + "".join('"%s" "%s"\n' % (k, v) for k, v in kv.items()) +
                             "".join(brushes) + "}\n")


def _overlap(a, b):
    return all(a[i] < b[i + 3] and b[i] < a[i + 3] for i in range(3))


def check_zones(zones):
    errors, owner = [], {}
    for z in zones:
        for t in getattr(z, "TEXTURES", {}):
            if t in owner:
                errors.append("texture %s defined by zones %s and %s" % (t, owner[t], z.ZONE))
            owner[t] = z.ZONE
    bounded = [z for z in zones if hasattr(z, "BOUNDS")]
    for i, a in enumerate(bounded):
        for b in bounded[i + 1:]:
            if _overlap(a.BOUNDS, b.BOUNDS):
                errors.append("zones %s and %s overlap: %s vs %s" % (a.ZONE, b.ZONE, a.BOUNDS, b.BOUNDS))
    return errors


def write_wad(path, textures):
    body, entries, pos = b"", [], 12
    for nm, fn in textures.items():
        data = L.to_mip(nm, fn())
        entries.append(struct.pack("<IIIbbH16s", pos, len(data), len(data), 0x43, 0, 0, nm.encode().ljust(16, b"\0")[:16]))
        body += data
        pos += len(data)
    with open(path, "wb") as f:
        f.write(b"WAD3" + struct.pack("<II", len(entries), pos) + body + b"".join(entries))


def build_map(build_file, worldspawn=None):
    map_dir = Path(build_file).resolve().parent
    name = map_dir.name
    zones = load_zones(map_dir)
    errors = check_zones(zones)
    ctxs = []
    for z in zones:
        c = Ctx(z.ZONE)
        if hasattr(z, "build"):
            z.build(c)
        ctxs.append(c)
    seen = {}
    for c in ctxs:
        for t in c.targetnames:
            if t in seen:
                errors.append("targetname %s used by zones %s and %s" % (t, seen[t], c.zone))
            seen[t] = c.zone
    if errors:
        raise SystemExit("zone conflicts:\n  " + "\n  ".join(errors))
    textures = {}
    for z in zones:
        textures.update(getattr(z, "TEXTURES", {}))
    out = build_dir(name)
    write_wad(out / (name + ".wad"), textures)
    ws = dict({"mapversion": "220", "wad": name + ".wad"}, **(worldspawn or {}))
    world = '{\n"classname" "worldspawn"\n' + "".join('"%s" "%s"\n' % kv for kv in ws.items()) + \
            "".join(b for c in ctxs for b in c.world) + "}\n"
    with open(out / (name + ".map"), "w", newline="\r\n") as f:
        f.write(world + "".join(e for c in ctxs for e in c.entities) + texlights(out / (name + ".wad"), L.EMISSIVE))
    print("%s: %d zones (%s), %d brushes, %d entities, %d textures" % (
        name, len(zones), ", ".join(z.ZONE for z in zones), sum(len(c.world) for c in ctxs),
        sum(len(c.entities) for c in ctxs), len(textures)))
    return out


def zone_routes(map_dir):
    return [r for z in load_zones(map_dir) for r in getattr(z, "ROUTES", [])]


def zone_views(map_dir):
    return [v for z in load_zones(map_dir) for v in getattr(z, "VIEWS", [])]


# ------------------------------------------------------------------ compiler-friendly output (SDHLT)
# Shared by the hand-written maps (battleaxecombatleague, prairieloop): they build lists of brush text and call
# these just before writing the .map.

def brush_bbox(text):
    """Bounding box (mins, maxs) of one brush, from its plane lines ("( x y z ) ( x y z ) ( x y z ) TEX ...")."""
    import itertools
    import re
    import numpy as np
    planes = []
    for line in text.splitlines():
        m = re.match(r"\s*\(([^)]*)\)\s*\(([^)]*)\)\s*\(([^)]*)\)", line)
        if not m:
            continue
        p0, p1, p2 = (np.array([float(v) for v in g.split()]) for g in m.groups())
        n = np.cross(p0 - p1, p2 - p1)
        n /= np.linalg.norm(n)
        planes.append((n, n @ p1))
    pts = []
    for (a, da), (b, db), (c, dc) in itertools.combinations(planes, 3):
        A = np.array([a, b, c])
        if abs(np.linalg.det(A)) < 1e-6:
            continue
        p = np.linalg.solve(A, [da, db, dc])
        if all(n @ p <= d + 0.01 for n, d in planes):
            pts.append(p)
    pts = np.array(pts)
    return pts.min(axis=0), pts.max(axis=0)


def split_detail(brushes, span=256):
    """Split world brushes into (structural, detail).

    Structural brushes are wall-, floor- and roof-sized: at least two of their three extents are >= span. They seal
    the map and cut it into the spaces VIS reasons about. Everything smaller (steps, seats, trim, props, lamps)
    is detail: still solid, still lit and casting shadows, but VIS ignores it, which is what keeps full VIS fast.
    A detail brush can't seal the map: if one was needed, hlbsp reports a leak and the build fails.
    """
    structural, detail = [], []
    for b in brushes:
        mn, mx = brush_bbox(b)
        (structural if sum(e >= span for e in mx - mn) >= 2 else detail).append(b)
    return structural, detail


def detail_entity(brushes):
    """One func_detail entity holding the given brushes (empty string if there are none)."""
    if not brushes:
        return ""
    return '{\n"classname" "func_detail"\n"zhlt_detaillevel" "1"\n' + "".join(brushes) + "}\n"


def chunks(brushes, cell=1024):
    """Group brushes by the cell-sized XY square their centre falls in, so one big decorative entity becomes
    several small ones (the engine checks a brush entity's visibility by its bounding box)."""
    groups = {}
    for b in brushes:
        mn, mx = brush_bbox(b)
        c = (mn + mx) / 2
        groups.setdefault((int(c[0] // cell), int(c[1] // cell)), []).append(b)
    return [groups[k] for k in sorted(groups)]


def texlights(wad_path, emissive, scale=2.0):
    """info_texlights entity: each texture in `emissive` (name -> 0..255 glow) emits light in its own average
    colour, brightness glow * scale. Colours come from the texture's pixels in the map's .wad."""
    data = Path(wad_path).read_bytes()
    num, ofs = struct.unpack_from("<4xii", data, 0)
    avg = {}
    for i in range(num):
        pos, _disk, _size, _typ = struct.unpack_from("<iiib", data, ofs + 32 * i)
        name = data[ofs + 32 * i + 16: ofs + 32 * i + 32].split(b"\0")[0].decode("latin-1").upper()
        if name not in {k.upper() for k in emissive}:
            continue
        w, h = struct.unpack_from("<ii", data, pos + 16)
        off0 = struct.unpack_from("<i", data, pos + 24)[0]
        pal_at = pos + 40 + w * h * 85 // 64 + 2
        pal = data[pal_at: pal_at + 768]
        px = data[pos + off0: pos + off0 + w * h]
        tot = [0, 0, 0]
        for p in px:
            for c in range(3):
                tot[c] += pal[3 * p + c]
        avg[name] = [t / len(px) for t in tot]
    lines = []
    for name, glow in emissive.items():
        rgb = avg.get(name.upper())
        if rgb is None:
            continue
        top = max(rgb) or 1
        lines.append('"%s" "%d %d %d %d"\n' % (name, *(round(255 * c / top) for c in rgb), round(glow * scale)))
    return '{\n"classname" "info_texlights"\n' + "".join(lines) + "}\n" if lines else ""
