---
title: Use additive func_illusionary sheets for weather everywhere; sprites only for local effects
area: entities
tags: [prairieloop]
source: prairieloop continuous snow
---

## What happened

Snow made of env_sprites spaced 380 units apart read as patches, and the user preferred a friend's continuous snow.

## Why

Sprites are camera-facing squares with faded edges; gaps between them show. They are also limited in number.

## Rule

For weather everywhere, use a grid of additive animated func_illusionary sheets (see prairieloop SNOW) clipped to stop at roofs; keep sprites for local effects (bursts, steam).
