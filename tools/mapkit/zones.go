package mapkit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Route is a walk that must stay passable: the walker starts standing on FloorZ, steps up at most 18 units and
// may drop any distance. Doors and breakables are entities, so they're ignored.
type Route struct {
	Name   string
	FloorZ float64
	Points []V2
}

// View is a named preview camera shot.
type View struct {
	Name        string
	Eye, Target V3
}

// Zone is one independent part of a zone-based map, in its own file maps/<map>/zone_<name>.go, so several
// agents can work on one map without merge conflicts. Bounds is the space it may build in (zones must not
// overlap); textures, routes and views are its own; Build adds brushes and entities.
type Zone struct {
	Name     string // set by AddZone from the file name
	Bounds   *[6]float64
	Textures []Texture
	Routes   []Route
	Views    []View
	Build    func(*Ctx)
}

var registered []Zone

// AddZone registers a zone (call it from the zone file's init). The zone is named after its file:
// zone_hall.go is zone "hall".
func AddZone(z Zone) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("AddZone: can't tell which file is calling")
	}
	z.Name = strings.TrimPrefix(strings.TrimSuffix(filepath.Base(file), ".go"), "zone_")
	registered = append(registered, z)
}

// Zones are the registered zones, sorted by name (file order).
func Zones() []Zone {
	z := slices.Clone(registered)
	slices.SortFunc(z, func(a, b Zone) int { return strings.Compare(a.Name, b.Name) })
	return z
}

// Ctx is what a zone's Build uses.
type Ctx struct {
	Zone        string
	World       []string // world brush text
	Entities    []string // whole entity blocks
	targetnames map[string]bool
}

// Name is a zone-prefixed targetname ("door" -> "hall_door"). Use it for every targetname.
func (c *Ctx) Name(short string) string {
	n := c.Zone + "_" + short
	c.targetnames[n] = true
	return n
}

// Brush adds a world brush.
func (c *Ctx) Brush(text string) { c.World = append(c.World, text) }

func (c *Ctx) noteTargetname(kv []any) {
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i] == "targetname" {
			c.targetnames[Val(kv[i+1])] = true
		}
	}
}

// Point adds a point entity (see mapkit.Point).
func (c *Ctx) Point(cls string, x, y, z float64, kv ...any) {
	c.noteTargetname(kv)
	c.Entities = append(c.Entities, Point(cls, x, y, z, kv...))
}

// BrushEntity adds a brush entity (see mapkit.BrushEntity).
func (c *Ctx) BrushEntity(cls string, brushes []string, kv ...any) {
	c.noteTargetname(kv)
	c.Entities = append(c.Entities, BrushEntity(cls, brushes, kv...))
}

var errZones = errors.New("zone conflicts")

func overlap(a, b [6]float64) bool {
	for i := range 3 {
		if !(a[i] < b[i+3] && b[i] < a[i+3]) {
			return false
		}
	}
	return true
}

func checkZones(zones []Zone) []string {
	var errs []string
	owner := map[string]string{}
	for _, z := range zones {
		for _, t := range z.Textures {
			if o, ok := owner[t.Name]; ok {
				errs = append(errs, fmt.Sprintf("texture %s defined by zones %s and %s", t.Name, o, z.Name))
			}
			owner[t.Name] = z.Name
		}
	}
	for i, a := range zones {
		for _, b := range zones[i+1:] {
			if a.Bounds != nil && b.Bounds != nil && overlap(*a.Bounds, *b.Bounds) {
				errs = append(errs, fmt.Sprintf("zones %s and %s overlap: %v vs %v", a.Name, b.Name, *a.Bounds, *b.Bounds))
			}
		}
	}
	return errs
}

// WriteMapFile writes .map text with CRLF line ends.
func WriteMapFile(path, text string) error {
	return os.WriteFile(path, []byte(strings.ReplaceAll(text, "\n", "\r\n")), 0o644)
}

// ZoneMap is a Program for a zone-based map: every registered zone, assembled. The map is named after the
// folder of the file that calls it (so a copied map needs no edits); worldspawn holds extra worldspawn keys
// (pairs); emissive the glowing textures.
func ZoneMap(emissive []Glow, worldspawn ...any) Program {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("ZoneMap: can't tell which file is calling")
	}
	name := filepath.Base(filepath.Dir(file))
	return Program{
		Name: name,
		Generate: func(out string) error {
			return buildZones(name, out, Zones(), emissive, worldspawn)
		},
		Routes: func() []Route {
			var r []Route
			for _, z := range Zones() {
				r = append(r, z.Routes...)
			}
			return r
		},
		Views: func() []View {
			var v []View
			for _, z := range Zones() {
				v = append(v, z.Views...)
			}
			return v
		},
	}
}

func buildZones(name, out string, zones []Zone, emissive []Glow, worldspawn []any) error {
	errs := checkZones(zones)
	ctxs := make([]*Ctx, len(zones))
	for i, z := range zones {
		ctxs[i] = &Ctx{Zone: z.Name, targetnames: map[string]bool{}}
		if z.Build != nil {
			z.Build(ctxs[i])
		}
	}
	seen := map[string]string{}
	for _, c := range ctxs {
		names := make([]string, 0, len(c.targetnames))
		for t := range c.targetnames {
			names = append(names, t)
		}
		slices.Sort(names)
		for _, t := range names {
			if o, ok := seen[t]; ok {
				errs = append(errs, fmt.Sprintf("targetname %s used by zones %s and %s", t, o, c.Zone))
			}
			seen[t] = c.Zone
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w:\n  %s", errZones, strings.Join(errs, "\n  "))
	}
	var texs []Texture
	var zoneNames []string
	for _, z := range zones {
		texs = append(texs, z.Textures...)
		zoneNames = append(zoneNames, z.Name)
	}
	wad := filepath.Join(out, name+".wad")
	if _, err := WriteWAD(wad, texs, MipCentre); err != nil {
		return err
	}
	var world, ents []string
	for _, c := range ctxs {
		world = append(world, c.World...)
		ents = append(ents, c.Entities...)
	}
	tl, err := Texlights(wad, emissive, 2)
	if err != nil {
		return err
	}
	ws := Worldspawn(world, append([]any{"mapversion", "220", "wad", name + ".wad"}, worldspawn...)...)
	if err := WriteMapFile(filepath.Join(out, name+".map"), ws+strings.Join(ents, "")+tl); err != nil {
		return err
	}
	fmt.Printf("%s: %d zones (%s), %d brushes, %d entities, %d textures\n", name, len(zones),
		strings.Join(zoneNames, ", "), len(world), len(ents), len(texs))
	return nil
}
