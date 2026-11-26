---
title: In a worktree, branch and rebase from FETCH_HEAD; origin/main can be stale
area: workflow
source: this session (Nov 26)
---

## What happened

Several times in one session (#64, #65 and the README PRs) a branch was created or rebased onto a weeks-old commit, or a rebase replayed an already-merged commit and conflicted.

## Why

In a git worktree, `git fetch origin main` may update only FETCH_HEAD, not `origin/main`. A squash-merged PR also leaves your local copy of its commits unmerged, so rebasing a branch built on top of it replays them.

## Rule

Always `git fetch -q origin main && git checkout -B <branch> FETCH_HEAD` for new work and rebase onto FETCH_HEAD. After a PR you built on squash-merges, rebuild the follow-up branch from FETCH_HEAD and cherry-pick only its own commits. Force-push with an explicit lease: `--force-with-lease=<branch>:<sha you expect>`.
