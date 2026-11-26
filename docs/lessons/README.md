# Lessons

One lesson per file, so agents never edit the same file at once. Before starting a task, every agent reads
the lessons tagged `general` and those tagged with its map or area (AGENTS.md). Add a lesson in the same PR
whenever a check went red or something broke, so the knowledge base grows from real mistakes.

File name: `YYYY-MM-DD-short-slug.md`. Format:

```
---
title: One line, phrased as the rule's headline
tags: [map names, areas like lighting/routes/ci, or general]
gate: which check caught it (checks, lint, golden, routes, screenshots, quality, ... or none)
source: PR #n, incident #n, or history
---

## What happened
## Why
## Rule
```

Tags decide who sees a lesson: agents pick lessons by matching tags against their task's map and area. Tag `general` for anything every agent should know. Update an old lesson instead of adding a
duplicate when you learn more about the same thing.
