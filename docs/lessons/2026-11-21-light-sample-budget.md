---
title: Spend the light-sample budget where players look
tags: [lighting, textures, battleaxecombatleague, prairieloop, general]
gate: checks
source: battleaxecombatleague restyle
---

## What happened

Adding a roof truss grid pushed BattleAxe to 752k of the 800k light samples; the trusses alone took 70k, a flat red trim 19k.

## Why

Light samples scale with surface area / (16 x texture scale)^2. Long thin decorative faces at scale 1 are expensive and nobody sees their shading.

## Rule

Give large, distant, hidden or flat-colour surfaces a coarse scale (4+) in the map's SCALE table. Before adding big geometry, check the per-texture sample count (see the snippet in docs/pipeline.md) and stay under ~700k.
