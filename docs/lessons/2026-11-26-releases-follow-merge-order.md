---
title: Check which commit a release came from before calling it wrong
area: ci
source: this session (Nov 26)
---

## What happened

Two PRs merged close together released in the order they landed, not the order they were opened, so one release had an unexpected name and content.

## Why

Each merge starts its own release run; runs finish in merge order, and the newest tag reflects the last merge.

## Rule

When a release looks wrong, compare its target commit with main's history before changing anything.
