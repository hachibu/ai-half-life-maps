---
title: Measure speed-ups on GitHub's runners, not only in the sandbox
area: ci
tags: [battleaxecombatleague]
source: this session (Nov 26)
---

## What happened

BattleAxe's patch-size change (#65) cut full lighting 33% in the 2-core sandbox but only 7% on GitHub's 4-core runners.

## Why

Phases parallelise differently on different core counts, so which phase dominates changes between machines. The sandbox ratio said nothing about the runner.

## Rule

Use the sandbox to find candidates and check for visual changes, then judge the speed-up from CI's step times on the PR (`make build` prints per-step times). Report the runner number, not the sandbox one.
