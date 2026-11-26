---
title: Lint Go with the golangci-lint version CI pins (download its release), and re-lint after rebasing
area: go
source: this session (Nov 26); the Go-only port
---

## What happened

PRs #30, #54 and the complexity-limit work: CI failed three times on lint that passed locally (goconst, a modernize rewrite, a limit lowered against an older main). The sandbox's own golangci-lint is older than the v2.15.0 CI pins and stops with "unknown linters: 'modernize'" on this config.

## Why

Linters change between versions, and CI lints the PR merged onto the latest main, not the commit you linted.

## Rule

Fetch CI's version from its GitHub release (the sandbox can reach github.com, but not the Go module proxy):
`curl -sSfL -o /tmp/gcl.tgz https://github.com/golangci/golangci-lint/releases/download/v2.15.0/golangci-lint-2.15.0-linux-amd64.tar.gz`, unpack it and put it first on PATH before `make lint`. Name strings that act as keys (constants: goconst counts per package). Rebase on the latest main and lint again right before pushing; set complexity limits from a measurement of that main.
