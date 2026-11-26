"""Satisfaction: a 0-100 score for how a map plays, from thousands of sampled positions and moves.

    python -m tools.checks.satisfaction build/<map>/<map>.bsp [--json out.json]

Inspired by "scenarios + satisfaction" testing: instead of a few hand-written routes (which an agent could edit
to pass), sample the whole map. Standable points are found by dropping a player box down random columns; two
points are linked when the walker (18-unit steps, free drops) can actually move between them. Then measure:

  weapon_rush   spawns that reach a weapon or ammo within 15 s of running (4800 units of path)
  no_traps      reachable points that can still get back to a spawn's area (no one-way dead ends)
  items         items that a player can reach from some spawn
  spawn_links   spawn pairs where each can reach the other

score = 100 * mean of the four. The sample is seeded by SATISFACTION_SEED (a CI secret, so agents can't tune
a map to the exact sample CI uses); locally the seed is 0. The pipeline fails a PR that lowers a map's score
by more than TOLERANCE against main's score with the same seed.
"""
import heapq
import json
import math
import os
import sys
import time

import numpy as np

from tools.checks.walk import Walker

TOLERANCE = 2.0
RUN_UNITS = 4800          # ~15 s at 320 u/s


def columns(w, spacing, rng):
    """Jittered grid of columns (dense enough to pass 64-unit doorways); jitter comes from the seed."""
    mins = np.array(w.m.models[0][0:3], float); maxs = np.array(w.m.models[0][3:6], float)
    pts = []
    xs = np.arange(mins[0] + 24, maxs[0] - 24, spacing); ys = np.arange(mins[1] + 24, maxs[1] - 24, spacing)
    for gx in xs:
      for gy in ys:
        x, y = gx + rng.uniform(-0.3, 0.3) * spacing, gy + rng.uniform(-0.3, 0.3) * spacing
        z, prev = maxs[2] - 40, False
        while z > mins[2] + 36:
            f = w.free(x, y, z)
            if prev and not f:                       # just passed from air into solid: floor is above here
                zz = z + 6
                while zz > z and w.free(x, y, zz - 1):
                    zz -= 1
                pts.append((x, y, zz))
            prev = f; z -= 6
    return pts


def nav(w, pts, LINK):
    cell = {}
    for i, p in enumerate(pts):
        cell.setdefault((int(p[0] // LINK), int(p[1] // LINK)), []).append(i)
    edges = [[] for _ in pts]
    for i, p in enumerate(pts):
        cx, cy = int(p[0] // LINK), int(p[1] // LINK)
        for dx in (-1, 0, 1):
            for dy in (-1, 0, 1):
                for j in cell.get((cx + dx, cy + dy), []):
                    if j == i: continue
                    q = pts[j]; d = math.hypot(q[0] - p[0], q[1] - p[1])
                    if d > LINK or q[2] - p[2] > 4 * 18 + 8: continue
                    ok, _, end = w.walk(p[2] - 36, [(p[0], p[1]), (q[0], q[1])])
                    if ok and abs(end - (q[2] - 36)) <= 18:
                        edges[i].append((j, d))
    return edges


def dijkstra(edges, srcs, limit=float("inf")):
    dist = {s: 0.0 for s in srcs}; h = [(0.0, s) for s in srcs]
    while h:
        d, u = heapq.heappop(h)
        if d > dist.get(u, float("inf")) or d > limit: continue
        for v, w_ in edges[u]:
            nd = d + w_
            if nd < dist.get(v, float("inf")) and nd <= limit:
                dist[v] = nd; heapq.heappush(h, (nd, v))
    return dist


def nearest(pts, x, y, z, r=80, dz=60):
    best, bi = None, None
    for i, p in enumerate(pts):
        d = math.hypot(p[0] - x, p[1] - y)
        if d <= r and abs((p[2] - 36) - z) <= dz and (best is None or d < best):
            best, bi = d, i
    return bi


class Graph:
    pass


def graph(bsp, seed=None, samples=None):
    seed = int(os.environ.get("SATISFACTION_SEED", "0") or 0) if seed is None else seed
    rng = np.random.default_rng(seed)
    w = Walker(bsp)
    ents = w.m.entities
    spawns = [tuple(map(float, e["origin"].split())) for e in ents if e.get("classname", "").startswith("info_player_")]
    items = [(e["classname"], *map(float, e["origin"].split())) for e in ents
             if e.get("classname", "").startswith(("weapon_", "ammo_", "item_"))]
    mins = np.array(w.m.models[0][0:3], float); maxs = np.array(w.m.models[0][3:6], float)
    area = (maxs[0] - mins[0]) * (maxs[1] - mins[1])
    spacing = samples or max(28.0, math.sqrt(area / 9000))
    LINK = spacing * 1.75
    pts = columns(w, spacing, rng)
    base = len(pts)
    for s in spawns:                                    # spawn origins are player centers already
        z = s[2]
        while w.free(s[0], s[1], z - 1) and z > mins[2]: z -= 1
        pts.append((s[0], s[1], z))
    for c, x, y, z in items:                            # a standing point at every item
        zz = z + 36
        while w.free(x, y, zz - 1) and zz > mins[2]: zz -= 1
        if w.free(x, y, zz): pts.append((x, y, zz))
    edges = nav(w, pts, LINK)
    g = Graph(); g.w, g.pts, g.edges, g.base, g.spawns, g.items = w, pts, edges, base, spawns, items
    g.spawn_nodes = list(range(base, base + len(spawns)))
    g.point = lambda x, y, z: nearest(pts, x, y, z)
    g.class_nodes = lambda cls: [nearest(pts, x, y, z) for c, x, y, z in items if c == cls]
    return g


def score(bsp, seed=None, samples=None):
    t0 = time.time()
    g = graph(bsp, seed, samples)
    w, pts, edges, base, spawns, items = g.w, g.pts, g.edges, g.base, g.spawns, g.items
    sp = g.spawn_nodes
    reach = dijkstra(edges, sp)
    rev = [[] for _ in pts]
    for u, es in enumerate(edges):
        for v, d in es: rev[v].append((u, d))
    back = dijkstra(rev, sp)
    targets = [nearest(pts, x, y, z) for c, x, y, z in items if c.startswith(("weapon_", "ammo_"))]
    targets = [t for t in targets if t is not None]
    rush = []
    for s in sp:
        d = dijkstra(edges, [s], RUN_UNITS)
        rush.append(any(t in d for t in targets))
    item_pts = [nearest(pts, x, y, z) for c, x, y, z in items]
    links = []
    for i, s in enumerate(sp):
        d = dijkstra(edges, [s])
        links += [t in d for t in sp if t != s]
    m = dict(
        weapon_rush=float(np.mean(rush)) if rush else 1.0,
        no_traps=float(np.mean([u in back for u in reach])) if reach else 0.0,
        items=float(np.mean([p is not None and p in reach for p in item_pts])) if item_pts else 1.0,
        spawn_links=float(np.mean(links)) if links else 1.0)
    out = dict(score=round(100 * float(np.mean(list(m.values()))), 1), metrics={k: round(v, 3) for k, v in m.items()},
               samples=base, nodes=len(pts), spawns=len(spawns), items=len(items), seconds=round(time.time() - t0, 1))
    return out


if __name__ == "__main__":
    res = score(sys.argv[1])
    print("satisfaction %5.1f  %s  (%d sampled points, %.0fs)" % (res["score"], " ".join(
        "%s=%.2f" % kv for kv in res["metrics"].items()), res["samples"], res["seconds"]))
    if "--json" in sys.argv:
        open(sys.argv[sys.argv.index("--json") + 1], "w").write(json.dumps(res, indent=1))
