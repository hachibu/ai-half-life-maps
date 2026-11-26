---
title: Claude sessions: GitHub REST only (no GraphQL)
tags: [claude, github, tooling]
gate: checks
source: pipeline setup
---

## What happened

`gh pr create`, `gh release list` and `gh label list` failed from a Claude session: GraphQL is blocked there; deleting refs and releases is also refused.

## Why

The session's GitHub proxy allows REST calls for normal work but not GraphQL or destructive repo admin.

## Rule

Open PRs with `gh api repos/{owner}/{repo}/pulls -f title=... -f head=... -f base=main -f body=...`. Ask a human for deletions and settings changes.
