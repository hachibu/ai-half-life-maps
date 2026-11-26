---
title: The modernize linter only runs in CI; apply its rewrites by hand
tags: [go, lint, ci, general]
gate: checks
source: Go tooling 3/4 (#53, PR #54)
---

## What happened

PR #54 passed golangci-lint locally but failed in CI on one `modernize` finding: `for _, m := range
strings.Fields(s)` should be `for m := range strings.FieldsSeq(s)`.

## Why

The sandbox's golangci-lint is older than the v2.15.0 that CI pins, and it has no `modernize` linter, so the
local run uses the config with that linter removed. The Go module proxy isn't reachable from the sandbox, so
the pinned version can't be installed there either.

## Rule

Before pushing Go code, apply modernize's usual rewrites yourself: `strings.FieldsSeq` / `strings.SplitSeq` when
you only range over the result, `slices.Clone`, `strings.CutPrefix`, `for i := range n`, `min` / `max`. Then
grep your diff for `range strings.Fields(` and `range strings.Split(`.
