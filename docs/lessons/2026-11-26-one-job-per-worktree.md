---
title: Never run two git-changing jobs in the same worktree at once
area: workflow
source: this session (Nov 26)
---

## What happened

A background comparison stashed a change, built, and was about to pop the stash, while a foreground fix switched branches in the same worktree. The stash would have landed on the wrong branch.

## Why

A worktree has one checked-out branch and one stash. Background commands that stash, checkout or rebase share both with everything else you do there.

## Rule

Run background jobs that touch git state in their own worktree (`git worktree add /tmp/<name> <ref>`), or wait for them to finish before changing branches. Prefer copying files aside to `git stash` in scripts.
