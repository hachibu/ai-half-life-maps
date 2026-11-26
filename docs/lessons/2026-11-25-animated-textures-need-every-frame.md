---
title: Register every frame of an animated texture; it plays at 10 frames per second
area: textures
source: prairieloop interactive upgrade
---

## What happened

An animated texture showed only one frame in game.

## Why

The engine animates `+0name`, `+1name`, ... only if every frame is in the WAD; it advances 10 frames per second.

## Rule

Register all frames (`T['+%dNAME' % k]` for k in range(n)). Design loops for 10 fps: 10 frames = 1 second. `make test` (structure) fails a missing frame.
