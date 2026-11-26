---
title: Decoration that players never touch should not cost collision
tags: [bsp, clipnodes, entities, general]
gate: checks
source: battleaxecombatleague history
---

## What happened

Thousands of crowd brushes blew the clipnode limit (32767).

## Why

Every world brush is cut into the three collision hulls. Crowds, beams and signage don't need collision.

## Rule

Put decoration in a `func_illusionary` (no collision, no clipnodes). Keep world brushes for things players stand on or bump into.
