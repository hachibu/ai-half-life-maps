---
title: Lighting time is surfaces times light sources; cut both in the map, not the compiler
tags: [lighting, compiler, battleaxecombatleague, prairieloop, general]
gate: none
source: faster PR builds (BattleAxe full lighting 174 s -> 104 s locally)
---

## What happened

Full lighting (SDHLT's hlrad) was almost all of a PR build: about 3 minutes for BattleAxe on GitHub's runners.
Three map changes cut it by 40% with no visible difference: `NULL` on the crowd blocks' bottoms (they sit on
the seats; half the crowd's light samples), no glow from the translucent light-beam sheets (they had been
turned into thousands of light sources), and half-resolution crowd textures at double scale in the stands
(same size in the world, a quarter of the light samples; the floor rows by the octagon keep full resolution).

## Why

hlrad's cost grows with the light samples on every face (one per 16 texels, so texture scale sets it) and the
number of light sources that reach them. Every patch of a texture in a map's EMISSIVE table becomes its own
light source. Faces of brush entities (func_illusionary crowds, signs) are never removed by CSG, even when
they're buried against something, so they get lit anyway. Additive brush entities are drawn without their
lighting in game, yet hlrad still lights them (`zhlt_striprad` only discards the result).

## Rule

- Give faces nobody can see the `NULL` texture (bottoms resting on something, backs against walls), above all
  on brush entities.
- Only real light sources go in EMISSIVE: lamps, screens, light banks. Not beams, trims or decorations.
- Measure with `build/bin/bspinfo lightbudget MAP.bsp`; big decorative surfaces far from players can use a
  half-resolution texture at double scale.
- Additive sheets (prairieloop's snow) cost lighting time for nothing visible; prefer fewer, smaller ones.
