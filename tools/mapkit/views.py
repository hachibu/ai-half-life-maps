"""Print a map's preview shots as JSON for the Go renderer (`make views` feeds this to build/bin/render).

    python -m tools.mapkit.views MAP        ->  [[name, [x, y, z] eye, [x, y, z] target], ...]

The shots themselves live with the map: VIEWS = [(name, eye, target), ...] in maps/<MAP>/tests/views.py.
"""
import importlib.util
import json
import sys

from tools.mapkit.paths import REPO


def views(name):
    spec = importlib.util.spec_from_file_location("views", REPO / "maps" / name / "tests" / "views.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return [[n, [float(x) for x in eye], [float(x) for x in tgt]] for n, eye, tgt in mod.VIEWS]


if __name__ == "__main__":
    print(json.dumps(views(sys.argv[1])))
