---
title: Keep the task's Paths in step with the PR, and re-claim after an overlapping task closes
tags: [pipeline, claims, scope, general]
gate: claim
source: release names (#16/#17), spec gate fix (#18)
---

## What happened

Issue #16 lost its `claimed:claude` label because the Go compiler task (#14), claimed earlier, also covered
`.github/workflows/`. Later #17 grew to edit `map-check.yml`, which #16's Paths didn't list.

## Why

The claim gate keeps the earlier of two overlapping claims; the scope gate holds a PR to its task's Paths.
Both are working as designed.

## Rule

List narrow Paths (files, not whole folders) when you can. When a task grows, edit its Paths before pushing.
If your claim was removed for overlapping an earlier task, wait for that task to close and add the label back.
