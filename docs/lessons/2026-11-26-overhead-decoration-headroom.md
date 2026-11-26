---
title: Overhead decoration must clear 72 units above every walkable floor
tags: [routes, headroom, decoration, battleaxecombatleague, general]
gate: checks
source: BattleAxe indoor restyle (#3)
---

## What happened

A red trim ledge under the new glass suites sat at z 852-860, just above the upper deck's top row (floor 800). Four walk routes along the diagonal sides failed with "blocked at (1853, 1853)".

## Why

A standing player is 72 units tall, so the ledge has to clear floor + 72 = 872. On the diagonal sides the player box reaches about 23 units closer to the wall than on the flat sides, so the ledge only blocked the diagonals.

## Rule

Before adding trim, signs or ledges near a walkable area, compute floor + 72 for the highest floor underneath and keep the bottom of the decoration above it (or more than 16 units horizontally from where players can stand). Check diagonal sides as well as straight ones, and run `make test` before pushing.
