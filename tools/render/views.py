"""Render a map's preview shots.  Usage: python -m tools.render.views MAP
Reads VIEWS = [(name, eye, target), ...] from maps/<MAP>/tests/views.py and writes
build/<MAP>/previews/<name>.png plus a contact sheet."""
import importlib.util
import sys

from PIL import Image

from tools.paths import REPO, build_dir
from tools.render.renderer import Renderer


def main(name, w=854, h=480):
    spec = importlib.util.spec_from_file_location("views", REPO / "maps" / name / "tests" / "views.py")
    mod = importlib.util.module_from_spec(spec); spec.loader.exec_module(mod)
    bd = build_dir(name); out = bd / "previews"; out.mkdir(exist_ok=True)
    r = Renderer(str(bd / (name + ".bsp")), asset_root=str(bd / "valve"))
    tiles = []
    for vname, eye, tgt in mod.VIEWS:
        im = Image.fromarray(r.frame(eye, tgt, 1.3, w, h, 80))
        im.save(out / (vname + ".png")); tiles.append(im.resize((w // 2, h // 2)))
        print("preview", vname, flush=True)
    cols = 3; rows = (len(tiles) + cols - 1) // cols
    sheet = Image.new("RGB", (cols * w // 2, rows * h // 2))
    for i, t in enumerate(tiles):
        sheet.paste(t, ((i % cols) * w // 2, (i // cols) * h // 2))
    sheet.save(out / "_sheet.png")


if __name__ == "__main__":
    main(sys.argv[1])
