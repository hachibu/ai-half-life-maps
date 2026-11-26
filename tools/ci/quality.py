"""Quality ratchet for CI: a map's satisfaction score and holdout pass rate may not drop below main's.

    python -m tools.ci.quality measure MAP            -> build/MAP/quality.json  (satisfaction + holdout)
    python -m tools.ci.quality compare ARTIFACTS BASELINE.json   (PR: exit 1 on a regression)
    python -m tools.ci.quality baseline ARTIFACTS OUT.json       (main: write the new baseline)

ARTIFACTS holds one folder per map containing quality.json (the CI artifact layout).
"""
import json
import os
import subprocess
import sys
from pathlib import Path

from tools.checks import holdout, satisfaction


SATISFY = Path("build/bin/satisfy")       # Go port of tools/checks/satisfaction.py (`make bspc`); same scores


def satisfaction_score(bsp):
    if SATISFY.exists():
        out = Path(bsp).with_suffix(".satisfaction.json")
        subprocess.run([str(SATISFY), bsp, "--json", str(out)], check=True)
        return json.loads(out.read_text())
    return satisfaction.score(bsp)


def measure(name):
    bsp = "build/%s/%s.bsp" % (name, name)
    q = dict(map=name, satisfaction=satisfaction_score(bsp), holdout=holdout.run(name, bsp))
    Path("build/%s/quality.json" % name).write_text(json.dumps(q, indent=1))
    s = q["satisfaction"]
    print("%s satisfaction %.1f %s | holdout %s" % (name, s["score"], s["metrics"],
          q["holdout"].get("rate") if q["holdout"].get("rate") is not None else q["holdout"].get("note")))


def _load(artifacts):
    out = {}
    for p in sorted(Path(artifacts).glob("*/quality.json")):
        q = json.loads(p.read_text()); out[q["map"]] = q
    return out


def compare(artifacts, baseline):
    cur = _load(artifacts)
    base = json.loads(Path(baseline).read_text()) if Path(baseline).exists() else {}
    bad, lines = [], ["| map | satisfaction (main → PR) | holdout (main → PR) |", "|---|---|---|"]
    for m, q in cur.items():
        b = base.get(m, {})
        s0, s1 = b.get("satisfaction"), q["satisfaction"]["score"]
        h0, h1 = b.get("holdout"), q["holdout"].get("rate")
        if s0 is not None and s1 < s0 - satisfaction.TOLERANCE:
            bad.append("%s satisfaction dropped %.1f -> %.1f (%s)" % (m, s0, s1, q["satisfaction"]["metrics"]))
        if h0 is not None and h1 is not None and h1 < h0:
            bad.append("%s holdout pass rate dropped %.1f%% -> %.1f%%" % (m, h0, h1))
        lines.append("| %s | %s → %.1f | %s → %s |" % (m, "%.1f" % s0 if s0 is not None else "new", s1,
                                                     "%s%%" % h0 if h0 is not None else "-", "%s%%" % h1 if h1 is not None else "-"))
    summary = "\n".join(["## Quality ratchet", ""] + lines + [""] + ["❌ " + b for b in bad])
    print(summary)
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as f: f.write(summary + "\n")
    return 1 if bad else 0


def baseline(artifacts, out):
    old = json.loads(Path(out).read_text()) if Path(out).exists() else {}
    for m, q in _load(artifacts).items():
        old[m] = dict(satisfaction=q["satisfaction"]["score"], holdout=q["holdout"].get("rate"),
                      metrics=q["satisfaction"]["metrics"])
    Path(out).parent.mkdir(parents=True, exist_ok=True)
    Path(out).write_text(json.dumps(old, indent=1, sort_keys=True))
    print("baseline:", json.dumps(old))


if __name__ == "__main__":
    cmd, *a = sys.argv[1:]
    sys.exit({"measure": measure, "compare": compare, "baseline": baseline}[cmd](*a) or 0)
