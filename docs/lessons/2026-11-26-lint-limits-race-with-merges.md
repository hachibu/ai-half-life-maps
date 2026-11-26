---
title: Re-check lint limits against the latest main before a PR that lowers them merges
tags: [go, lint, ci, pipeline, general]
gate: checks
source: PR #42 merged after PR #39; fixed in the PR for issue #43
---

## What happened

PR #42 lowered the funlen ceiling from 101 to 53 statements. Its CI was green, but it ran on a base without
PR #39, which merged minutes earlier and added a Go renderer with functions of 63-93 statements. After both
merged, main failed `make lint`, and every new PR would have failed map-check with it.

## Why

The ruleset doesn't require a PR to be up to date with main before it merges, so CI checks each PR against
the main it branched from, not the main it lands on. A PR that tightens a rule (a lower lint limit, a new
linter) can pass while code it never saw breaks the new rule. The release workflow doesn't run lint, so
nothing raised an incident.

## Rule

Before pushing a PR that tightens a lint rule, fetch main, rebase, and run `make lint` again, and do it again
if main moves while the PR is open. Set limits from a measurement of the latest main, never from an earlier
one.
