"""Every spawn must stand clear and on the ground; every pickup must be out of solid.

Usage: python -m tools.checks.placements MAP.bsp
Spawns (info_player_*) and pickups (weapon_*, ammo_*, item_*) are read from the compiled map itself.
"""
import sys

from tools.bsp.reader import load


def main(bsp):
    m = load(bsp)
    head1 = m.world_heads[1]
    bad = spawns = items = 0
    for e in m.entities:
        cls = e.get("classname", "")
        if "origin" not in e:
            continue
        x, y, z = map(float, e["origin"].split())
        if cls.startswith("info_player_"):
            spawns += 1
            free = m.hullc(head1, (x, y, z)) == -1
            grounded = m.hullc(head1, (x, y, z - 20)) == -2
            if not (free and grounded):
                bad += 1
                print("BAD SPAWN", cls, x, y, z, "free" if free else "IN SOLID", "grounded" if grounded else "FLOATING")
        elif cls.startswith(("weapon_", "ammo_", "item_")):
            items += 1
            if m.solid((x, y, z)):
                bad += 1
                print("ITEM IN SOLID", cls, x, y, z)
    print("spawns: %d  items: %d  placement problems: %d" % (spawns, items, bad))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1]))
