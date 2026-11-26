"""Every target and multi_manager key must name an existing targetname.  Usage: python -m tools.checks.wiring MAP.bsp"""
import sys

from tools.bsp.reader import load

MM_SKIP = {"classname", "targetname", "origin", "spawnflags", "wait", "angles"}


def main(path):
    m = load(path)
    names = {}
    for e in m.entities:
        if "targetname" in e:
            names.setdefault(e["targetname"], []).append(e["classname"])
    bad = 0
    for e in m.entities:
        refs = [e["target"]] if e.get("target") else []
        if e["classname"] == "multi_manager":
            refs += [k.split("#")[0] for k in e if k not in MM_SKIP]
        for r in refs:
            if r not in names:
                bad += 1
                print("MISSING target %-16s <- %s" % (r, e["classname"]))
    print("named entities: %d  wiring problems: %d" % (len(names), bad))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1]))
