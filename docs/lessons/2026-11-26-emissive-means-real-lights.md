---
title: List only real light sources in a map's EMISSIVE table; each becomes many lights
area: lighting
tags: [battleaxecombatleague]
source: this session (Nov 26)
---

## What happened

After the switch to SDHLT, glowing textures stopped lighting anything until EMISSIVE was turned into an `info_texlights` entity; later, BattleAxe's translucent light-beam sheets in EMISSIVE turned out to be ~1,800 light sources.

## Why

SDHLT reads glowing textures from `info_texlights` (written by `texlights()` in tools/mapkit), and every patch of an emitting surface becomes its own light that hlrad traces.

## Rule

Put lamps, screens, light banks and signs that should glow in EMISSIVE; leave out beams, trims and decoration. Large emitters cost lighting time per patch, so prefer a few compact ones.
