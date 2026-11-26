---
title: Running a module inside a package imports the package's __init__ first
tags: [ci, python, tools, general]
gate: checks
source: tools/ reorganization (PR #64)
---

## What happened

PR #64 moved `tools/render/views.py` (no dependencies) to `tools/mapkit/views.py`. CI's README check runs
`python -m tools.mapkit.views` in a job that hadn't installed the Python requirements yet, and it failed with
`ModuleNotFoundError: No module named 'numpy'`.

## Why

`python -m tools.mapkit.views` imports `tools.mapkit` first, and `tools/mapkit/__init__.py` imports the texture
library, which needs numpy. The module itself didn't change; its package did.

## Rule

When you move a module into a package, check what the package's `__init__.py` imports: every command that runs
the module now needs those dependencies too. In CI, install `requirements.txt` before any step that runs Python
from `tools/mapkit`.
