package main

// Quality ratchet: a map's satisfaction score and holdout pass rate may not drop below main's. ARTIFACTS holds
// one folder per map containing quality.json (the CI artifact layout).

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// tolerance: a PR may lower a map's satisfaction score by at most this much.
const tolerance = 2.0

type satisfaction struct {
	Score   float64         `json:"score"`
	Metrics json.RawMessage `json:"metrics"`
}

type quality struct {
	Map          string          `json:"map"`
	Satisfaction json.RawMessage `json:"satisfaction"`
	Holdout      holdoutResult   `json:"holdout"`
}

// baseEntry is one map in the baseline (scores/main.json on the previews branch); fields sorted by name.
type baseEntry struct {
	Holdout      *float64        `json:"holdout"`
	Metrics      json.RawMessage `json:"metrics,omitempty"`
	Satisfaction *float64        `json:"satisfaction"`
}

func cmdQuality(root string, args []string) (int, error) {
	switch {
	case args[0] == "measure" && len(args) == 2:
		return 0, measure(root, args[1])
	case args[0] == "compare" && len(args) == 3:
		return compare(args[1], args[2])
	case args[0] == "baseline" && len(args) == 3:
		return 0, baseline(args[1], args[2])
	}
	return 2, errUsage
}

func satisfactionScore(root, bsp string) (json.RawMessage, error) {
	sat := filepath.Join(root, satisfyBin)
	if _, err := os.Stat(sat); err != nil {
		return nil, errNoSatisfy
	}
	out := strings.TrimSuffix(bsp, filepath.Ext(bsp)) + ".satisfaction.json"
	cmd := exec.Command(sat, bsp, "--json", out) //nolint:gosec // our own scorer, on a map path from the Makefile
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return os.ReadFile(out)
}

func measure(root, name string) error {
	bsp := filepath.Join("build", name, name+".bsp")
	sat, err := satisfactionScore(root, bsp)
	if err != nil {
		return err
	}
	ho, err := runHoldout(root, name, bsp)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(quality{Map: name, Satisfaction: sat, Holdout: ho}, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("build", name, "quality.json"), b, 0o644); err != nil {
		return err
	}
	var s satisfaction
	if err := json.Unmarshal(sat, &s); err != nil {
		return err
	}
	fmt.Printf("%s satisfaction %.1f %s | holdout %s\n", name, s.Score, compact(s.Metrics), ho)
	return nil
}

func compact(raw json.RawMessage) string {
	var m any
	if json.Unmarshal(raw, &m) != nil {
		return string(raw)
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// loadArtifacts reads ARTIFACTS/*/quality.json, in folder order.
func loadArtifacts(artifacts string) ([]quality, error) {
	paths, err := filepath.Glob(filepath.Join(artifacts, "*", "quality.json"))
	if err != nil {
		return nil, err
	}
	var out []quality
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var q quality
		if err := json.Unmarshal(b, &q); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, q)
	}
	return out, nil
}

func readBaseline(path string) (map[string]baseEntry, error) {
	base := map[string]baseEntry{}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return base, nil
	}
	if err != nil {
		return nil, err
	}
	return base, json.Unmarshal(b, &base)
}

func compare(artifacts, baselinePath string) (int, error) {
	cur, err := loadArtifacts(artifacts)
	if err != nil {
		return 1, err
	}
	base, err := readBaseline(baselinePath)
	if err != nil {
		return 1, err
	}
	var bad []string
	lines := []string{"| map | satisfaction (main → PR) | holdout (main → PR) |", "|---|---|---|"}
	for _, q := range cur {
		var s satisfaction
		if err := json.Unmarshal(q.Satisfaction, &s); err != nil {
			return 1, err
		}
		b := base[q.Map]
		s0, s1, h0, h1 := b.Satisfaction, s.Score, b.Holdout, q.Holdout.Rate
		if s0 != nil && s1 < *s0-tolerance {
			bad = append(bad, fmt.Sprintf("%s satisfaction dropped %.1f -> %.1f (%s)", q.Map, *s0, s1, compact(s.Metrics)))
		}
		if h0 != nil && h1 != nil && *h1 < *h0 {
			bad = append(bad, fmt.Sprintf("%s holdout pass rate dropped %.1f%% -> %.1f%%", q.Map, *h0, *h1))
		}
		lines = append(lines, fmt.Sprintf("| %s | %s → %.1f | %s → %s |", q.Map, orText(s0, "%.1f", "new"), s1, pct(h0), pct(h1)))
	}
	out := append([]string{"## Quality ratchet", ""}, lines...)
	out = append(out, "")
	for _, b := range bad {
		out = append(out, "❌ "+b)
	}
	summary := strings.Join(out, "\n")
	fmt.Println(summary)
	if p := os.Getenv("GITHUB_STEP_SUMMARY"); p != "" {
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return 1, err
		}
		_, werr := f.WriteString(summary + "\n")
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return 1, werr
		}
	}
	if len(bad) > 0 {
		return 1, nil
	}
	return 0, nil
}

func orText(v *float64, format, missing string) string {
	if v == nil {
		return missing
	}
	return fmt.Sprintf(format, *v)
}

func pct(v *float64) string {
	if v == nil {
		return "-"
	}
	return pyFloat(*v) + "%"
}

func baseline(artifacts, out string) error {
	old := map[string]json.RawMessage{}
	if b, err := os.ReadFile(out); err == nil {
		if err := json.Unmarshal(b, &old); err != nil {
			return err
		}
	}
	cur, err := loadArtifacts(artifacts)
	if err != nil {
		return err
	}
	for _, q := range cur {
		var s satisfaction
		if err := json.Unmarshal(q.Satisfaction, &s); err != nil {
			return err
		}
		score := s.Score
		e, err := json.Marshal(baseEntry{Satisfaction: &score, Holdout: q.Holdout.Rate, Metrics: s.Metrics})
		if err != nil {
			return err
		}
		old[q.Map] = e
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return err
	}
	b, err := json.MarshalIndent(old, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		return err
	}
	c, _ := json.Marshal(old)
	fmt.Println("baseline:", string(c))
	return nil
}
