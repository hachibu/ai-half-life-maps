---
title: After changing tools/, run `make build test views` on a map and look at the images before pushing
area: ci
source: repo setup
---

## What happened

`make views` was broken for every map by a loop variable named `m` that shadowed the loaded map in the renderer.

## Why

Nothing ran the renderer between the change and the push.

## Rule

After any change under tools/, run `make build test views` on at least one map (`LIGHT=quick VIS=fast` is fine) and look at the screenshots before pushing. CI rebuilds every map for tools changes, but finding it there costs a round trip.
