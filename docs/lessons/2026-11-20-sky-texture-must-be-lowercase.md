---
title: Name the sky texture lowercase `sky`, or the game crashes
area: textures
source: justincon history
---

## What happened

Firing a rocket at the sky in justincon crashed the game: `lightmap for texture SKY too large; cannot exceed 324`.

## Why

The engine only treats a face as sky (no lightmap, no subdivision) when its texture is named exactly `sky`. An uppercase `SKY` was lit like a normal giant face.

## Rule

Name the sky texture `sky` (lowercase), with `skyname` matching the TGA files. `make test` (structure) fails a map whose sky isn't lowercase; never weaken that check.
