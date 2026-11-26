---
title: Animated textures need every frame, and loop at 10 fps
tags: [textures, animation, general]
gate: checks
source: prairieloop interactive upgrade
---

## What happened

An animated texture showed only one frame in game.

## Why

The engine animates `+0name`, `+1name`, ... only if every frame is in the WAD; it advances 10 frames per second.

## Rule

Register all frames (`T['+%dNAME' % k]` for k in range(n)). Design loops for 10 fps: 10 frames = 1 second. The reader check fails a missing +1 frame.
