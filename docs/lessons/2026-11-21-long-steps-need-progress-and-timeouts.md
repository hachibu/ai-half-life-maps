---
title: Run long steps under a timeout with progress output; measure before guessing
area: workflow
source: battleaxecombatleague history
---

## What happened

A compile sat silent for 30 minutes; the first guess (light beams) was wrong. The real cause was lighting testing every face against every light.

## Why

Lighting cost grew with faces x lights x samples, and nothing reported progress, so a slow step looked like a hang.

## Rule

Wrap long commands in `timeout`, make them print progress, and diagnose with timings (per step, per phase) before guessing. A silent process is a bug.
