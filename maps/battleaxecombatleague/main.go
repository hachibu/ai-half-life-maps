// Command battleaxecombatleague generates BattleAxe Combat League - Fight Night, an indoor arena card for
// Half-Life DM. Players spawn in the red / blue corner locker rooms and walk out to the octagon. Look: enclosed
// arena (roof, rigging, glass suites), charcoal panels, red/black branding.
//
//	go run ./maps/battleaxecombatleague    (or: make build MAP=battleaxecombatleague)
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"halflifemaps/tools/mapkit"
)

const name = "battleaxecombatleague"

func main() {
	mapkit.Main(mapkit.Program{Name: name, Generate: generate, Routes: routes, Views: views})
}

var scale = map[string]float64{texArenafloor: 4, texStands: 2, texSeats: 4, texStadwall: 4, texConcrete: 2, texCrowdtop: 4,
	texCeiltile: 2, texRoof: 4, texTruss: 4, texRedstrip: 4, texBluestrip: 4}

// ------------------------------------------------------------------ octagon geometry

var t8 = math.Tan(math.Pi / 8)

// octv are the corners of the octagon with apothem a.
func octv(a float64) []mapkit.V2 {
	b := a * t8
	return []mapkit.V2{{a, -b}, {a, b}, {b, a}, {-b, a}, {-a, b}, {-a, -b}, {-b, -a}, {b, -a}}
}

// segn is the outward normal of octagon side seg.
func segn(seg int) mapkit.V2 {
	return mapkit.V2{math.Cos(float64(seg) * math.Pi / 4), math.Sin(float64(seg) * math.Pi / 4)}
}

// onoct is the point t of the way along side seg of the octagon with apothem a.
func onoct(a float64, seg int, t float64) mapkit.V2 {
	v := octv(a)
	p, q := v[seg], v[(seg+1)%8]
	return p.Add(q.Sub(p).Mul(t))
}

// rq is the quad between apothems a0 and a1 on side seg, from t0 to t1 along it.
func rq(a0, a1 float64, seg int, t0, t1 float64) []mapkit.V2 {
	return []mapkit.V2{onoct(a0, seg, t0), onoct(a1, seg, t0), onoct(a1, seg, t1), onoct(a0, seg, t1)}
}

const (
	shell  = 2800.0 // shell half-size
	skyTop = 1500.0 // roof deck height (indoor arena: no sky)
)

// ringToSquare is the piece between octagon apothem a and the square shell, for one side.
func ringToSquare(a float64, seg int) []mapkit.V2 {
	s := shell
	rays := []mapkit.V2{{s, -s * t8}, {s, s * t8}, {s * t8, s}, {-s * t8, s}, {-s, s * t8}, {-s, -s * t8}, {-s * t8, -s}, {s * t8, -s}}
	corners := map[int]mapkit.V2{1: {s, s}, 3: {-s, s}, 5: {-s, -s}, 7: {s, -s}}
	v := octv(a)
	j := (seg + 1) % 8
	poly := []mapkit.V2{v[seg], rays[seg]}
	if c, ok := corners[seg]; ok {
		poly = append(poly, c)
	}
	return append(poly, rays[j], v[j])
}

func area2(poly []mapkit.V2) float64 {
	s := 0.0
	n := len(poly)
	for i := range n {
		s += poly[i][0]*poly[(i+1)%n][1] - poly[(i+1)%n][0]*poly[i][1]
	}
	return 0.5 * s
}

// clip2 keeps the part of a convex polygon with n·p <= d (nil when too little is left).
func clip2(poly []mapkit.V2, n mapkit.V2, d float64) []mapkit.V2 {
	var out []mapkit.V2
	for i := range poly {
		p, q := poly[i], poly[(i+1)%len(poly)]
		dp, dq := n.Dot(p)-d, n.Dot(q)-d
		if dp <= 1e-6 {
			out = append(out, p)
		}
		if (dp < -1e-6 && dq > 1e-6) || (dp > 1e-6 && dq < -1e-6) {
			out = append(out, p.Add(q.Sub(p).Mul(dp/(dp-dq))))
		}
	}
	if len(out) >= 3 && math.Abs(area2(out)) > 1 {
		return out
	}
	return nil
}

// sideFn picks the texture for a prism side from its outward normal.
type sideFn func(n2 mapkit.V2) mapkit.Tex

// facing is t on the side whose normal points along dir (dot > 0.99) or against it (dot < -0.99, inward), else
// other.
func facing(dir mapkit.V2, inward bool, t mapkit.Tex, other string) sideFn {
	return func(n2 mapkit.V2) mapkit.Tex {
		d := n2.Dot(dir)
		if (inward && d < -0.99) || (!inward && d > 0.99) {
			return t
		}
		return mapkit.T(other)
	}
}

type item struct {
	cls     string
	x, y, z float64
}

type gen struct {
	*mapkit.Maker
	w, crowd, ents, bents []string
	items                 []item
	spawns                int
}

func (g *gen) point(cls string, x, y, z float64, kv ...any) {
	g.ents = append(g.ents, mapkit.Point(cls, x, y, z, kv...))
}

func (g *gen) light(x, y, z float64, rgb [3]int, i int) {
	g.point("light", x, y, z, "_light", fmt.Sprintf("%d %d %d %d", rgb[0], rgb[1], rgb[2], i))
}

func (g *gen) bent(cls string, brushes []string, kv ...any) {
	g.bents = append(g.bents, mapkit.BrushEntity(cls, brushes, kv...))
}

func (g *gen) box(x0, y0, z0, x1, y1, z1 float64, tex string, sides ...mapkit.Sides) string {
	return g.Box(x0, y0, z0, x1, y1, z1, tex, sides...)
}

func (g *gen) sign(tex string, x0, y0, z0, x1, y1, z1 float64, face, back string) string {
	return g.Sign(tex, x0, y0, z0, x1, y1, z1, face, back)
}

// prism extrudes a convex counter-clockwise polygon from z0 to z1. Unnamed top and bottom are texStands.
func (g *gen) prism(poly []mapkit.V2, z0, z1 float64, side sideFn, top, bottom mapkit.Tex) string {
	if z1 <= z0 || area2(poly) <= 1 {
		panic(fmt.Sprintf("bad prism %v %v..%v", poly, z0, z1))
	}
	faces := []mapkit.Face{{N: mapkit.V3{0, 0, 1}, D: z1, Tex: orT(top, mapkit.T(texStands))},
		{N: mapkit.V3{0, 0, -1}, D: -z0 + 0, Tex: orT(bottom, mapkit.T(texStands))}}
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		e := b.Sub(a)
		if e.Norm() < 0.01 {
			continue
		}
		n2 := mapkit.V2{e[1] / e.Norm(), -e[0] / e.Norm()}
		faces = append(faces, mapkit.Face{N: mapkit.V3{n2[0], n2[1], 0}, D: n2.Dot(a), Tex: side(n2)})
	}
	return g.Brush(faces)
}

// prismT is prism with one texture on every side, and on top and bottom unless they're given.
func (g *gen) prismT(poly []mapkit.V2, z0, z1 float64, side, top, bottom mapkit.Tex) string {
	return g.prism(poly, z0, z1, func(mapkit.V2) mapkit.Tex { return side }, orT(top, side), orT(bottom, side))
}

func orT(t, def mapkit.Tex) mapkit.Tex {
	if t.Name == "" {
		return def
	}
	return t
}

var (
	none = mapkit.Tex{}
	tt   = mapkit.T
)

func generate(out string) error {
	sizes, err := mapkit.WriteWAD(filepath.Join(out, name+".wad"), textureList(), mapkit.MipCorner)
	if err != nil {
		return err
	}
	g := &gen{Maker: &mapkit.Maker{Scale: scale, Sizes: sizes}}
	g.shellAndFloor()
	g.lowerBowl()
	g.upperDeck()
	g.south()
	g.north()
	g.octagon()
	g.rig()
	g.floor()
	for _, part := range mapkit.Chunks(g.crowd, 384) { // the crowd: decoration only, no collision; in chunks so each has a small bbox
		g.bent("func_illusionary", part)
	}
	g.pyroAndLights()
	g.weaponsAndSpawns()
	return g.write(out)
}

func (g *gen) write(out string) error {
	structural, detail := mapkit.SplitDetail(g.w, 256) // small brushes -> func_detail, so full VIS stays fast
	// Bigger radiosity patches on the big, plain surfaces: bounced light there is smooth anyway, and patch count drives
	// most of hlrad's time (visibility between patches, form factors, bounces). Direct light keeps full detail.
	// (Not the walls the locker rooms and tunnels are built from: STANDS, STADWALL, CONCRETE. Those rooms are lit
	// mostly by bounces, and bigger patches there visibly darkened them.)
	chop := []any{texSeats, 2, texRoof, 2, texCrowdtop, 2, "CROWDF1", 2, "CROWDF2", 2, "CROWDF3", 2, texMetal, 2, texTruss, 2, texLightbeam, 4}
	tl, err := mapkit.Texlights(filepath.Join(out, name+".wad"), emissive, 2)
	if err != nil {
		return err
	}
	world := mapkit.Worldspawn(structural, "mapversion", "220", "wad", name+".wad", "message", "BattleAxe Combat League - Fight Night")
	chopEnt := mapkit.BrushEntity("info_chopscale", nil, chop...)
	text := world + mapkit.DetailEntity(detail) + strings.Join(g.bents, "") + strings.Join(g.ents, "") + chopEnt + tl
	if err := mapkit.WriteMapFile(filepath.Join(out, name+".map"), text); err != nil {
		return err
	}
	valve := filepath.Join(out, "valve")
	for _, sub := range []string{"gfx/env", "sprites"} {
		if err := os.MkdirAll(filepath.Join(valve, sub), 0o750); err != nil {
			return err
		}
	}
	frames := mapkit.FireFrames(seeded(7), 64, 128, 12)
	bytes := make([][]byte, len(frames))
	for i, f := range frames {
		bytes[i] = mapkit.ToBytes(f)
	}
	if err := mapkit.WriteSPR(filepath.Join(valve, "sprites", name+"_fire.spr"), 64, 128, bytes, mapkit.FirePalette(), 128); err != nil {
		return err
	}
	fmt.Println("world brushes", len(structural), "detail", len(detail), "brush ents", len(g.bents), "items", len(g.items),
		"spawns", g.spawns, "point ents", len(g.ents))
	return nil
}
