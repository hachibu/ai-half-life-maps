---
title: Design vertical trigger_push launch pads for players dropping or jumping in
area: entities
tags: [prairieloop]
source: prairieloop interactive upgrade
---

## What happened

A lift built from trigger_push did nothing for players walking onto it.

## Why

A vertical push sets base velocity, which the engine applies like acceleration; on the ground friction wins.

## Rule

Design launch pads so players drop or jump into them, and say so in the PR's 'What a player will notice'.
