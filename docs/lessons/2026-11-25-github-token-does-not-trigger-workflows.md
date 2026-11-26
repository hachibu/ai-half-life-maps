---
title: Dispatch follow-up workflows explicitly: the Actions token's pushes and merges start nothing
area: ci
source: pipeline design
---

## What happened

A workflow that pushes a branch or merges with GITHUB_TOKEN produces events that start no other workflows.

## Why

GitHub blocks GITHUB_TOKEN events from triggering workflows, to prevent loops (workflow_dispatch is the exception).

## Rule

automerge.yml merges with the Actions token, so it dispatches release.yml itself. Any workflow that pushes, merges or opens PRs with GITHUB_TOKEN must dispatch what should run next (workflow_dispatch is the exception GitHub allows), or use a personal access token.
