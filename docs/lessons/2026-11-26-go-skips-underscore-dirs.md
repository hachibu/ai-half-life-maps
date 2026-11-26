---
title: Name maps/_template's packages explicitly: Go's ./... skips directories starting with _
area: go
source: the Go-only port
---

## What happened

After moving the maps to Go, `go vet ./maps/_template/...` reported "matched no packages", and `./...` silently left the template out of lint and vet.

## Why

The go tool ignores directories whose names start with `_` or `.` (and `testdata`) when expanding `...` patterns, even when the pattern starts inside one. Naming the directory itself (`./maps/_template`) works.

## Rule

Keep the template's code in the one package `maps/_template` (zones are `zone_*.go` files in it, not a subpackage), and keep `GO_PKGS` in the Makefile listing `./maps/_template` next to `./...`. A new tool that walks packages must do the same.
