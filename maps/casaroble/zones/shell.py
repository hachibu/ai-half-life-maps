"""Zone 'shell': the sky box around the whole lot and the afternoon sun. No BOUNDS: it wraps every other zone."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from _helpers import box
from tools.mapkit import textures as L

TEXTURES = {"sky": L.T["sky"]}                 # lowercase, or the game crashes
X0, Y0, Z0, X1, Y1, Z1 = -480, -400, -16, 480, 1856, 624     # the playable box every other zone sits inside


def build(ctx):
    t = 16
    for b in [(X0 - t, Y0 - t, Z1, X1 + t, Y1 + t, Z1 + t), (X0 - t, Y0 - t, Z0 - t, X1 + t, Y1 + t, Z0),
              (X0 - t, Y0 - t, Z0, X0, Y1 + t, Z1), (X1, Y0 - t, Z0, X1 + t, Y1 + t, Z1),
              (X0, Y0 - t, Z0, X1, Y0, Z1), (X0, Y1, Z0, X1, Y1 + t, Z1)]:
        ctx.brush(box(*b, "sky"))
    # warm late-afternoon sun from the south-west, soft blue fill from the sky
    ctx.point("light_environment", 0, -300, 560, pitch="-42", angles="0 60 0",
              _light="255 228 190 105", _diffuse_light="150 175 220 70")
