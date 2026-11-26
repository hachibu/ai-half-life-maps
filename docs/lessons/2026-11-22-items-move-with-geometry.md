---
title: Re-check item spots when you move geometry
tags: [placements, items, battleaxecombatleague, general]
gate: checks
source: battleaxecombatleague restyle
---

## What happened

After the restyle two batteries ended up inside the stands (`ITEM IN SOLID`).

## Why

Item coordinates were hard-coded while the solid around them changed.

## Rule

Place items relative to the geometry constants they sit on, and run `make test` (placements) after any geometry change.
