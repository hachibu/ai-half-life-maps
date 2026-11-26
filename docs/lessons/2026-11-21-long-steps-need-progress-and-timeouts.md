---
title: Every long step prints progress and runs under a timeout
tags: [compiler, lighting, ci, general]
gate: checks
source: battleaxecombatleague history
---

## What happened

A compile sat silent for 30 minutes; the first guess (light beams) was wrong. The real cause was lighting testing every face against every light.

## Why

Lighting cost grew with faces x lights x samples, and nothing reported progress, so a slow step looked like a hang.

## Rule

Cull per face before per sample, print progress, and wrap long commands in `timeout`. A silent process is a bug. Diagnose with timings before guessing.
