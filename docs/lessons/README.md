# Lessons

One rule per file, learned from something that went wrong or took too long. Every rule's title appears in the
Lessons section of [AGENTS.md](../../AGENTS.md), which every agent reads first; the file holds the details.
`make readme` regenerates that list, and CI fails if a lesson is malformed or the list is out of date.

## When to write one

- a check went red on your PR,
- the human corrected you or your approach,
- something took much longer than it should have, or a measurement surprised you.

Update an existing lesson instead of adding a near-duplicate, and delete a lesson when what it describes no
longer exists (git history keeps it).

## Format

File name: `YYYY-MM-DD-short-slug.md`.

```
---
title: The rule itself, one line, no trailing period
area: workflow | geometry | lighting | textures | entities | content | ci | github | go
tags: [map names, only if the lesson is about specific maps]
source: the PR, map or change it came from
---

## What happened
## Why
## Rule
```

The title is what every agent reads, so write it as an instruction someone can follow without opening the
file ("Keep overhead decoration 72+ units above every walkable floor", not "Headroom problem").
