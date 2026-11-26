---
title: Write tasks an agent can finish in one PR and a check can prove
tags: [planning]
gate: spec
source: pipeline design
---

## What happened

Large vague tasks stall in review and can't be verified by the gates.

## Why

The gates only prove what a route, view, placement or score can measure.

## Rule

One task = one PR = one map or zone. Acceptance criteria name the route, view or score that proves them. Split anything bigger and link it with `Depends on`.
