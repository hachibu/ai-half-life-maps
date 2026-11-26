"""Python client for the Go renderer's frame server (build/bin/render serve; `make tools` builds it).

Scripts that make videos keep their camera paths and overlays in Python and get frames from Go:

    r = Renderer("build/prairieloop/prairieloop.bsp", asset_root="build/prairieloop/valve")
    r.minfo[mi], r.mtrans[mi]             # entity keys and origin per brush model (like the old renderer)
    img = r.frame(eye, target, t, W, H, fov, dyn)   # numpy (H, W, 3) uint8; dyn = {model: (R 3x3, T 3)}
    r.solid((x, y, z))                    # point inside solid?
"""
import json
import subprocess
from pathlib import Path

import numpy as np

RENDER = Path(__file__).resolve().parents[2] / "build" / "bin" / "render"


class Renderer:
    def __init__(self, bsp, asset_root):
        if not RENDER.exists():
            raise SystemExit("build/bin/render is missing: run `make tools` first")
        self.p = subprocess.Popen([str(RENDER), "serve", str(bsp), str(asset_root)],
                                  stdin=subprocess.PIPE, stdout=subprocess.PIPE)
        info = json.loads(self.p.stdout.readline())
        self.minfo = {int(k): v for k, v in info["models"].items()}
        self.mtrans = {int(k): (np.eye(3), np.array(v, float)) for k, v in info["origins"].items()}

    def _ask(self, req, nbytes):
        self.p.stdin.write((json.dumps(req) + "\n").encode())
        self.p.stdin.flush()
        data = self.p.stdout.read(nbytes)
        if len(data) != nbytes:
            raise RuntimeError("renderer stopped (exit %s)" % self.p.poll())
        return data

    def frame(self, eye, target, t, W_=854, H_=480, fov=85, dyn=None):
        d = {str(mi): [float(x) for x in np.asarray(R, float).reshape(9)] + [float(x) for x in np.asarray(T, float)]
             for mi, (R, T) in (dyn or {}).items()}
        req = {"frame": {"eye": [float(x) for x in eye], "target": [float(x) for x in target], "t": float(t),
                         "w": int(W_), "h": int(H_), "fov": float(fov), "dyn": d}}
        return np.frombuffer(self._ask(req, W_ * H_ * 3), np.uint8).reshape(H_, W_, 3)

    def solid(self, p):
        return self._ask({"solid": [float(x) for x in p]}, 1) == b"\x01"

    def close(self):
        if self.p.poll() is None:
            self.p.stdin.close()
            self.p.wait()
