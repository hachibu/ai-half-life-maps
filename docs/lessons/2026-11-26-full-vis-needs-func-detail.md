---
title: Keep full VIS fast: build walls, floors and roofs as single big brushes, and let SplitDetail make the rest func_detail
area: geometry
tags: [battleaxecombatleague, prairieloop]
source: this session (Nov 26)
---

## What happened

Full VIS on prairieloop was heading for ~40 minutes (7,326 portals); with small brushes moved to func_detail it took 0.15 s (677 portals). BattleAxe went from 8+ minutes to ~20 s.

## Why

VIS works on the spaces structural brushes carve out. Every small structural brush (a step, a lamp, a seat) cuts more spaces and portals; func_detail brushes are still solid and lit but VIS ignores them.

## Rule

Write the map's big walls, floors and roofs as single brushes so `mapkit.SplitDetail` keeps them structural (two sides >= 256 units); everything smaller becomes func_detail automatically. A detail brush can't seal the map: if hlbsp reports a leak, the sealing piece was too small. Full VIS taking minutes means a map lost its detail split.
