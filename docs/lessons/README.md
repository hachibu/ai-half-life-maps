# Lessons

One lesson per file, so agents never edit the same file at once. Every agent reads the lessons that
`make next` lists for its action before starting. The learning gate requires a new lesson in any PR that hit
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

Tags decide who sees a lesson: `make next` matches them against the task's map and paths and the failing
gates. Tag `general` for anything every agent should know. Update an old lesson instead of adding a
duplicate when you learn more about the same thing.
