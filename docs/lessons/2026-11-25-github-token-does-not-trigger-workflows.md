---
title: Pushes and PRs made with the default Actions token don't start CI
tags: [ci, github, pipeline]
gate: checks
source: pipeline design
---

## What happened

A workflow that pushes a branch or merges with GITHUB_TOKEN produces events that start no other workflows.

## Why

GitHub blocks GITHUB_TOKEN events from triggering workflows, to prevent loops (workflow_dispatch is the exception).

## Rule

After the pipeline merges, it dispatches release.yml explicitly. A bot that opens PRs must use a real token (a fine-grained personal access token) so map-check runs.
