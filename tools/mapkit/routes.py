"""Routes that must stay walkable, checked by the Go walker (tools/walker, via build/bin/mapcheck).

A route is (name, floor_z, [(x, y), ...]). The walker starts standing on floor_z, can step up at most 18 units
(the engine's step height) and can drop any distance. Doors and breakables are entities, so they are ignored:
a route through a closed door passes if the space behind it is open.

Maps call run() from tests/routes.py:   from tools.mapkit.routes import run;  run(ROUTES)
  python maps/<map>/tests/routes.py build/<map>/<map>.bsp     check the routes (needs `make tools`)
  ROUTES_JSON=1 python maps/<map>/tests/routes.py             print the routes as JSON (what `make test` feeds Go)
"""
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

MAPCHECK = Path(__file__).resolve().parents[2] / "build" / "bin" / "mapcheck"


def as_json(routes):
    return json.dumps([[name, float(floor_z), [[float(x), float(y)] for x, y in pts]] for name, floor_z, pts in routes])


def run(routes, bsp=None):
    if os.environ.get("ROUTES_JSON"):
        print(as_json(routes))
        return
    bsp = bsp or sys.argv[1]
    if not MAPCHECK.exists():
        sys.exit("build/bin/mapcheck is missing: run `make tools` first")
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        f.write(as_json(routes))
    try:
        sys.exit(subprocess.run([str(MAPCHECK), "routes", bsp, f.name]).returncode)
    finally:
        os.unlink(f.name)
