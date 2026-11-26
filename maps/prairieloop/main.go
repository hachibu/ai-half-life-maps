// Command prairieloop generates Prairie Loop Winter: JustinCon After Hours (Half-Life 1 deathmatch): a snowed-in
// prairie loop street; nine basements joined by a ring tunnel; the blue house's second staircase reveals
// JustinCon, a 32-PC neon LAN hall where BattleAxe faces Crescent.
//
//	go run ./maps/prairieloop    (or: make build MAP=prairieloop)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"halflifemaps/maps/prairieloop/layout"
	"halflifemaps/tools/mapkit"
)

const name = "prairieloop"

func main() {
	mapkit.Main(mapkit.Program{Name: name, Generate: generate, Routes: routes, Views: views})
}

// Layout constants (the street plan itself is in package layout).
const (
	xw, xe, ys, yn = layout.XW, layout.XE, layout.YS, layout.YN
	roof           = layout.Roof
	eb             = -512.0         // earth bottom
	bf, bc         = -176.0, -64.0  // basement / tunnel floor and tunnel ceiling
	hf, hc         = -464.0, -48.0  // hall floor and ceiling
	px0, px1       = 464.0, 560.0   // the blue house's reveal passage
	gx0, gy0       = 1280.0, 448.0  // north garage
	gx1, gy1       = 1600.0, 704.0  //
	sdoorX0        = -620.0         // server room door
	sdoorX1        = -500.0         //
	tableH         = hf + 30        // LAN pod desks
	snowGap        = 256            // snowfall sheet spacing
	snowTop        = 800.0          //
	snowBot        = -16.0          //
	srvX0, srvY0   = -700.0, -560.0 // server room
	srvX1, srvY1   = -420.0, -384.0 //
	srvZ0, srvZ1   = hf, -336.0     //
	hallX0, hallY0 = -832.0, -384.0 // the hall
	hallX1, hallY1 = 832.0, 384.0   //
	lin            = 8.0            // hall wall lining
	ix0, ix1       = hallX0 + lin, hallX1 - lin
	iy0, iy1       = hallY0 + lin, hallY1 - lin
)

var (
	park    = [4]float64{-896, -448, 896, 448}
	road    = [4]float64{-1152, -704, 1152, 704} // outer edge of the loop road
	wstreet = [4]float64{-3072, -128, -1152, 128}
	scale   = map[string]float64{texSnow: 4, texSnowroad: 3, texRoofsnow: 3, texTunnelwall: 2, texTunnelfloor: 2,
		texHallwall: 2, texHallceil: 2, texArcadecarpet: 1, texIce: 3, texCeiltile: 2, texCeiling: 2, texWallpaper: 1}
)

type item struct {
	cls     string
	x, y, z float64
}

// gen collects the map as it's built: world brushes, decoration (func_illusionary), entities, pickups, spawns.
type gen struct {
	*mapkit.Maker
	w, decor, glass, snow []string
	ents, bents           []string
	items, spawns         []item
	earthN                int
	weaponN               int    // round-robin over the weapon/ammo lists
	handleSlots           [2]int // CRT handle screens placed per side (west, east)
}

func (g *gen) point(cls string, x, y, z float64, kv ...any) {
	g.ents = append(g.ents, mapkit.Point(cls, x, y, z, kv...))
}

func (g *gen) light(x, y, z float64, rgb [3]int, i int) {
	g.point("light", mapkit.Round(x), mapkit.Round(y), mapkit.Round(z), "_light", fmt.Sprintf("%d %d %d %d", rgb[0], rgb[1], rgb[2], i))
}

func (g *gen) bent(cls string, brushes []string, kv ...any) {
	g.bents = append(g.bents, mapkit.BrushEntity(cls, brushes, kv...))
}

func (g *gen) item(cls string, x, y, z float64) { g.items = append(g.items, item{cls, x, y, z}) }

func (g *gen) spawn(x, y, z float64) { g.spawns = append(g.spawns, item{"", x, y, z + 37}) }

func (g *gen) box(x0, y0, z0, x1, y1, z1 float64, tex string, sides ...mapkit.Sides) string {
	return g.Box(x0, y0, z0, x1, y1, z1, tex, sides...)
}

func (g *gen) sign(tex string, x0, y0, z0, x1, y1, z1 float64, facing string) string {
	return g.Sign(tex, x0, y0, z0, x1, y1, z1, facing, texMetal)
}

// wall cuts openings (a, b, z0, z1) out of a wall running from lo to hi; mk makes the box spanning a..b
// along the wall and z0..z1 in height.
func wall(lo, hi, z0, z1 float64, openings [][4]float64, mk func(a, b, z0, z1 float64) string) []string {
	var out []string
	cur := lo
	for _, o := range sortedOpenings(openings) {
		a, b, oz0, oz1 := o[0], o[1], o[2], o[3]
		if a > cur {
			out = append(out, mk(cur, a, z0, z1))
		}
		if oz0 > z0 {
			out = append(out, mk(a, b, z0, oz0))
		}
		if oz1 < z1 {
			out = append(out, mk(a, b, oz1, z1))
		}
		cur = b
	}
	if cur < hi {
		out = append(out, mk(cur, hi, z0, z1))
	}
	return out
}

// wallX is a wall along x (y0..y1 thick) with openings cut out of it.
func (g *gen) wallX(x0, x1, y0, y1, z0, z1 float64, tex string, openings [][4]float64, sides ...mapkit.Sides) []string {
	return wall(x0, x1, z0, z1, openings, func(a, b, za, zb float64) string { return g.box(a, y0, za, b, y1, zb, tex, sides...) })
}

// wallY is a wall along y (x0..x1 thick) with openings cut out of it.
func (g *gen) wallY(y0, y1, x0, x1, z0, z1 float64, tex string, openings [][4]float64, sides ...mapkit.Sides) []string {
	return wall(y0, y1, z0, z1, openings, func(a, b, za, zb float64) string { return g.box(x0, a, za, x1, b, zb, tex, sides...) })
}

func sortedOpenings(o [][4]float64) [][4]float64 {
	s := append([][4]float64(nil), o...)
	sort.SliceStable(s, func(i, j int) bool {
		for k := range 4 {
			if s[i][k] != s[j][k] {
				return s[i][k] < s[j][k]
			}
		}
		return false
	})
	return s
}

func generate(out string) error {
	sizes, err := mapkit.WriteWAD(filepath.Join(out, name+".wad"), textureList(), mapkit.MipCentre)
	if err != nil {
		return err
	}
	g := &gen{Maker: &mapkit.Maker{Scale: scale, Sizes: sizes}}
	g.earth()
	g.skyShell()
	for _, h := range layout.Houses {
		g.house(h)
	}
	for _, b := range g.glass {
		g.bent("func_breakable", []string{b}, "rendermode", 2, "renderamt", 80, "material", 0, "health", 20)
	}
	g.garages()
	g.roofs()
	g.outdoors()
	g.tunnelDressing()
	g.hall()
	g.restoreThePower()
	g.weather()
	g.weaponsAndSpawns()
	return g.write(out)
}

func (g *gen) write(out string) error {
	for _, it := range g.items {
		g.point(it.cls, mapkit.Round1(it.x), mapkit.Round1(it.y), it.z, "angles", fmt.Sprintf("0 %d 0", angle45(it.x, it.y)))
	}
	for _, s := range g.spawns {
		g.point("info_player_deathmatch", mapkit.Round1(s.x), mapkit.Round1(s.y), s.z, "angles",
			fmt.Sprintf("0 %d 0", mapkit.PyMod(int(mapkit.Deg(atan2(-s.y, -s.x))), 360)))
	}
	for _, part := range mapkit.Chunks(g.decor, 512) {
		g.bent("func_illusionary", part)
	}
	structural, detail := mapkit.SplitDetail(g.w, 256) // small brushes -> func_detail, so full VIS stays fast
	tl, err := mapkit.Texlights(filepath.Join(out, name+".wad"), emissive(), 2)
	if err != nil {
		return err
	}
	world := mapkit.Worldspawn(structural, "mapversion", "220", "wad", name+".wad", "skyname", name,
		"message", "Prairie Loop Winter: JustinCon After Hours")
	text := world + mapkit.DetailEntity(detail) + strings.Join(g.bents, "") + strings.Join(g.ents, "") + tl
	if err := mapkit.WriteMapFile(filepath.Join(out, name+".map"), text); err != nil {
		return err
	}
	// generated assets in the game's valve/ layout (sky, sprites, sounds)
	valve := filepath.Join(out, "valve")
	for _, sub := range []string{"gfx/env", "sprites", "sound/justincon"} {
		if err := os.MkdirAll(filepath.Join(valve, sub), 0o750); err != nil {
			return err
		}
	}
	if err := writeSkybox(filepath.Join(valve, "gfx", "env", name)); err != nil {
		return err
	}
	if err := mapkit.WriteSPR(filepath.Join(valve, "sprites", "prairie_burst.spr"), 128, 128, burstFrames(), snowPal, 64); err != nil {
		return err
	}
	if err := mapkit.WriteSPR(filepath.Join(valve, "sprites", "prairie_steam.spr"), 64, 64, steamFrames(), snowPal, 32); err != nil {
		return err
	}
	if err := makeSounds(filepath.Join(valve, "sound", "justincon")); err != nil {
		return err
	}
	fmt.Printf("world brushes %d (detail %d, earth %d)  decor %d  brush ents %d  items %d  spawns %d  snow sheets %d  point ents %d\n",
		len(structural), len(detail), g.earthN, len(g.decor), len(g.bents), len(g.items), len(g.spawns), len(g.snow), len(g.ents))
	return nil
}
