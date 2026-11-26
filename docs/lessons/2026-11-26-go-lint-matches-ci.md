---
title: Lint Go with CI's golangci-lint in mind: apply modernize's rewrites by hand and re-lint after rebasing
area: go
source: this session (Nov 26)
---

## What happened

PRs #30, #54 and the complexity-limit work: CI failed three times on lint that passed locally (goconst, a modernize rewrite, a limit lowered against an older main).

## Why

The sandbox's golangci-lint is older than the v2.15.0 CI pins and has no `modernize` linter, and CI lints the PR merged onto the latest main, not the commit you linted.

## Rule

Run `make lint` with the config minus `modernize`, then apply modernize's usual rewrites yourself: `strings.FieldsSeq`/`SplitSeq` when you only range over the result, `slices.Clone`, `strings.CutPrefix`, `for i := range n`, `min`/`max`. Name strings that act as keys (constants). Rebase on the latest main and lint again right before pushing; set complexity limits from a measurement of that main.
