"""Package a built map as a drop-in zip.  Usage: python -m tools.package MAP

build/<MAP>/valve/ is laid out exactly like the game's valve/ folder (maps/, gfx/env/, sprites/, sound/).
This copies the compiled .bsp into it, writes maps/<MAP>.res so servers send the custom files to
joining players, and zips valve/ + the map's README into build/<MAP>/<MAP>.zip.
"""
import shutil
import sys
import zipfile

from tools.paths import REPO, build_dir


def main(name):
    bd = build_dir(name)
    valve = bd / "valve"
    (valve / "maps").mkdir(parents=True, exist_ok=True)
    shutil.copy(bd / (name + ".bsp"), valve / "maps" / (name + ".bsp"))
    extras = sorted(p.relative_to(valve).as_posix() for sub in ("gfx", "sprites", "sound") if (valve / sub).exists()
                    for p in (valve / sub).rglob("*") if p.is_file())
    res = valve / "maps" / (name + ".res")
    if extras:
        res.write_text("// %s custom content (auto-downloaded by clients)\n" % name + "\n".join(extras) + "\n")
    elif res.exists():
        res.unlink()
    zpath = bd / (name + ".zip")
    with zipfile.ZipFile(zpath, "w", zipfile.ZIP_DEFLATED) as z:
        for p in sorted(valve.rglob("*")):
            if p.is_file():
                z.write(p, p.relative_to(bd).as_posix())
        readme = REPO / "maps" / name / "README.md"
        if readme.exists():
            z.write(readme, "README.md")
    print("packaged %s  (%d custom files)  -> %s" % (name, len(extras), zpath.relative_to(REPO)))


if __name__ == "__main__":
    main(sys.argv[1])
