---
title: Run `make views` before pushing renderer or reader changes
tags: [render, tools, ci, general]
gate: checks
source: repo setup
---

## What happened

`make views` was broken for every map by a loop variable named `m` that shadowed the loaded map in the renderer.

## Why

Nothing ran the renderer between the change and the push.

## Rule

Any change under tools/ must be followed by `make build test views` on at least one map before pushing. CI builds every map for tools changes; look at the screenshots.
