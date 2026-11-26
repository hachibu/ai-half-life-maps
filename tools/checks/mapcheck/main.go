// mapcheck: the map checks `make test` runs (Go port of the Python structure/placements/wiring/walk checks,
// same output).
//
//	mapcheck structure  MAP.bsp            engine limits + file consistency
//	mapcheck placements MAP.bsp            spawns stand clear and grounded; pickups out of solid
//	mapcheck wiring     MAP.bsp            every target / multi_manager key names an existing targetname
//	mapcheck routes     MAP.bsp ROUTES.json   walk routes ([[name, floor_z, [[x, y], ...]], ...])
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"halflifemaps/tools/bsp/bspfile"
	"halflifemaps/tools/checks/walker"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mapcheck structure|placements|wiring MAP.bsp | mapcheck routes MAP.bsp ROUTES.json")
		os.Exit(2)
	}
	m, err := bspfile.Load(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var bad bool
	switch os.Args[1] {
	case "structure":
		bad = structure(m)
	case "placements":
		bad = placements(m)
	case "wiring":
		bad = wiring(m)
	case "routes":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "routes needs ROUTES.json")
			os.Exit(2)
		}
		bad = routes(m, os.Args[3])
	default:
		fmt.Fprintln(os.Stderr, "unknown check", os.Args[1])
		os.Exit(2)
	}
	if bad {
		os.Exit(1)
	}
}

// pyStr mirrors Python's repr() of a str (quotes chosen the way Python does).
func pyStr(s string) string {
	q := "'"
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		q = `"`
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	if q == "'" {
		s = strings.ReplaceAll(s, "'", `\'`)
	}
	return q + s + q
}

// pyFloat mirrors Python's repr() of a float.
func pyFloat(x float64) string {
	s := strconv.FormatFloat(x, 'g', -1, 64)
	if strings.ContainsAny(s, "e") {
		mant, exp, _ := strings.Cut(s, "e")
		e, _ := strconv.Atoi(exp)
		if e >= -4 && e < 16 {
			s = strconv.FormatFloat(x, 'f', -1, 64)
		} else {
			sign := "+"
			if e < 0 {
				sign, e = "-", -e
			}
			return fmt.Sprintf("%se%s%02d", mant, sign, e)
		}
	}
	if !strings.ContainsAny(s, ".eE") && s != "inf" && s != "-inf" && s != "NaN" {
		s += ".0"
	}
	return s
}

func structure(m *bspfile.BSP) bool {
	for _, l := range bspfile.Limits {
		v := m.Counts[l.Key]
		st := "OK"
		if v >= l.Max {
			st = "OVER"
		}
		fmt.Printf("  %-14s %8d / %-7d %s\n", l.Label, v, l.Max, st)
	}
	set := map[string]bool{}
	for _, e := range m.Errors {
		set[e] = true
	}
	var errs []string
	for e := range set {
		errs = append(errs, e)
	}
	sort.Strings(errs)
	shown := errs
	if len(shown) > 10 {
		shown = shown[:10]
	}
	parts := make([]string, len(shown))
	for i, e := range shown {
		parts[i] = pyStr(e)
	}
	fmt.Printf("structure errors: %d [%s]\n", len(errs), strings.Join(parts, ", "))
	return len(errs) > 0
}

func origin(e bspfile.Entity) ([3]float64, bool) {
	f := strings.Fields(e.Get("origin"))
	var p [3]float64
	if len(f) < 3 {
		return p, false
	}
	for i := range p {
		v, err := strconv.ParseFloat(f[i], 64)
		if err != nil {
			return p, false
		}
		p[i] = v
	}
	return p, true
}

func placements(m *bspfile.BSP) bool {
	head1 := m.WorldHeads()[1]
	bad, spawns, items := 0, 0, 0
	for _, e := range m.Entities {
		cls := e.Get("classname")
		if !e.Has("origin") {
			continue
		}
		p, ok := origin(e)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(cls, "info_player_"):
			spawns++
			free := m.HullC(head1, p) == -1
			grounded := m.HullC(head1, [3]float64{p[0], p[1], p[2] - 20}) == -2
			if !free || !grounded {
				bad++
				f, g := "free", "grounded"
				if !free {
					f = "IN SOLID"
				}
				if !grounded {
					g = "FLOATING"
				}
				fmt.Println("BAD SPAWN", cls, pyFloat(p[0]), pyFloat(p[1]), pyFloat(p[2]), f, g)
			}
		case strings.HasPrefix(cls, "weapon_") || strings.HasPrefix(cls, "ammo_") || strings.HasPrefix(cls, "item_"):
			items++
			if m.Solid(p) {
				bad++
				fmt.Println("ITEM IN SOLID", cls, pyFloat(p[0]), pyFloat(p[1]), pyFloat(p[2]))
			}
		}
	}
	fmt.Printf("spawns: %d  items: %d  placement problems: %d\n", spawns, items, bad)
	return bad > 0
}

var mmSkip = map[string]bool{"classname": true, "targetname": true, "origin": true, "spawnflags": true, "wait": true, "angles": true}

func wiring(m *bspfile.BSP) bool {
	names := map[string]bool{}
	for _, e := range m.Entities {
		if e.Has("targetname") {
			names[e.Get("targetname")] = true
		}
	}
	bad := 0
	for _, e := range m.Entities {
		var refs []string
		if t := e.Get("target"); t != "" {
			refs = append(refs, t)
		}
		if e.Get("classname") == "multi_manager" {
			for _, k := range e.Keys {
				if !mmSkip[k] {
					refs = append(refs, strings.SplitN(k, "#", 2)[0])
				}
			}
		}
		for _, r := range refs {
			if !names[r] {
				bad++
				fmt.Printf("MISSING target %-16s <- %s\n", r, e.Get("classname"))
			}
		}
	}
	fmt.Printf("named entities: %d  wiring problems: %d\n", len(names), bad)
	return bad > 0
}

type route struct {
	Name  string
	Floor float64
	Pts   [][2]float64
}

func (r *route) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 3 {
		return fmt.Errorf("route must be [name, floor_z, points]")
	}
	if err := json.Unmarshal(raw[0], &r.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[1], &r.Floor); err != nil {
		return err
	}
	return json.Unmarshal(raw[2], &r.Pts)
}

func routes(m *bspfile.BSP, path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var rs []route
	if err := json.Unmarshal(data, &rs); err != nil {
		fmt.Fprintln(os.Stderr, "bad routes file:", err)
		os.Exit(2)
	}
	w := walker.Walker{M: m}
	bad := 0
	for _, r := range rs {
		ok, why, end := w.Walk(r.Floor, r.Pts)
		res := "FAIL " + why
		if ok {
			res = fmt.Sprintf("OK   floor %5.0f -> %5.0f", r.Floor, end)
		} else {
			bad++
		}
		fmt.Printf("%-48s %s\n", r.Name, res)
	}
	fmt.Printf("routes: %d  failed: %d\n", len(rs), bad)
	return bad > 0
}
