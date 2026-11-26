# A video script change made the visual gate expect a changed map

**What happened.** PR #39 edited `maps/prairieloop/video/walkthrough.py` only to switch it to the Go renderer.
map-check counted any file under `maps/prairieloop/` as a map change, so the visual gate wanted a screenshot
that differs from main. None could, because the map didn't change. The screenshots check went red.

**Fix.** map-check's map-change filter now ignores `maps/<map>/video/`, the same way it ignores `README.md` and `map.toml`.

**Rule.** When you add a non-map folder under `maps/<map>/`, add it to that filter in `.github/workflows/map-check.yml`.
