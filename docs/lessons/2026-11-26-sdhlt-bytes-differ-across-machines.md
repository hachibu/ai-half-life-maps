---
title: SDHLT's output bytes differ between machines; pin only what is portable
tags: [sdhlt, golden, ci, general]
gate: checks
source: SDHLT switch (#45, PR #46)
---

## What happened

PR #46 recorded `make golden` hashes for the template map and prairieloop on the sandbox. On GitHub's runner
the template matched, but prairieloop hashed differently, so the golden check went red.

## Why

SDHLT is the same commit on both machines, but it is built with each machine's C++ compiler and runs on a
different CPU. Lighting (hlrad) is floating-point heavy, so the last bits of a big map's lightmaps differ. SDHLT is
deterministic on one machine (same hash across runs and thread counts), not across machines. The small template
map has no such differences.

## Rule

Pin a byte hash only for output that is the same on every machine (here, the template map). For anything else,
check behaviour (limits, routes, scores, screenshots) rather than bytes. Before recording a golden value from
the sandbox, ask whether CI's machine will compute the same bytes.
