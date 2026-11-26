---
title: Lint with the same golangci-lint version CI pins
tags: [go, lint, ci, general]
gate: checks
source: Go tooling 1/4 (#28, PR #30)
---

## What happened

The new Go packages passed `make lint` locally (golangci-lint v2.5.0) but failed in CI (v2.15.0): goconst
flagged the engine-limit names repeated as labels, keys and counter names. The old local version also
flagged two lines in rng.go that CI's version accepts.

## Why

Linter versions add and tune checks between releases, so a different version is a different rule set.

## Rule

Check `golangci-lint --version` against the version pinned in `.github/workflows/map-check.yml` before
trusting a local lint run; if they differ, treat CI as the judge and expect a round trip. Name strings that
act as keys (constants) instead of repeating literals.
