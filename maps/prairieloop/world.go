package main

import (
	"fmt"
	"math"
	"slices"

	"halflifemaps/maps/prairieloop/layout"
	"halflifemaps/tools/mapkit"
)

func atan2(y, x float64) float64 { return math.Atan2(y, x) }

func angle45(x, y float64) int { return int(math.Abs(x*3+y)) % 8 * 45 }

// ------------------------------------------------------------------ earth with holes (tunnels, hall, basements)

type hole struct {
	x0, y0, x1, y1, zl, zh float64
	floor, ceil            string // texture of the solid's top below the hole and bottom above it ("" = default)
}

func holes() []hole {
	var hs []hole
	for _, h := range layout.Houses {
		f := layout.Footprint(h)
		hs = append(hs, hole{f[0], f[1], f[2], f[3], -192, 0, "", ""})
	}
	tun := func(x0, y0, x1, y1 float64) hole { return hole{x0, y0, x1, y1, bf, bc, texTunnelfloor, texTunnelwall} }
	for _, r := range [][4]float64{{-1216, 512, 1216, 640}, {-1216, -640, 1216, -512}, {-1088, -512, -960, 512}, {960, -512, 1088, 512}} {
		hs = append(hs, tun(r[0], r[1], r[2], r[3]))
	}
	for _, h := range layout.Houses {
		if h.Rot == 1 {
			hs = append(hs, tun(1088, -48, 1280, 48))
			continue
		}
		cx := h.CX
		if h.Blue {
			cx -= 80
		}
		y0, y1 := 640.0, 832.0
		if h.Rot != 0 {
			y0, y1 = -832, -640
		}
		hs = append(hs, tun(cx-48, y0, cx+48, y1))
	}
	hs = append(hs, tun(-960, -48, -832, 48), tun(832, -48, 960, 48)) // BattleAxe and Crescent connectors
	hs = append(hs, hole{hallX0, hallY0, hallX1, hallY1, hf, hc, texArcadecarpet, texHallceil})
	hs = append(hs, hole{srvX0, srvY0, srvX1, srvY1, srvZ0, srvZ1, texTunnelfloor, texTunnelwall})
	for j := 1; j <= 18; j++ { // the blue house's reveal stairs: 8-unit risers, 12-unit treads
		zh := -48.0
		if j >= 17 {
			zh = -192
		}
		fj := float64(j)
		hs = append(hs, hole{px0, 832 - 12*fj, px1, 844 - 12*fj, bf - 8*fj, zh, texStep, texTunnelwall})
	}
	return append(hs, hole{px0, 384, px1, 616, -320, -192, texTunnelfloor, texTunnelwall})
}

func inRect(r [4]float64, x, y float64) bool { return r[0] <= x && x <= r[2] && r[1] <= y && y <= r[3] }

func surfaceTex(x, y float64) string {
	if (inRect(road, x, y) && !inRect(park, x, y)) || inRect(wstreet, x, y) {
		return texSnowroad
	}
	return texSnow
}

type solid struct {
	z0, z1    float64
	top, bott string
}

// cellSig is the stack of solid earth in one grid cell, between the holes that pass through it.
func cellSig(hs []hole, xa, xb, ya, yb float64) []solid {
	mx, my := (xa+xb)/2, (ya+yb)/2
	var in []hole
	for _, h := range hs {
		if h.x0 <= mx && mx <= h.x1 && h.y0 <= my && my <= h.y1 {
			in = append(in, h)
		}
	}
	slices.SortStableFunc(in, func(a, b hole) int { return cmpF(a.zl, b.zl) })
	var out []solid
	z, ftop := eb, ""
	for _, h := range in {
		if h.zl > z {
			out = append(out, solid{z, h.zl, h.floor, ftop})
		}
		ftop = h.ceil
		z = max(z, h.zh)
	}
	if z < 0 {
		out = append(out, solid{z, 0, surfaceTex(mx, my), ftop})
	}
	return out
}

func cmpF(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func uniqSorted(v []float64) []float64 {
	slices.Sort(v)
	return slices.Compact(v)
}

type run struct {
	xa, xb float64
	sig    []solid
}

type rect struct {
	ya, yb float64
	boxes  []string
	alive  bool
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// earth fills the ground below z = 0 with solid boxes, leaving the holes open, merging equal cells into strips
// along x and then strips into rectangles along y.
func (g *gen) earth() {
	hs := holes()
	xs := []float64{xw, xe, road[0], road[2], park[0], park[2]}
	ysl := []float64{ys, yn, road[1], road[3], park[1], park[3], wstreet[1], wstreet[3]}
	for _, h := range hs {
		xs = append(xs, h.x0, h.x1)
		ysl = append(ysl, h.y0, h.y1)
	}
	xs, ysl = uniqSorted(xs), uniqSorted(ysl)
	byKey := map[string]*rect{}
	var order []*rect
	for yi := range len(ysl) - 1 {
		ya, yb := ysl[yi], ysl[yi+1]
		var runs []run
		for xi := range len(xs) - 1 {
			xa, xb := xs[xi], xs[xi+1]
			sg := cellSig(hs, xa, xb, ya, yb)
			if n := len(runs); n > 0 && slices.Equal(runs[n-1].sig, sg) && runs[n-1].xb == xa {
				runs[n-1].xb = xb
			} else {
				runs = append(runs, run{xa, xb, sg})
			}
		}
		for _, r := range runs {
			k := fmt.Sprint(r.xa, r.xb, r.sig)
			if c, ok := byKey[k]; ok && c.yb == ya {
				c.yb = yb
			} else {
				if ok {
					g.w = append(g.w, c.boxes...)
					c.alive = false
				}
				c = &rect{ya: ya, yb: yb, alive: true}
				byKey[k] = c
				order = append(order, c)
			}
			c := byKey[k]
			c.boxes = nil
			for _, s := range r.sig {
				if s.z1 > s.z0 {
					c.boxes = append(c.boxes, g.box(r.xa, c.ya, s.z0, r.xb, c.yb, s.z1, texTunnelwall,
						mapkit.Sides{Top: mapkit.T(or(s.top, texTunnelwall)), Bottom: mapkit.T(or(s.bott, texTunnelwall))}))
				}
			}
		}
	}
	for _, c := range order {
		if c.alive {
			g.w = append(g.w, c.boxes...)
		}
	}
	g.earthN = len(g.w)
}

// ------------------------------------------------------------------ sky shell

func (g *gen) skyShell() {
	const sk0, sk1 = eb - 16, 1100.0
	for _, b := range [][6]float64{{xw - 32, ys - 32, sk1, xe + 32, yn + 32, sk1 + 16}, {xw - 32, ys - 32, sk0 - 16, xe + 32, yn + 32, sk0},
		{xw - 32, ys - 32, sk0, xw, yn + 32, sk1}, {xe, ys - 32, sk0, xe + 32, yn + 32, sk1},
		{xw, ys - 32, sk0, xe, ys, sk1}, {xw, yn, sk0, xe, yn + 32, sk1}} {
		g.w = append(g.w, g.box(b[0], b[1], b[2], b[3], b[4], b[5], "sky"))
	}
}

// ------------------------------------------------------------------ houses

// fmap turns a house-local face name into the world face for each rotation.
var fmap = map[int]map[string]string{
	0: {"n": "n", "s": "s", "e": "e", "w": "w"},
	2: {"n": "s", "s": "n", "e": "w", "w": "e"},
	1: {"n": "e", "s": "w", "e": "s", "w": "n"},
}

// faces is a box's per-face textures in house-local terms (sideTop, sideBottom, "n", "s", "e", "w").
type faces map[string]string

type houseKit struct {
	g *gen
	h layout.House
}

// b adds a house-local box to dest (the world unless given).
func (k houseKit) b(x0, y0, z0, x1, y1, z1 float64, tex string, f faces, dest ...*[]string) {
	ax, ay := layout.L2W(k.h, x0, y0)
	bx, by := layout.L2W(k.h, x1, y1)
	var s mapkit.Sides
	for _, key := range []string{sideTop, sideBottom, "n", "s", "e", "w"} {
		if t, ok := f[key]; ok {
			wk := key
			if m, ok := fmap[k.h.Rot][key]; ok {
				wk = m
			}
			s = mapkit.SetSide(s, wk, mapkit.T(t))
		}
	}
	d := &k.g.w
	if len(dest) > 0 {
		d = dest[0]
	}
	*d = append(*d, k.g.box(ax, ay, z0, bx, by, z1, tex, s))
}

// s adds a house-local sign facing the house-local direction facing.
func (k houseKit) s(tex string, x0, y0, z0, x1, y1, z1 float64, facing string, dest *[]string, back string) {
	ax, ay := layout.L2W(k.h, x0, y0)
	bx, by := layout.L2W(k.h, x1, y1)
	*dest = append(*dest, k.g.Sign(tex, ax, ay, z0, bx, by, z1, fmap[k.h.Rot][facing], back))
}

func (k houseKit) light(x, y, z float64, rgb [3]int, i int) {
	px, py := layout.L2W(k.h, x, y)
	k.g.light(px, py, z, rgb, i)
}

func (g *gen) house(h layout.House) {
	k := houseKit{g, h}
	top := "CARPET"
	if h.Blue {
		top = texArcadecarpet
	}
	k.b(-256, -224, -192, 256, 224, bf, texTunnelwall, faces{sideTop: top}) // basement shell
	var doors [][4]float64
	switch {
	case h.Rot == 1:
		doors = [][4]float64{{-48, 48, bf, bc}}
	case h.Rot == 2:
		doors = [][4]float64{{-48, 48, bf, bc}}
	case h.Blue:
		doors = [][4]float64{{-128, -32, bf, bc}}
	default:
		doors = [][4]float64{{-48, 48, bf, bc}}
	}
	if h.Blue {
		doors = append(doors, [4]float64{px0 - h.CX, px1 - h.CX, bf, bc})
	}
	// front basement wall (local y -224..-208) with doors, back wall, sides
	cur := -256.0
	for _, d := range sortedOpenings(doors) {
		if d[0] > cur {
			k.b(cur, -224, bf, d[0], -208, 0, texTunnelwall, faces{"n": texBasewall})
		}
		k.b(d[0], -224, d[3], d[1], -208, 0, texTunnelwall, faces{"n": texBasewall})
		cur = d[1]
	}
	k.b(cur, -224, bf, 256, -208, 0, texTunnelwall, faces{"n": texBasewall})
	k.b(-256, 208, bf, 256, 224, 0, texTunnelwall, faces{"s": texBasewall})
	k.b(-256, -208, bf, -240, 208, 0, texTunnelwall, faces{"e": texBasewall})
	k.b(240, -208, bf, 256, 208, 0, texTunnelwall, faces{"w": texBasewall})
	// ground slab with stair opening, stairs, rails
	k.b(-240, -208, -16, 240, 128, 0, texWoodfloor, faces{sideBottom: texCeiltile})
	k.b(-240, 128, -16, -40, 208, 0, texWoodfloor, faces{sideBottom: texCeiltile})
	k.b(200, 128, -16, 240, 208, 0, texWoodfloor, faces{sideBottom: texCeiltile})
	for j := 1.0; j <= 21; j++ { // 8-unit risers down to the basement floor (bf = -176)
		k.b(200-12*j, 128, bf, 212-12*j, 208, -8*j, texBasewall, faces{sideTop: texStep})
	}
	k.b(-40, 124, 0, 200, 128, 36, texTrim, nil, &g.decor)
	g.houseWalls(k)
	g.houseInterior(k)
}

func (g *gen) houseWalls(k houseKit) {
	const gh = 128.0
	sd := k.h.Siding
	wallLocal := func(x0, x1, y0, y1 float64, openings [][4]float64, inner string) {
		f := faces{inner: texWallpaper}
		cur := x0
		for _, o := range sortedOpenings(openings) {
			a, b, z0, z1 := o[0], o[1], o[2], o[3]
			if a > cur {
				k.b(cur, y0, 0, a, y1, gh, sd, f)
			}
			if z0 > 0 {
				k.b(a, y0, 0, b, y1, z0, sd, f)
			}
			k.b(a, y0, z1, b, y1, gh, sd, f)
			cur = b
		}
		if cur < x1 {
			k.b(cur, y0, 0, x1, y1, gh, sd, f)
		}
	}
	wins := [][4]float64{{-200, -104, 40, 100}, {104, 200, 40, 100}}
	wallLocal(-256, 256, -224, -208, append([][4]float64{{-40, 40, 0, 112}}, wins...), "n")
	wallLocal(-256, 256, 208, 224, [][4]float64{{-160, -64, 40, 100}}, "s")
	sideWall := func(x0, x1 float64, inner string) {
		f := faces{inner: texWallpaper}
		k.b(x0, -208, 0, x1, -120, gh, sd, f)
		k.b(x0, -120, 0, x1, -24, 40, sd, f)
		k.b(x0, -120, 100, x1, -24, gh, sd, f)
		k.b(x0, -24, 0, x1, 208, gh, sd, f)
	}
	sideWall(-256, -240, "e")
	sideWall(240, 256, "w")
	for _, p := range [][6]float64{{-200, -218, -104, -214, 40, 100}, {104, -218, 200, -214, 40, 100},
		{-160, 214, -64, 218, 40, 100}, {-250, -120, -246, -24, 40, 100}, {246, -120, 250, -24, 40, 100}} {
		ax, ay := layout.L2W(k.h, p[0], p[1])
		bx, by := layout.L2W(k.h, p[2], p[3])
		g.glass = append(g.glass, g.box(ax, ay, p[4], bx, by, p[5], "GLASS"))
	}
	// roof slab (walkable, snowy) + chimney for cover
	k.b(-256, -224, gh, 256, 224, roof, texTrim, faces{sideTop: texRoofsnow, sideBottom: texCeiling})
	k.b(-190, 40, roof, -140, 90, 196, "BRICK", faces{sideTop: texRoofsnow})
}

func (g *gen) houseInterior(k houseKit) {
	const gh = 128.0
	h := k.h
	// partition, living room, kitchen
	k.b(40, -208, 0, 56, -20, gh, texWallpaper, nil)
	k.b(-236, -150, 0, -200, -30, 36, texCouch, nil)
	k.b(-240, -150, 0, -228, -30, 56, texCouch, nil)
	k.s(texNewscast, 30, -140, 36, 40, -80, 76, "w", &g.w, "TVBLACK")
	k.b(-140, -120, 0, -90, -60, 18, texStep, nil)
	k.b(200, -200, 0, 240, 60, 36, texTrim, faces{sideTop: "COUNTER"})
	k.b(206, 70, 0, 240, 110, 80, "FRIDGE", nil)
	// porch
	k.b(-64, -272, 0, 64, -224, 8, texTrim, faces{sideTop: texWoodfloor})
	k.b(56, -232, 92, 68, -224, 110, texMetal, faces{"s": texLamp}, &g.decor)
	k.light(62, -250, 98, [3]int{255, 190, 120}, 150)
	k.light(-100, -60, 100, [3]int{255, 225, 180}, 120)
	k.light(130, 40, 100, [3]int{255, 235, 200}, 100)
	// basement dressing
	if h.Blue { // the small computer room
		for _, y0 := range []float64{-170, -90, -10} {
			k.b(-240, y0, bf, -200, y0+70, bf+30, texDesk, nil)
			screen := texCrtSaver
			if y0 == -90 {
				screen = texCrtTerm
			}
			k.s(screen, -236, y0+20, bf+34, -232, y0+50, bf+58, "e", &g.decor, texCrt)
			k.b(-240, y0+18, bf+30, -236, y0+52, bf+60, texCrt, nil)
		}
		k.s("POSTER98", -80, 204, bf+40, -16, 208, bf+136, "s", &g.decor, texMetal)
		k.s(texTsJc, px0-h.CX, -208, bc+4, px1-h.CX, -204, bc+28, "n", &g.decor, texMetal)
		k.light(-120, -60, -60, [3]int{120, 170, 255}, 140)
		k.light(120, -150, -60, [3]int{255, 110, 210}, 90)
		px, py := layout.L2W(h, -120, -60)
		g.point("ambient_generic", px, py, -100, "message", "justincon/fans_loop.wav", "health", 6, "spawnflags", 2, "pitch", 100, "pitchstart", 100)
	} else {
		k.b(-232, -150, bf, -196, -30, bf+36, texCouch, nil)
		k.b(-60, -60, bf, 60, 20, bf+30, texDesk, nil)
		k.light(0, -60, -60, [3]int{255, 220, 180}, 120)
	}
	if h.Name == "S1" || h.Name == "N3" {
		emb := texEmbMoon
		if h.CX < 0 {
			emb = texEmbAxe
		}
		k.s(emb, -32, -208, bc+8, 32, -204, bc+56, "n", &g.decor, texMetal)
	}
}

// ------------------------------------------------------------------ garages

func (g *gen) garages() {
	g.w = append(g.w, g.box(gx0, gy0, roof-16, gx1, gy1, roof, texSidingGray, mapkit.Sides{Top: mapkit.T(texRoofsnow), Bottom: mapkit.T(texCeiling)}),
		g.box(gx0, gy0, 0, gx1, gy0+16, roof-16, texSidingGray, mapkit.Sides{N: mapkit.T(texTunnelwall)}),
		g.box(gx0, gy1-16, 0, gx1, gy1, roof-16, texSidingGray, mapkit.Sides{S: mapkit.T(texTunnelwall)}),
		g.box(gx1-16, gy0+16, 0, gx1, gy1-16, roof-16, texSidingGray, mapkit.Sides{W: mapkit.T(texTunnelwall)}))
	g.w = append(g.w, g.wallY(gy0+16, gy1-16, gx0, gx0+16, 0, roof-16, texSidingGray, [][4]float64{{496, 656, 0, 112}}, mapkit.Sides{E: mapkit.T(texTunnelwall)})...)
	g.w = append(g.w, g.box(gx0+16, gy0+16, 0, gx1-16, gy1-16, 1, texTunnelfloor))
	s := layout.Garages[1]
	g.w = append(g.w, g.box(s[0], s[1], 0, s[2], s[3], roof, texSidingGray,
		mapkit.Sides{Top: mapkit.T(texRoofsnow), W: g.FitBox(texGaragedoor, "w", s[0], s[1], s[2], s[3], 0, roof)}))
}

// garageDoor is the north garage's breakable door.
func (g *gen) garageDoor() string {
	return g.box(gx0+2, 496, 0, gx0+14, 656, 112, texSidingGray, mapkit.Sides{W: g.FitBox(texGaragedoor, "w", gx0, 496, gx0+2, 656, 0, 112)})
}

// ------------------------------------------------------------------ roofs: bridges, exterior stairs, snowbank

func (g *gen) bridge(x0, y0, x1, y1 float64) {
	g.w = append(g.w, g.box(x0, y0, roof-16, x1, y1, roof, texPlank, mapkit.Sides{Top: mapkit.T(texRoofsnow)}))
	if x1-x0 > y1-y0 {
		g.decor = append(g.decor, g.box(x0, y0, roof, x1, y0+4, roof+32, texPlank), g.box(x0, y1-4, roof, x1, y1, roof+32, texPlank))
		g.w = append(g.w, g.box(x0, y0, roof, x1, y0+4, roof+32, texPlank), g.box(x0, y1-4, roof, x1, y1, roof+32, texPlank))
	} else {
		g.w = append(g.w, g.box(x0, y0, roof, x0+4, y1, roof+32, texPlank), g.box(x1-4, y0, roof, x1, y1, roof+32, texPlank))
	}
}

func (g *gen) roofs() {
	hx := layout.HouseXs
	for i := range len(hx) - 1 {
		a1, b0 := hx[i][1], hx[i+1][0]
		g.bridge(a1, 1024, b0, 1088)
		g.bridge(a1, -1088, b0, -1024)
	}
	g.bridge(1312, 704, 1376, 832)
	g.bridge(1408, 256, 1472, 448)
	g.bridge(1408, -448, 1472, -256)
	g.bridge(1312, -832, 1376, -704)
	snowTop := mapkit.Sides{Top: mapkit.T(texRoofsnow)}
	for j := 1.0; j <= 18; j++ { // exterior stairs (8-unit risers): N0 west side
		g.w = append(g.w, g.box(-1472, 1000+12*(j-1), 0, -1408, 1000+12*j, 8*j, texPlank, snowTop))
	}
	for j := 1.0; j <= 18; j++ { // exterior stairs: east house south side
		g.w = append(g.w, g.box(1700-12*j, -320, 0, 1712-12*j, -256, 8*j, texPlank, snowTop))
	}
	g.w = append(g.w, g.Hull([]mapkit.V3{{-1760, -1240, 0}, {-1760, -1112, 0}, {-1408, -1240, 0}, {-1408, -1112, 0},
		{-1408, -1240, roof}, {-1408, -1112, roof}}, texSnow, texSnow)) // climbable snowbank
}
