---
title: Keep light samples under ~700k: coarse texture scale on big, distant or plain surfaces
area: lighting
tags: [battleaxecombatleague]
source: battleaxecombatleague restyle
---

## What happened

Adding a roof truss grid pushed BattleAxe to 752k of the 800k light samples; the trusses alone took 70k, a flat red trim 19k.

## Why

Light samples scale with surface area / (16 x texture scale)^2. Long thin decorative faces at scale 1 are expensive and nobody sees their shading.

## Rule

Give large, distant, hidden or flat-colour surfaces a coarse scale (4+) in the map's SCALE table. Before adding big geometry, run `build/bin/mapcheck lightbudget build/<map>/<map>.bsp` and stay under ~700k of the 800k limit.
