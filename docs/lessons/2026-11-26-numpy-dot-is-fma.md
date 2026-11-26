---
title: numpy's small dot products are fused multiply-adds; match them when porting
tags: [compiler, go, bsp, numerics, general]
gate: checks
source: Go compiler port (#14)
---

## What happened

The Go port of the BSP compiler matched Python byte for byte on mostly axis-aligned maps but built different
geometry for BattleAxe (octagons, beams, hulls): different splits, faces and lightmaps.

## Why

numpy computes `a @ b` for 3-vectors with fused multiply-adds (via OpenBLAS), gemv and einsum use yet
other orders. The Go code used plain `a0*b0 + a1*b1 + a2*b2`. The last-bit differences flipped a few
`EPS` comparisons on diagonal planes, and one flipped split changes the whole subtree.

## Rule

When porting numeric code, measure each primitive against the original on thousands of random inputs and
copy the exact operation order (see tools/bsp/bspc/README.md), and compare every map lump by lump before
switching over. Once the original is retired (the Python compiler was deleted in #24), pin the output with a
golden check (`make golden`) so a refactor that changes last bits is still caught.
