"""Publish preview screenshots to the `previews` branch and write the PR comment that shows them.

CI-only helper (see .github/workflows/).  Agents don't run this; they add views and run `make views`.

  python -m tools.ci.previews store   ARTIFACTS WORKTREE PREFIX
      Copy ARTIFACTS/<map>/previews/*.png into WORKTREE/PREFIX/<map>/*.jpg (replacing that folder).
      PREFIX is "main" for merges and "pr-<number>" for pull requests.

  python -m tools.ci.previews comment WORKTREE PR_PREFIX REPO REF RUN_URL > comment.md

  python -m tools.ci.previews gate WORKTREE PR_PREFIX "map1 map2"
      Visual gate: every map whose own files changed has at least one changed or new view, and no
      screenshot is blank (one flat colour). Exit 1 with reasons otherwise.
      Markdown comparing WORKTREE/main/<map> (before) with WORKTREE/PR_PREFIX/<map> (after),
      changed and new views first, unchanged ones folded away.
"""
import shutil
import sys
from pathlib import Path

import numpy as np
from PIL import Image

MARKER = "<!-- map-previews -->"
CHANGED = 0.5          # mean absolute pixel difference (0-255) that counts as a visible change


def store(artifacts, worktree, prefix):
    for prev in sorted(Path(artifacts).glob("*/previews")):
        name = prev.parent.name
        dest = Path(worktree) / prefix / name
        shutil.rmtree(dest, ignore_errors=True)
        dest.mkdir(parents=True)
        for png in sorted(prev.glob("*.png")):
            Image.open(png).convert("RGB").save(dest / (png.stem + ".jpg"), quality=85)
        print("stored", name, len(list(dest.glob("*.jpg"))), "images", file=sys.stderr)


def diff(a, b):
    x = np.asarray(Image.open(a), dtype=np.int16)
    y = np.asarray(Image.open(b), dtype=np.int16)
    return 255.0 if x.shape != y.shape else float(np.abs(x - y).mean())


def comment(worktree, prefix, repo, ref, run_url):
    root = Path(worktree)
    url = lambda p: f"https://github.com/{repo}/blob/{ref}/{p.relative_to(root).as_posix()}?raw=true"
    img = lambda p, alt: f'<img src="{url(p)}" alt="{alt}" width="420">'
    out = [MARKER, "## Screenshots", "",
           f"Rendered by CI from this branch ([run]({run_url})). *Before* is `main`, *after* is this pull request. "
           "To get a shot of your change, add a named view to your zone's `VIEWS` "
           "(or `maps/<map>/tests/views.py`).", ""]
    maps = sorted(d for d in (root / prefix).iterdir() if d.is_dir()) if (root / prefix).is_dir() else []
    if not maps:
        out.append("_No previews were produced. The build probably failed; check the run._")
    for after_dir in maps:
        before_dir = root / "main" / after_dir.name
        changed, new, same = [], [], []
        for a in sorted(after_dir.glob("*.jpg")):
            if a.stem == "_sheet":
                continue
            b = before_dir / a.name
            if not b.exists():
                new.append(a)
            elif diff(a, b) > CHANGED:
                changed.append((b, a))
            else:
                same.append(a)
        gone = sorted(b.stem for b in before_dir.glob("*.jpg")
                      if b.stem != "_sheet" and not (after_dir / b.name).exists())
        out += [f"### `{after_dir.name}`", "",
                f"{len(changed)} changed, {len(new)} new, {len(same)} unchanged"
                + (f", removed: {', '.join(gone)}" if gone else ""), ""]
        if changed:
            out += ["| view | before | after |", "|---|---|---|"]
            out += [f"| `{a.stem}` | {img(b, 'before')} | {img(a, 'after')} |" for b, a in changed]
            out.append("")
        if new:
            out += ["| new view | after |", "|---|---|"]
            out += [f"| `{a.stem}` | {img(a, 'after')} |" for a in new]
            out.append("")
        if same:
            out += ["<details><summary>Unchanged views</summary>", ""]
            out += [f"`{a.stem}`<br>{img(a, a.stem)}<br>" for a in same]
            out += ["", "</details>", ""]
    print("\n".join(out))


def gate(worktree, prefix, changed):
    root = Path(worktree); bad = []
    for after_dir in sorted(d for d in (root / prefix).iterdir() if d.is_dir()) if (root / prefix).is_dir() else []:
        for a in after_dir.glob("*.jpg"):
            if a.stem != "_sheet" and np.asarray(Image.open(a).convert("L"), float).std() < 4:
                bad.append("%s/%s is blank (flat colour): camera inside a wall or nothing rendered" % (after_dir.name, a.stem))
    for m in changed.split():
        after_dir, before_dir = root / prefix / m, root / "main" / m
        if not after_dir.is_dir():
            bad.append("%s: no screenshots were produced" % m); continue
        moved = [a for a in after_dir.glob("*.jpg") if a.stem != "_sheet" and
                 (not (before_dir / a.name).exists() or diff(a, before_dir / a.name) > CHANGED)]
        if not moved and before_dir.is_dir():
            bad.append("%s changed but no screenshot shows a difference: add a view aimed at the change" % m)
    for b in bad:
        print("visual gate:", b)
    print("visual gate: %s" % ("FAIL" if bad else "ok"))
    sys.exit(1 if bad else 0)


if __name__ == "__main__":
    cmd, *args = sys.argv[1:]
    {"store": store, "comment": comment, "gate": gate}[cmd](*args)
