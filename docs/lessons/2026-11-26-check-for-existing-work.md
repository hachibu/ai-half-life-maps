---
title: Look for an existing PR or issue before starting a task
tags: [planning, github, pipeline, general]
gate: scope
source: README screenshots task (#11/#12 vs #13)
---

## What happened

The README screenshot task was built twice: one session opened issue #11 and PR #12 (merged), while another
opened #13 and built the same thing on a branch with the same name. The second copy was thrown away.

## Why

Nothing forced a search for existing work, and two sessions can work on the repo at the same time.

## Rule

Before claiming or creating a task, search open and recently merged issues and PRs for the same goal
(`gh api "repos/{owner}/{repo}/issues?state=all&per_page=30"`). If a branch name you want already exists,
stop and find out why before reusing it.
