---
title: Vertical trigger_push only lifts players who jump in
tags: [entities, trigger_push, prairieloop]
gate: review
source: prairieloop interactive upgrade
---

## What happened

A lift built from trigger_push did nothing for players walking onto it.

## Why

A vertical push sets base velocity, which the engine applies like acceleration; on the ground friction wins.

## Rule

Design launch pads so players drop or jump into them, and say so in the PR's 'What a player will notice'.
