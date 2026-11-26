---
title: No face may need more than 324 light samples
tags: [bsp, compiler, lighting, general]
gate: checks
source: justincon history
---

## What happened

Large faces crashed the engine when hit by dynamic light.

## Why

GoldSrc allocates at most 18x18 = 324 light samples per face. Faces bigger than 240 texels in either direction must be split.

## Rule

Let the compiler subdivide (240 texels). For huge surfaces, raise the texture scale instead of making faces bigger. The structure check enforces 324.
