"""Check a compiled BSP is valid and inside every engine limit.  Usage: python -m tools.checks.structure MAP.bsp"""
import sys

from tools.bsp.reader import LIMITS, load


def main(path):
    m = load(path)
    for label, key, lim in LIMITS:
        v = m.counts[key]
        print("  %-14s %8d / %-7d %s" % (label, v, lim, "OK" if v < lim else "OVER"))
    errs = sorted(set(m.errors))
    print("structure errors:", len(errs), errs[:10])
    return 1 if errs else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1]))
