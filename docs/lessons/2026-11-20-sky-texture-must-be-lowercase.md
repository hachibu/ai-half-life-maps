---
title: The sky texture must be named lowercase `sky`
tags: [bsp, compiler, sky, general]
gate: checks
source: justincon history
---

## What happened

Firing a rocket at the sky in justincon crashed the game: `lightmap for texture SKY too large; cannot exceed 324`.

## Why

The engine only treats a face as sky (no lightmap, no subdivision) when its texture is named exactly `sky`. An uppercase `SKY` was lit like a normal giant face.

## Rule

Name the sky texture `sky` (lowercase). The structure check (`tools/bsp/bspfile`) fails a build whose sky is not lowercase; never weaken that check.
