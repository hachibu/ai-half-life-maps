---
title: Build stairs with 8-unit risers (16 at most) and 72+ units of headroom; add a route for every new walkable area
area: geometry
tags: [prairieloop, casaroble]
source: prairieloop history
---

## What happened

Several walk routes failed: stage steps too tall, a stair head bumping a ceiling.

## Why

The player can step up at most 18 units and is 72 units tall; the walker models exactly that.

## Rule

Stairs: 8-unit risers (they look better, see the GoldSrc dimension references in AGENTS.md); never more than 16. Extending a stair at its foot moves the bottom step: re-check routes that start there. Leave 72+ units of headroom over every stair tread, including the last one under a floor above. Add a route for every new walkable area.
