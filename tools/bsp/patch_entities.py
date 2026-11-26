"""Swap a compiled BSP's entity lump for the entities in a .map (same brush entities, same order)."""
import re, struct, sys
mapf, bspf = sys.argv[1], sys.argv[2]
src = open(mapf).read()
ents, depth, cur, has_brush = [], 0, None, False
for line in src.splitlines():
    line = line.strip()
    if line == "{":
        depth += 1
        if depth == 1: cur, has_brush = [], False
        elif depth == 2: has_brush = True
    elif line == "}":
        if depth == 1: ents.append((cur, has_brush))
        depth -= 1
    elif line.startswith('"') and depth == 1:
        cur.append(tuple(re.findall(r'"([^"]*)"', line)))
out, mi = [], 0
for i, (kv, hb) in enumerate(ents):
    kv = [p for p in kv if p[0] != "wad"]
    if i == 0: kv.append(("wad", ""))
    elif hb:
        mi += 1; kv.append(("model", "*%d" % mi))
    out.append("{\n" + "".join('"%s" "%s"\n' % p for p in kv) + "}\n")
lump0 = "".join(out).encode() + b"\0"
d = open(bspf, "rb").read()
L = [struct.unpack_from("<ii", d, 4 + 8 * i) for i in range(15)]
datas = [d[o:o + l] for o, l in L]; datas[0] = lump0
o2 = bytearray(struct.pack("<i", 30) + b"\0" * 120); info = []
for x in datas:
    while len(o2) % 4: o2.append(0)
    info.append((len(o2), len(x))); o2.extend(x)
struct.pack_into("<i30i", o2, 0, 30, *[v for p in info for v in p])
open(bspf, "wb").write(o2)
print("entities:", len(out), "brush models:", mi)
