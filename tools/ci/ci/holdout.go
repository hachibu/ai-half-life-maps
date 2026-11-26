package main

// Holdout scenarios: checks the agents cannot see, written by humans, run only in CI.
//
// The scenarios live encrypted in holdout/scenarios.enc. Only CI has the key (Actions secret HOLDOUT_KEY), so an
// agent can't read them and tune a map to pass them. CI prints only a pass rate per map, never which scenario
// failed. A PR may not lower a map's holdout pass rate.
//
// Plain-JSON format before encryption (a list):
//
//	{"map": "prairieloop", "name": "spawn to hall fast", "kind": "reach",
//	 "from": "spawns" | [x, y, z_floor], "to": "weapon_rpg" | [x, y, z_floor], "within": 4800}
//
// kind "reach": from every start (or the point), some target (an entity class, or a point) is reachable within
// `within` units of running (default: any distance). The Go scorer (build/bin/satisfy) evaluates them on its
// navigation graph. The decrypted scenarios go to it over stdin and are never written to disk.
//
// Humans: HOLDOUT_KEY=... make holdout-encrypt (holdout/scenarios.json -> holdout/scenarios.enc; never commit
// the .json).

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	holdoutEnc = "holdout/scenarios.enc"
	satisfyBin = "build/bin/satisfy"
)

var (
	errDecrypt    = errors.New("holdout: cannot decrypt (wrong HOLDOUT_KEY?)")
	errNoSatisfy  = errors.New(satisfyBin + " is missing; run `make bspc`")
	errSatisfyRun = errors.New("holdout: satisfy failed")
)

// holdoutResult is a map's pass rate (nil when there is nothing to run, with a note saying why).
type holdoutResult struct {
	Rate *float64 `json:"rate"`
	N    int      `json:"n"`
	Note string   `json:"note,omitempty"`
}

func (h holdoutResult) String() string {
	if h.Rate == nil {
		return h.Note
	}
	return fmt.Sprintf("%.1f%% of %d scenarios", *h.Rate, h.N)
}

// loadScenarios decrypts the scenarios (nil when there is no key or no file).
func loadScenarios(root string) ([]json.RawMessage, error) {
	enc := filepath.Join(root, holdoutEnc)
	if os.Getenv("HOLDOUT_KEY") == "" {
		return nil, nil
	}
	if _, err := os.Stat(enc); err != nil {
		return nil, nil //nolint:nilerr // no scenarios file: nothing to run, same as no key
	}
	cmd := exec.Command("openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2", "-in", enc, "-pass", "env:HOLDOUT_KEY") //nolint:gosec // fixed command; enc is the repo's scenarios file
	out, err := cmd.Output()
	if err != nil {
		return nil, errDecrypt
	}
	var sc []json.RawMessage
	if err := json.Unmarshal(out, &sc); err != nil {
		return nil, errDecrypt
	}
	return sc, nil
}

func runHoldout(root, mapName, bsp string) (holdoutResult, error) {
	all, err := loadScenarios(root)
	if err != nil {
		return holdoutResult{}, err
	}
	if all == nil {
		return holdoutResult{Note: "no holdout scenarios (or no key): skipped"}, nil
	}
	var mine []json.RawMessage
	for _, s := range all {
		var head struct {
			Map string `json:"map"`
		}
		if json.Unmarshal(s, &head) == nil && head.Map == mapName {
			mine = append(mine, s)
		}
	}
	if len(mine) == 0 {
		return holdoutResult{Note: "no holdout scenarios for this map"}, nil
	}
	sat := filepath.Join(root, satisfyBin)
	if _, err := os.Stat(sat); err != nil {
		return holdoutResult{}, errNoSatisfy
	}
	out := strings.TrimSuffix(bsp, filepath.Ext(bsp)) + ".holdout.json"
	input, err := json.Marshal(mine)
	if err != nil {
		return holdoutResult{}, err
	}
	cmd := exec.Command(sat, bsp, "--holdout", "-", "--json", out) //nolint:gosec // our own scorer, on a map path from the Makefile
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// satisfy's message names a scenario index, never a scenario
		return holdoutResult{}, fmt.Errorf("%w: %s", errSatisfyRun, strings.TrimSpace(stderr.String()))
	}
	var res holdoutResult
	b, err := os.ReadFile(out)
	if err != nil {
		return holdoutResult{}, err
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return holdoutResult{}, err
	}
	return holdoutResult{Rate: res.Rate, N: res.N}, nil
}

// cmdHoldout: ci holdout MAP BSP [--json OUT].
func cmdHoldout(root string, args []string) error {
	res, err := runHoldout(root, args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Printf("holdout %s: %s\n", args[0], res)
	if len(args) == 4 && args[2] == "--json" {
		b, err := json.Marshal(res)
		if err != nil {
			return err
		}
		return os.WriteFile(args[3], b, 0o644)
	}
	return nil
}
