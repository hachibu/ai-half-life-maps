---
title: Add non-map folders under maps/<map>/ to map-check's map-change filter
area: ci
source: Go tooling 2/4 (#33, PR #39)
---

## What happened

PR #39 edited `maps/prairieloop/video/walkthrough.py` only to switch it to the Go renderer.
map-check counted any file under `maps/prairieloop/` as a map change, so the visual gate wanted a screenshot
that differs from main. None could, because the map didn't change. The screenshots check went red.

## Why

map-check decides "this map changed" from file paths. It skipped only `README.md` and `map.toml`, so any other
file under a map's folder counted, including scripts that never touch the compiled `.bsp`. The fix: the
map-change filter now ignores `maps/<map>/video/` too.

## Rule

When you add a non-map folder under `maps/<map>/`, add it to that filter in `.github/workflows/map-check.yml`.
