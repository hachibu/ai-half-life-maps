// ci: the CI helpers, in one command (run from the repo root; `make bspc` builds build/bin/ci).
//
//	ci package MAP                                  build/MAP/valve + .res + README -> build/MAP/MAP.zip
//	ci quality measure MAP                          satisfaction + holdout -> build/MAP/quality.json
//	ci quality compare ARTIFACTS BASELINE.json      PR: exit 1 if a map's quality dropped below main's
//	ci quality baseline ARTIFACTS OUT.json          main: write the new baseline
//	ci holdout MAP BSP [--json OUT]                 encrypted holdout scenarios (needs HOLDOUT_KEY)
//	ci previews store ARTIFACTS WORKTREE PREFIX     PNG previews -> WORKTREE/PREFIX/<map>/*.jpg
//	ci previews comment WORKTREE PREFIX REPO REF RUN_URL    the PR's before/after markdown, to stdout
//	ci previews gate WORKTREE PREFIX "map1 map2"    visual gate: exit 1 with reasons
//	ci readme [--check]                             regenerate (or check) the READMEs' screenshot sections
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	errUsage      = errors.New("usage: see the comment at the top of tools/ci/ci/main.go")
	errNoRepoRoot = errors.New("not inside the repository (no go.mod above the current directory)")
)

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ci:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func run(args []string) (int, error) {
	if len(args) == 0 {
		return 2, errUsage
	}
	root, err := repoRoot()
	if err != nil {
		return 1, err
	}
	switch {
	case args[0] == "package" && len(args) == 2:
		return 0, cmdPackage(root, args[1])
	case args[0] == "quality" && len(args) >= 2:
		return cmdQuality(root, args[1:])
	case args[0] == "holdout" && len(args) >= 3:
		return 0, cmdHoldout(root, args[1:])
	case args[0] == "previews" && len(args) >= 2:
		return cmdPreviews(args[1:])
	case args[0] == "readme":
		return cmdReadme(root, len(args) > 1 && args[1] == "--check")
	}
	return 2, errUsage
}

// repoRoot is the nearest directory at or above the working directory that holds go.mod.
func repoRoot() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		up := filepath.Dir(d)
		if up == d {
			return "", errNoRepoRoot
		}
		d = up
	}
}

// pyFloat formats a float the way Python's str() does for ordinary values (100.0, 87.5).
func pyFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eEn") {
		s += ".0"
	}
	return s
}
