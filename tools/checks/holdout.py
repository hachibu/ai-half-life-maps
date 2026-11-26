"""Holdout scenarios: checks the agents cannot see, written by humans, run only in CI.

The scenarios live encrypted in holdout/scenarios.enc. Only CI has the key (Actions secret HOLDOUT_KEY), so an
agent can't read them and tune a map to pass them. CI prints only a pass rate per map, never which
scenario failed. A PR may not lower a map's holdout pass rate.

Plain-JSON format before encryption (a list):
  {"map": "prairieloop", "name": "spawn to hall fast", "kind": "reach",
   "from": "spawns" | [x, y, z_floor], "to": "weapon_rpg" | [x, y, z_floor], "within": 4800}

  kind "reach": from every start (or the point), some target (an entity class, or a point) is reachable
                within `within` units of running (default: any distance).

The Go scorer (build/bin/satisfy, `make bspc`) evaluates them on its navigation graph. The decrypted scenarios
go to it over stdin and are never written to disk.

Humans:
  HOLDOUT_KEY=... make holdout-encrypt   (holdout/scenarios.json -> holdout/scenarios.enc; never commit the .json)
CI:
  python -m tools.checks.holdout MAP build/MAP/MAP.bsp [--json out.json]
"""
import json
import os
import subprocess
import sys
from pathlib import Path

ENC = Path("holdout/scenarios.enc")
SATISFY = Path("build/bin/satisfy")


def load_scenarios():
    key = os.environ.get("HOLDOUT_KEY")
    if not key or not ENC.exists():
        return None
    out = subprocess.run(["openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2", "-in", str(ENC), "-pass", "env:HOLDOUT_KEY"],
                         capture_output=True)
    if out.returncode != 0:
        raise SystemExit("holdout: cannot decrypt (wrong HOLDOUT_KEY?)")
    return json.loads(out.stdout)


def run(map_name, bsp):
    sc = load_scenarios()
    if sc is None:
        return dict(rate=None, n=0, note="no holdout scenarios (or no key): skipped")
    sc = [s for s in sc if s.get("map") == map_name]
    if not sc:
        return dict(rate=None, n=0, note="no holdout scenarios for this map")
    if not SATISFY.exists():
        raise SystemExit("holdout: %s is missing; run `make bspc`" % SATISFY)
    out = Path(bsp).with_suffix(".holdout.json")
    r = subprocess.run([str(SATISFY), bsp, "--holdout", "-", "--json", str(out)], input=json.dumps(sc).encode(),
                       capture_output=True)
    if r.returncode != 0:
        raise SystemExit("holdout: satisfy failed: %s" % r.stderr.decode().strip())   # names an index, never a scenario
    res = json.loads(out.read_text())
    return dict(rate=res["rate"], n=res["n"])


if __name__ == "__main__":
    res = run(sys.argv[1], sys.argv[2])
    print("holdout %s: %s" % (sys.argv[1], "%.1f%% of %d scenarios" % (res["rate"], res["n"]) if res["rate"] is not None else res["note"]))
    if "--json" in sys.argv:
        open(sys.argv[sys.argv.index("--json") + 1], "w").write(json.dumps(res))
