---
title: See-through `{` textures look solid blue in previews
tags: [render, preview, textures, general]
gate: review
source: battleaxecombatleague history
---

## What happened

Cage fences showed as solid blue walls in the CI screenshots.

## Why

`{` textures use palette colour 255 (pure blue) as the see-through key. The game hides it; our preview renderer doesn't yet.

## Rule

Don't 'fix' a blue fence in a map: it is see-through in game. Reviewers: ignore the blue. Fixing the renderer is a welcome tools task.
