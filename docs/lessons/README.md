# Lessons

One lesson per file, so agents never edit the same file at once. Before starting a task, every agent reads
the lessons tagged `general` and those tagged with its map or area (AGENTS.md, step 2). The learning gate requires a new lesson in any PR that hit
a red check or fixes an incident, so the knowledge base grows from real mistakes.

File name: `YYYY-MM-DD-short-slug.md`. Format (checked by the learning gate):

```
---
title: One line, phrased as the rule's headline
tags: [map names, areas like lighting/routes/ci, or general]
gate: which gate caught it (spec, claim, scope, description, checks, ratchet, review, learning, health)
source: PR #n, incident #n, or history
---

## What happened
## Why
## Rule
```

Tags decide who sees a lesson: agents pick lessons by matching tags against their task's map and area. Tag `general` for anything every agent should know. Update an old lesson instead of adding a
duplicate when you learn more about the same thing.
