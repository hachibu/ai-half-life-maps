package main

import (
	"fmt"
	"math"
	"slices"

	"halflifemaps/tools/mapkit"
)

const (
	a0, dep, nt  = 900.0, 48.0, 20 // first tier apothem, tier depth, tier count
	conc0, conc1 = a0 + dep*nt, 2040.0
	concZ        = 64 + 16*(nt-1) // concourse height: the last lower tier's
	tunX, roomX  = 64.0, 256.0    // blue tunnel / blue locker room half widths (side 2)
	roomK0       = 12             // tiers 12..18 hold the blue locker room underneath
	roomK1       = 19             //
	tunTop       = 128.0          //
	roomTop      = 176.0          //
	b0, b1       = 1980.0, 2076.0 // balcony walkway (overhangs the concourse 1980..2040)
	balZ         = concZ + 8*30   // 608: 30 steps of 8 up from the concourse
	und          = 560.0          // balcony underside
	ua0, unt     = b1, 12         // upper deck
	utop         = ua0 + dep*unt  // 2652
	suiteZ0      = 900.0          // trim ledge must clear a player (72) on the top row (floor 800)
	suiteZ1      = 1020.0         //
	stA0, ledA   = 1300.0, 1716.0 // walkout stage, LED wall
	ca, fa, pa   = 300.0, 304.0, 340.0
	pz           = 36.0 // octagon platform height
	tz           = 560.0
	jz0, jz1     = 700.0, 860.0
	rz0          = 24.0 // red locker room floor and ceiling
	rz1          = 184.0
)

var aisles = [][2]float64{{0.04, 0.46}, {0.54, 0.96}}

func lh(k int) float64 { return float64(64 + 16*k) }       // lower tier height (tier 0 = 64-high fascia)
func uh(k int) float64 { return balZ + float64(16*(k+1)) } // upper tier height

func ribbonSpec(name string, su, sv, ov float64) mapkit.Tex {
	return tt(name).WithScale(su, sv).WithOV(ov)
}

func tierSide(seg int, ribbon string, zt float64) sideFn {
	return facing(segn(seg), true, ribbonSpec(ribbon, 2, 2, zt/2), texStands)
}

// crowdSpec is a crowd row's texture. Rows in the stands use the half-resolution CROWDF* at scale 2.5 (a quarter
// of the light samples); near rows (the floor rows by the octagon) keep full resolution.
func crowdSpec(seed int, ztop float64, near bool) mapkit.Tex {
	if near {
		return tt(fmt.Sprintf("CROWD%d", seed%3+1)).WithScale(1.25, 1.25).WithOV(ztop / 1.25).WithOU(float64((seed * 37) % 128))
	}
	return tt(fmt.Sprintf("CROWDF%d", seed%3+1)).WithScale(2.5, 2.5).WithOV(ztop / 2.5).WithOU(float64((seed*37)%128) / 2)
}

type part struct {
	poly []mapkit.V2
	cav  float64 // cavity top, or 0 for none
}

// seg2Parts splits a side-2 polygon at the tunnel / locker room x-planes.
func seg2Parts(poly []mapkit.V2, k int) []part {
	var xa, xb, cav float64
	switch {
	case k < roomK0:
		xa, xb, cav = -tunX, tunX, tunTop
	case k < roomK1:
		xa, xb, cav = -roomX, roomX, roomTop
	default:
		return []part{{poly, 0}}
	}
	var out []part
	if l := clip2(poly, mapkit.V2{1, 0}, xa); l != nil {
		out = append(out, part{l, 0})
	}
	if r := clip2(poly, mapkit.V2{-1, 0}, -xb); r != nil {
		out = append(out, part{r, 0})
	}
	if m := clip2(poly, mapkit.V2{1, 0}, xb); m != nil {
		if m = clip2(m, mapkit.V2{-1, 0}, -xa); m != nil {
			out = append(out, part{m, cav})
		}
	}
	return out
}

func parts(poly []mapkit.V2, seg, k int) []part {
	if seg != 2 {
		return []part{{poly, 0}}
	}
	return seg2Parts(poly, k)
}

// ------------------------------------------------------------------ shell + floor

func (g *gen) shellAndFloor() {
	const s = shell
	g.w = append(g.w, g.box(-s-16, -s-16, skyTop, s+16, s+16, skyTop+16, texStadwall, mapkit.Sides{Bottom: tt(texRoof)}),
		g.box(-s-16, -s-16, -48, s+16, s+16, -32, texStadwall),
		g.box(s, -s-16, -32, s+16, s+16, skyTop, texStadwall),
		g.box(-s-16, -s-16, -32, -s, s+16, skyTop, texStadwall),
		g.box(-s, s, -32, s, s+16, skyTop, texStadwall),
		g.box(-s, -s-16, -32, s, -s, skyTop, texStadwall),
		g.box(-s, -s, -32, s, s, 0, texArenafloor))
}

// ------------------------------------------------------------------ lower bowl

func (g *gen) lowerBowl() {
	for seg := range 8 {
		if seg == 6 { // walkout stage side
			continue
		}
		for k := range nt {
			ta, tb := a0+dep*float64(k), a0+dep*float64(k+1)
			h := lh(k)
			for _, p := range parts(rq(ta, tb, seg, 0, 1), seg, k) {
				side := func(mapkit.V2) mapkit.Tex { return tt(texStands) }
				if k == 0 { // the first tier's fascia carries an LED ribbon
					rib := texRibbon3
					if seg%2 == 0 {
						rib = texRibbon1
					}
					side = tierSide(seg, rib, h)
				}
				switch {
				case p.cav == 0:
					g.w = append(g.w, g.prism(p.poly, 0, h, side, tt(texSeats), none))
				case h > p.cav+16:
					bottom := "TUNNELWALL"
					if p.cav == roomTop {
						bottom = texCeiltile
					}
					g.w = append(g.w, g.prism(p.poly, p.cav, h, side, tt(texSeats), tt(bottom)))
				}
			}
			for i, a := range aisles { // crowd
				for _, p := range parts(rq(ta+6, ta+42, seg, a[0], a[1]), seg, k) {
					if p.cav != 0 && h <= p.cav+16 {
						continue
					}
					g.crowd = append(g.crowd, g.prismT(p.poly, h, h+40, crowdSpec(k*3+seg+i, h+40, false), tt(texCrowdtop), tt(texNull)))
				}
			}
		}
		if seg != 2 { // aisle steps up to the fascia-high first tier, 8-unit risers
			for j := range 6 {
				fj := float64(j)
				g.w = append(g.w, g.prismT(rq(a0-48+8*fj, a0-40+8*fj, seg, 0.46, 0.54), 0, 8*(fj+1), tt(texStands), tt(texSeats), none))
			}
		}
	}
	for seg := range 8 { // concourse (side 2 keeps it solid; the tunnel and room end before it)
		if seg != 6 {
			g.w = append(g.w, g.prismT(rq(conc0, conc1, seg, 0, 1), 0, concZ, tt(texStadwall), tt(texConcrete), none))
		}
	}
}

// ------------------------------------------------------------------ upper deck + balcony + stairs

func (g *gen) upperDeck() {
	const st0, st1 = 0.10, 0.32 // stair run along the side
	holeT0 := st0 + (st1-st0)*6/30
	wall, conc := tt(texStadwall), tt(texConcrete)
	for seg := range 8 {
		if seg == 6 {
			continue
		}
		pars := [][2]float64{{0, 0.46}, {0.54, 1}}
		if seg%2 == 1 { // balcony slab (overhang part) with a stair hole
			g.w = append(g.w, g.prismT(rq(b0, conc1, seg, 0, holeT0), und, balZ, wall, conc, none),
				g.prismT(rq(b0, conc1, seg, st1+0.01, 1), und, balZ, wall, conc, none))
			for j := range 30 {
				t0 := st0 + (st1-st0)*float64(j)/30
				t1 := st0 + (st1-st0)*float64(j+1)/30
				g.w = append(g.w, g.prismT(rq(1976, conc1, seg, t0, t1), concZ, concZ+8*float64(j+1), wall, tt("STEP"), none))
			}
			pars = [][2]float64{{0, st0 - 0.01}, {st1 + 0.02, 0.46}, {0.54, 1}}
		} else {
			g.w = append(g.w, g.prismT(rq(b0, conc1, seg, 0, 1), und, balZ, wall, conc, none))
		}
		g.w = append(g.w, g.prismT(rq(conc1, b1, seg, 0, 1), 0, balZ, wall, conc, none))
		for _, p := range pars { // balcony parapet with an LED face
			g.w = append(g.w, g.prism(rq(b0, b0+6, seg, p[0], p[1]), balZ, balZ+32,
				facing(segn(seg), true, ribbonSpec(texRibbon2, 1, 1, (balZ+32)/1), texStadwall), tt(texCagepad), none))
		}
		for k := range unt { // upper tiers + crowd
			ta, tb := ua0+dep*float64(k), ua0+dep*float64(k+1)
			g.w = append(g.w, g.prismT(rq(ta, tb, seg, 0, 1), 0, uh(k), tt(texStands), tt(texSeats), none))
			for i, a := range aisles {
				g.crowd = append(g.crowd, g.prismT(rq(ta+6, ta+42, seg, a[0], a[1]), uh(k), uh(k)+40,
					crowdSpec(k*5+seg+i+1, uh(k)+40, false), tt(texCrowdtop), tt(texNull)))
			}
		}
	}
	sv := (suiteZ1 - suiteZ0) / 64
	for seg := range 8 {
		g.w = append(g.w, g.prismT(ringToSquare(utop, seg), 0, suiteZ0, wall, conc, none),
			g.prism(ringToSquare(utop, seg), suiteZ0, suiteZ1, facing(segn(seg), true, tt(texSuites).WithScale(2, sv).WithOV(suiteZ1/sv), texStadwall), none, none),
			g.prismT(ringToSquare(utop, seg), suiteZ1, skyTop, wall, none, none),
			g.prismT(rq(utop-10, utop, seg, 0, 1), suiteZ0-8, suiteZ0, tt(texRedstrip), none, none))
	}
}

// ------------------------------------------------------------------ south: walkout stage, LED wall, red locker room

func (g *gen) south() {
	g.w = append(g.w, g.prismT(rq(stA0, ledA, 6, 0, 1), 0, 24, tt(texStage), none, none))
	bl := ledA * t8
	walk := g.Fit(texWalkout, -bl, bl, 24, 424)
	for _, b := range [][4]float64{{-bl, -48, 24, 424}, {48, bl, 24, 424}, {-48, 48, 136, 424}} {
		g.w = append(g.w, g.box(b[0], -ledA-16, b[2], b[1], -ledA, b[3], texMetal, mapkit.Sides{N: walk}))
	}
	g.w = append(g.w, g.box(-bl, -ledA-16, 0, bl, -ledA, 24, texStage))
	// mass behind the LED wall, carved for the red locker room (x +-320, y -2032..-1732, z 24..184)
	const rx, ry0, ry1 = 320.0, -2032.0, -ledA - 16
	mass := ringToSquare(ledA+16, 6)
	conc := tt(texConcrete)
	g.w = append(g.w, g.prismT(clip2(mass, mapkit.V2{1, 0}, -rx), 0, balZ, tt(texStadwall), conc, none),
		g.prismT(clip2(mass, mapkit.V2{-1, 0}, -rx), 0, balZ, tt(texStadwall), conc, none),
		g.box(-rx, -shell, 0, rx, ry0, balZ, texStadwall, mapkit.Sides{Top: conc, N: tt(texLockerR)}),
		g.box(-rx, ry0, 0, rx, ry1, rz0, texStage),
		g.box(-rx, ry0, rz1, rx, ry1, balZ, texStadwall, mapkit.Sides{Top: conc, Bottom: tt(texCeiltile)}))
	for _, p := range [][2]float64{{0, 0.46}, {0.54, 1}} {
		g.w = append(g.w, g.prism(rq(ledA+16, ledA+22, 6, p[0], p[1]), balZ, balZ+32,
			facing(mapkit.V2{0, 1}, false, ribbonSpec(texRibbon1, 1, 1, balZ+32), texStadwall), tt(texCagepad), none))
	}
	// red locker room dressing
	for x := -rx + 8; x < rx-40; x += 48 {
		if math.Abs(x+20) < 110 {
			continue
		}
		g.w = append(g.w, g.box(x, ry0, rz0, x+40, ry0+24, rz0+96, texLockerR))
	}
	g.w = append(g.w, g.box(-rx, ry0+60, rz0, -rx+40, ry1-40, rz0+96, texLockerR),
		g.box(rx-40, ry0+60, rz0, rx, ry1-40, rz0+96, texLockerR),
		g.box(-200, -1930, rz0, 200, -1910, rz0+18, texCouch),
		g.sign(texCornerR, -112, ry0+2, rz0+28, 104, ry0+6, rz0+136, "n", texMetal))
	g.light(0, -1880, 160, [3]int{255, 80, 70}, 160)
	g.light(-200, -1800, 160, [3]int{255, 230, 210}, 90)
	g.light(200, -1800, 160, [3]int{255, 230, 210}, 90)
	// runway + cage stairs
	run := func(name string) mapkit.Tex {
		return tt(name).WithScale(112.0/128, 112.0/128).WithOU(56 / (112.0 / 128))
	}
	g.w = append(g.w, g.box(-56, -stA0, 0, 56, -388, 16, texStage, mapkit.Sides{Top: run("WALKWAY_R")}),
		g.box(-56, 388, 0, 56, a0, 16, texStage, mapkit.Sides{Top: run("WALKWAY_B")})) // blue walkway
	for _, x0 := range []float64{-60, 52} { // red gate light bars
		g.w = append(g.w, g.box(x0, -ledA, 24, x0+8, -ledA+8, 144, texRedstrip))
	}
	g.w = append(g.w, g.box(-60, -ledA, 136, 60, -ledA+8, 144, texRedstrip))
	for _, sgn := range []float64{1, -1} {
		for _, st := range [][3]float64{{12, 388, 364}, {24, 364, 340}} {
			ya, yb := st[1]*sgn, st[2]*sgn
			g.w = append(g.w, g.box(-40, min(ya, yb), 0, 40, max(ya, yb), st[0], texStage))
		}
	}
}

// ------------------------------------------------------------------ north: blue locker room under the stands

func (g *gen) north() {
	brY0, brY1 := a0+dep*roomK0, a0+dep*(roomK1-1)
	for x := -roomX + 8; x < roomX-40; x += 48 {
		if math.Abs(x+20) < 110 {
			continue
		}
		g.w = append(g.w, g.box(x, brY1-24, 0, x+40, brY1, 96, texLockerB))
	}
	g.w = append(g.w, g.box(-roomX, brY0+40, 0, -roomX+24, brY1-60, 96, texLockerB),
		g.box(roomX-24, brY0+40, 0, roomX, brY1-60, 96, texLockerB),
		g.box(-160, 1640, 0, 160, 1660, 18, texCouch),
		g.sign(texCornerB, -112, brY1-6, 28, 104, brY1-2, 136, "s", texMetal))
	for _, x0 := range []float64{-tunX - 8, tunX} {
		g.w = append(g.w, g.box(x0, a0-8, 0, x0+8, a0, tunTop, texBluestrip))
	}
	g.w = append(g.w, g.sign(texCornerB, -96, a0-12, tunTop+24, 96, a0-8, tunTop+120, "s", texCagepad))
	for _, x := range []float64{-90, 82} {
		g.w = append(g.w, g.box(x, a0-12, 0, x+8, a0-8, tunTop+24, texMetal))
	}
	g.light(0, 1600, 150, [3]int{80, 120, 255}, 170)
	g.light(-180, 1720, 150, [3]int{220, 230, 255}, 80)
	g.light(180, 1720, 150, [3]int{220, 230, 255}, 80)
	for _, y := range []float64{1000, 1200, 1400} {
		g.w = append(g.w, g.box(-6, y-6, tunTop-4, 6, y+6, tunTop, texMetal, mapkit.Sides{Bottom: tt(texLamp)}))
		g.light(0, y, 100, [3]int{120, 150, 255}, 80)
	}
}

// ------------------------------------------------------------------ the octagon

// apronSide fits the apron banner to exactly one octagon side, so it never wraps mid-word (it used to read
// "Batt|tleAxe").
func (g *gen) apronSide(n2 mapkit.V2, seg int) mapkit.Tex {
	u, _ := mapkit.Axes(mapkit.V3{n2[0], n2[1], 0})
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, t := range []float64{0, 1} {
		p := onoct(pa, seg, t)
		v := mapkit.V3{p[0], p[1], 0}.Dot(u)
		lo, hi = min(lo, v), max(hi, v)
	}
	return g.Fit("APRONSIDE", lo, hi, 0, pz)
}

func (g *gen) octagon() {
	g.w = append(g.w, g.prismT(octv(ca), 0, pz, tt(texApron), tt("CANVAS").WithScale(2.4, 2.4).WithOU(128).WithOV(128), none))
	for seg := range 8 {
		g.w = append(g.w, g.prismT(rq(ca, pa-6, seg, 0, 1), 0, pz, tt(texApron), none, none),
			g.prism(rq(pa-6, pa, seg, 0, 1), 0, pz, func(n2 mapkit.V2) mapkit.Tex {
				if n2.Dot(segn(seg)) > 0.99 {
					return g.apronSide(n2, seg)
				}
				return tt(texApron)
			}, tt(texRedstrip), none))
	}
	var fence, rail []string
	for seg := range 8 {
		spans := [][2]float64{{0, 1}}
		if seg == 2 || seg == 6 {
			spans = [][2]float64{{0, 0.36}, {0.64, 1}}
		}
		for _, s := range spans {
			fence = append(fence, g.prismT(rq(ca, fa, seg, s[0], s[1]), pz, pz+80, tt("{CAGEFENCE").WithScale(0.5, 0.5), none, none))
			rail = append(rail, g.prismT(rq(ca-2, fa+2, seg, s[0], s[1]), pz+80, pz+86, tt(texCagepad), none, none))
		}
	}
	g.bent("func_wall", fence, "rendermode", 4, "renderamt", 255)
	g.w = append(g.w, rail...)
	const ph = 104.0
	for _, p := range octv(fa - 2) {
		x, y := p[0], p[1]
		var post mapkit.Sides
		for _, f := range []string{"n", "s", "e", "w"} {
			post = mapkit.SetSide(post, f, g.FitBox("CAGEPOST", f, x-12, y-12, x+12, y+12, pz, pz+ph))
		}
		g.w = append(g.w, g.box(x-12, y-12, pz, x+12, y+12, pz+ph, texCagepad, post))
	}
}

// ------------------------------------------------------------------ lighting rig, beams, jumbotron, towers

func (g *gen) rig() {
	for _, b := range [][4]float64{{-336, -336, 336, -320}, {-336, 320, 336, 336}, {-336, -320, -320, 320}, {320, -320, 336, 320}} {
		g.w = append(g.w, g.box(b[0], b[1], tz, b[2], b[3], tz+16, texTruss))
	}
	var beams []string
	for _, x := range []float64{-328, 0, 328} {
		for _, y := range []float64{-328, 0, 328} {
			if x == 0 && y == 0 {
				continue
			}
			g.w = append(g.w, g.box(x-12, y-12, tz-16, x+12, y+12, tz, texMetal, mapkit.Sides{Bottom: tt(texLamp)}))
			g.light(x*0.8, y*0.8, tz-40, [3]int{255, 248, 235}, 420)
			tx, ty := x*0.35, y*0.35
			var pts []mapkit.V3
			for _, dx := range []float64{-10, 10} {
				for _, dy := range []float64{-10, 10} {
					pts = append(pts, mapkit.V3{x + dx, y + dy, tz - 18})
				}
			}
			for _, dx := range []float64{-46, 46} {
				for _, dy := range []float64{-46, 46} {
					pts = append(pts, mapkit.V3{tx + dx, ty + dy, pz + 2})
				}
			}
			beams = append(beams, g.Hull(pts, texLightbeam, ""))
		}
	}
	for _, part := range mapkit.Chunks(beams, 256) {
		g.bent("func_illusionary", part, "rendermode", 5, "renderamt", 28)
	}
	g.light(0, 0, tz-60, [3]int{255, 255, 255}, 700)
	g.w = append(g.w, g.box(-128, -128, jz0, 128, 128, jz1, texMetal, mapkit.Sides{N: g.Fit(texJumbo2, -128, 128, jz0, jz1),
		S: g.Fit(texJumbo1, -128, 128, jz0, jz1), E: g.Fit(texJumbo1, -128, 128, jz0, jz1), W: g.Fit(texJumbo2, -128, 128, jz0, jz1)}))
	g.w = append(g.w, g.box(-136, -136, jz0-24, 136, 136, jz0, texMetal, mapkit.Sides{
		Bottom: g.Fit(texAxelogo, -136, 136, -136, 136).WithSV(272.0 / 64).WithOV(136 / (272.0 / 64)),
		N:      ribbonSpec(texRibbon1, 1.0625, 0.75, jz0/0.75), S: ribbonSpec(texRibbon2, 1.0625, 0.75, jz0/0.75),
		E: ribbonSpec(texRibbon3, 1.0625, 0.75, jz0/0.75), W: ribbonSpec(texRibbon1, 1.0625, 0.75, jz0/0.75)}))
	corners := []float64{-1, 1}
	for _, sx := range corners {
		for _, sy := range corners {
			g.w = append(g.w, g.Beam(mapkit.V3{sx * 328, sy * 328, tz + 16}, mapkit.V3{sx * 120, sy * 120, jz0 - 24}, 4, texMetal))
		}
	}
	const tw = 1900.0
	for _, sx := range corners {
		for _, sy := range corners {
			bx, by := sx*tw-60*sx, sy*tw-60*sy
			bank := mapkit.Sides{Bottom: tt(texLightbank)}
			bank = mapkit.SetSide(bank, map[bool]string{true: "w", false: "e"}[sx > 0], tt(texLightbank))
			bank = mapkit.SetSide(bank, map[bool]string{true: "s", false: "n"}[sy > 0], tt(texLightbank))
			g.w = append(g.w, g.box(bx-50, by-50, 1300, bx+50, by+50, 1320, texMetal, bank),
				g.box(bx-6, by-6, 1320, bx+6, by+6, skyTop, texMetal),
				g.Beam(mapkit.V3{bx, by, 1320}, mapkit.V3{sx * 120, sy * 120, jz1}, 6, texMetal),
				g.Beam(mapkit.V3{sx * 120, sy * 120, jz1}, mapkit.V3{sx * 120, sy * 120, skyTop}, 3, texMetal))
		}
	}
	const rz = skyTop - 60 // roof rigging
	for k := -4.0; k <= 4; k++ {
		g.w = append(g.w, g.box(k*600-10, -shell+200, rz, k*600+10, shell-200, rz+24, texTruss),
			g.box(-shell+200, k*600-10, rz-24, shell-200, k*600+10, rz, texTruss))
	}
	for x := -2100.0; x <= 2100; x += 600 {
		for y := -2100.0; y <= 2100; y += 600 {
			if math.Abs(x) < 400 && math.Abs(y) < 400 {
				continue
			}
			g.w = append(g.w, g.box(x+280-80, y+300-10, skyTop-8, x+280+80, y+300+10, skyTop, texMetal, mapkit.Sides{Bottom: tt(texLamp)}))
			// Every roof lamp's light sits at one spot near the north-east tower (1700, 1700): the original script
			// placed it with the tower loop's last corner. Kept as is, so the port doesn't change the lighting.
			g.light(tw-200, tw-200, 1240, [3]int{255, 250, 235}, 4200)
		}
	}
}

// ------------------------------------------------------------------ floor: press row, broadcast desk, camera crane, floor seats

func (g *gen) floor() {
	tv := tt(texTvblack)
	for _, seg := range []int{0, 3, 4, 5, 7} {
		g.w = append(g.w, g.prismT(rq(430, 460, seg, 0.12, 0.88), 0, 30, tt(texPress), tt(texTable), none))
		for _, t := range []float64{0.25, 0.5, 0.75} {
			g.w = append(g.w, g.prism(rq(446, 450, seg, t-0.06, t+0.06), 30, 46,
				facing(segn(seg), true, tt(texBroadcast).WithScale(0.6, 0.34).WithOV(46/0.34), texTvblack), tv, none))
		}
	}
	g.w = append(g.w, g.prismT(rq(430, 490, 1, 0.15, 0.85), 0, 32, tt(texPress), tt(texTable), none))
	for _, t := range []float64{0.3, 0.5, 0.7} {
		g.w = append(g.w, g.prism(rq(470, 474, 1, t-0.07, t+0.07), 32, 56,
			facing(segn(1), true, tt(texBroadcast).WithScale(0.7, 0.5).WithOV(56/0.5), texTvblack), tv, none))
	}
	const cx, cy = -370.0, 370.0 // camera crane
	g.w = append(g.w, g.box(cx-24, cy-24, 0, cx+24, cy+24, 64, texCamera),
		g.Beam(mapkit.V3{cx - 60, cy + 60, 70}, mapkit.V3{cx + 150, cy - 150, 300}, 10, texCamera),
		g.box(cx+140, cy-170, 270, cx+175, cy-135, 300, texCamera),
		g.box(cx-80, cy+50, 50, cx-50, cy+80, 90, texMetal))
	for seg := range 8 {
		if seg == 6 {
			continue
		}
		for r := range 3 {
			ra := 560 + 64*float64(r)
			for i, a := range aisles {
				cp := rq(ra, ra+40, seg, a[0], a[1])
				ps := [][]mapkit.V2{cp}
				if seg == 2 {
					ps = slices.DeleteFunc([][]mapkit.V2{clip2(cp, mapkit.V2{1, 0}, -80), clip2(cp, mapkit.V2{-1, 0}, -80)},
						func(p []mapkit.V2) bool { return p == nil })
				}
				for _, p := range ps {
					g.crowd = append(g.crowd, g.prismT(p, 0, 40, crowdSpec(r+seg*3+i, 40, true), tt(texCrowdtop), tt(texNull)))
				}
			}
		}
	}
}

// ------------------------------------------------------------------ pyro + general lights

func (g *gen) pyroAndLights() {
	fire := func(x, y, z, sc float64) {
		g.point("env_sprite", x, y, z, "model", "sprites/"+name+"_fire.spr", "rendermode", 5, "renderamt", 255,
			"rendercolor", "255 255 255", "framerate", 16, "scale", sc, "spawnflags", 1)
	}
	for _, x := range []float64{-560, -300, 300, 560} {
		fire(x, -1330, 24, 2.4)
	}
	for _, x := range []float64{-90, 90} {
		fire(x, -1290, 16, 1.4)
		fire(x, -900, 0, 1.2)
	}
	for _, x := range []float64{-100, 100} {
		fire(x, 910, 0, 1.6)
	}
	g.light(0, -1450, 300, [3]int{255, 60, 40}, 600)
	g.light(-450, -1450, 260, [3]int{255, 140, 40}, 400)
	g.light(450, -1450, 260, [3]int{255, 140, 40}, 400)
	g.light(0, 960, 180, [3]int{80, 120, 255}, 350)
	for seg := range 8 {
		p := onoct(1400, seg, 0.5)
		g.light(p[0], p[1], 1000, [3]int{240, 240, 255}, 1100)
		p = onoct(2350, seg, 0.5)
		g.light(p[0], p[1], 1150, [3]int{240, 240, 255}, 900)
		if seg != 6 {
			p = onoct(1950, seg, 0.7)
			g.light(p[0], p[1], und-30, [3]int{255, 230, 200}, 160)
		}
		p = onoct(650, seg, 0.5)
		g.light(p[0], p[1], 420, [3]int{255, 245, 230}, 300)
	}
}

// ------------------------------------------------------------------ weapons + spawns (spawns ONLY in the locker rooms)

func (g *gen) weaponsAndSpawns() {
	weapons := []string{"weapon_shotgun", "weapon_9mmAR", "weapon_357", "weapon_crossbow", "weapon_rpg", "weapon_gauss",
		"weapon_egon", "weapon_hornetgun", "weapon_handgrenade", "weapon_satchel", "weapon_snark", "weapon_tripmine"}
	ammo := []string{"ammo_buckshot", "ammo_9mmAR", "ammo_357", "ammo_crossbow", "ammo_rpgclip", "ammo_gaussclip",
		"ammo_ARgrenades", "item_battery", "item_healthkit"}
	wi := 0
	for _, ring := range []struct {
		r float64
		n int
	}{{120, 8}, {230, 12}} {
		for i := range ring.n {
			ang := 2 * math.Pi * float64(i) / float64(ring.n)
			cls := ammo[i%len(ammo)]
			if ring.r == 230 {
				ang += 0.2
				cls = weapons[wi%len(weapons)]
			}
			g.items = append(g.items, item{cls, ring.r * math.Cos(ang), ring.r * math.Sin(ang), pz + 10})
			wi++
		}
	}
	for seg := range 8 {
		if seg != 6 {
			w, a := "weapon_357", "ammo_357"
			if seg%2 == 1 {
				w, a = "weapon_crossbow", "ammo_crossbow"
			}
			p := onoct(1950, seg, 0.85)
			g.items = append(g.items, item{w, p[0], p[1], concZ + 10})
			p = onoct(2040, seg, 0.6)
			g.items = append(g.items, item{a, p[0], p[1], balZ + 10})
		}
		if p := onoct(800, seg, 0.5); seg != 2 && seg != 6 {
			g.items = append(g.items, item{ammo[seg%len(ammo)], p[0], p[1], 10})
		}
	}
	for _, x := range []float64{-200, 200} {
		g.items = append(g.items, item{"item_healthkit", x, -1500, 34})
	}
	for _, x := range []float64{-40, 40} {
		g.items = append(g.items, item{"item_battery", x, 1100, 10}) // in the blue tunnel
	}
	for _, s := range [][3]float64{{-200, -1830, rz0}, {-60, -1830, rz0}, {80, -1830, rz0}, {200, -1830, rz0}, {-120, -1870, rz0},
		{120, -1870, rz0}, {-180, 1560, 0}, {-60, 1560, 0}, {60, 1560, 0}, {180, 1560, 0}, {-120, 1720, 0}, {120, 1720, 0}} {
		face := 270
		if s[1] < 0 {
			face = 90
		}
		g.spawns++
		g.point("info_player_deathmatch", s[0], s[1], s[2]+37, "angles", fmt.Sprintf("0 %d 0", face))
	}
	for _, it := range g.items {
		g.point(it.cls, mapkit.Round1(it.x), mapkit.Round1(it.y), it.z, "angles", fmt.Sprintf("0 %d 0", int(math.Abs(it.x*3+it.y))%8*45))
	}
	g.point("ambient_generic", 0, 0, 400, "message", "ambience/crowd1.wav", "health", 10, "spawnflags", 1)
}
