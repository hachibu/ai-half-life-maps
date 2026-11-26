package main

import (
	"math"

	"halflifemaps/maps/prairieloop/layout"
	"halflifemaps/tools/mapkit"
)

// tunnelLamps are the ring tunnel's ceiling lamps.
var tunnelLamps = [][2]float64{{-1024, 576}, {-600, 576}, {0, 576}, {600, 576}, {1024, 576}, {-1024, -576}, {-600, -576},
	{0, -576}, {600, -576}, {1024, -576}, {-1024, -300}, {-1024, 300}, {1024, -300}, {1024, 300}, {-1024, 0}, {1024, 0}}

// pods are the LAN hall's eight four-seat pods.
var pods = [][2]float64{{-480, 144}, {-160, 144}, {160, 144}, {480, 144}, {-480, -112}, {-160, -112}, {160, -112}, {480, -112}}

// ------------------------------------------------------------------ outdoors: rink, snowman, trees, lamps, cars, fences

func spherePts(c mapkit.V3, r float64) []mapkit.V3 {
	const n, rings = 10, 5
	var pts []mapkit.V3
	for i := 1; i < rings; i++ {
		ph := math.Pi * float64(i) / rings
		for j := range n {
			th := 2 * math.Pi * (float64(j) + 0.5*float64(i%2)) / n
			pts = append(pts, mapkit.V3{c[0] + r*math.Sin(ph)*math.Cos(th), c[1] + r*math.Sin(ph)*math.Sin(th), c[2] + r*math.Cos(ph)})
		}
	}
	return append(pts, mapkit.V3{c[0], c[1], c[2] + r}, mapkit.V3{c[0], c[1], c[2] - r})
}

func (g *gen) cyl(cx, cy, r, z0, z1 float64, tex string) string {
	const n = 10
	var pts []mapkit.V3
	for k := range n {
		a := 2 * math.Pi * float64(k) / n
		for _, z := range []float64{z0, z1} {
			pts = append(pts, mapkit.V3{cx + r*math.Cos(a), cy + r*math.Sin(a), z})
		}
	}
	return g.Hull(pts, tex, "")
}

func (g *gen) tree(x, y, s float64) {
	g.w = append(g.w, g.box(x-8, y-8, 0, x+8, y+8, 40*s, texBark))
	for _, t := range [][3]float64{{36, 70, 190}, {120, 52, 260}, {190, 34, 320}} {
		var pts []mapkit.V3
		for k := range 8 {
			a := 2 * math.Pi * float64(k) / 8
			pts = append(pts, mapkit.V3{x + t[1]*s*math.Cos(a), y + t[1]*s*math.Sin(a), t[0] * s})
		}
		g.w = append(g.w, g.Hull(append(pts, mapkit.V3{x, y, t[2] * s}), "PINE", ""))
	}
}

func (g *gen) car(x, y float64, paint string, alongX bool) {
	const l, wd = 200.0, 84.0
	snowTop := mapkit.Sides{Top: mapkit.T(texSnow)}
	if alongX {
		g.w = append(g.w, g.box(x-l/2, y-wd/2, 14, x+l/2, y+wd/2, 46, paint, snowTop),
			g.box(x-50, y-wd/2+6, 46, x+50, y+wd/2-6, 82, texCarglass, snowTop))
	} else {
		g.w = append(g.w, g.box(x-wd/2, y-l/2, 14, x+wd/2, y+l/2, 46, paint, snowTop),
			g.box(x-wd/2+6, y-50, 46, x+wd/2-6, y+50, 82, texCarglass, snowTop))
	}
	for _, dx := range []float64{-62, 62} {
		for _, dy := range []float64{-wd / 2, wd/2 - 12} {
			if alongX {
				g.w = append(g.w, g.box(x+dx-14, y+dy-2, 0, x+dx+14, y+dy+14, 26, texTire))
			} else {
				g.w = append(g.w, g.box(x+dy-2, y+dx-14, 0, x+dy+14, y+dx+14, 26, texTire))
			}
		}
	}
}

func (g *gen) outdoors() {
	g.w = append(g.w, g.box(-384, -192, 0, 384, 192, 2, texIce))
	for _, b := range [][4]float64{{-392, -200, -32, -192}, {32, -200, 392, -192}, {-392, 192, -32, 200}, {32, 192, 392, 200},
		{-392, -192, -384, 192}, {384, -192, 392, 192}} {
		g.w = append(g.w, g.box(b[0], b[1], 0, b[2], b[3], 28, texRinkboard))
	}
	g.snowman(232, 768)
	for _, t := range [][3]float64{{-700, -300, 1}, {-640, 320, 1.1}, {680, 300, 0.9}, {720, -280, 1}, {-560, 0, 0.8}, {560, 40, 0.85},
		{-1700, 1350, 1.2}, {1700, 1350, 1.1}, {-1700, -1350, 1.1}, {1750, -1300, 1.2}, {-2000, 520, 1.3},
		{-2400, -560, 1.2}, {-2700, 600, 1.4}, {-1600, -600, 1}} {
		g.tree(t[0], t[1], t[2])
	}
	for _, p := range [][2]float64{{-768, 720}, {0, 720}, {768, 720}, {-768, -720}, {0, -720}, {768, -720}, {1168, 360}, {1168, -360},
		{-1600, 150}, {-2200, -150}, {-2800, 150}, {-1168, 500}} {
		x, y := p[0], p[1]
		g.w = append(g.w, g.box(x-4, y-4, 0, x+4, y+4, 196, texMetal))
		g.decor = append(g.decor, g.box(x-12, y-12, 188, x+12, y+12, 198, texMetal, mapkit.Sides{Bottom: mapkit.T(texLamp)}))
		g.light(x, y, 176, [3]int{255, 200, 130}, 300)
	}
	for _, p := range [][2]float64{{-420, -260}, {420, 260}, {-420, 260}, {420, -260}} {
		g.light(p[0], p[1], 160, [3]int{220, 230, 255}, 160)
	}
	for _, c := range []struct {
		x, y   float64
		paint  string
		alongX bool
	}{{-480, 650, texCarred, true}, {560, -650, texCarblue, true}, {-1080, 200, "CARSILVER", false},
		{-2000, -80, texCarblue, true}, {1080, -150, texCarred, false}} {
		g.car(c.x, c.y, c.paint, c.alongX)
	}
	g.w = append(g.w, g.box(xw+32, yn-40, 0, xe-32, yn-32, 96, texWoodfence), g.box(xw+32, ys+32, 0, xe-32, ys+40, 96, texWoodfence),
		g.box(xe-40, ys+40, 0, xe-32, yn-40, 96, texWoodfence),
		g.box(-3072, 400, 0, -1840, 408, 72, texWoodfence), g.box(-3072, -408, 0, -1840, -400, 72, texWoodfence),
		g.box(-3040, -128, 0, -3024, 128, 40, texRinkboard),
		g.box(-3020, -6, 0, -3014, 6, 70, texMetal), g.sign("DEADEND", -3014, -40, 70, -3010, 40, 110, "e"),
		g.box(-1180, 160, 0, -1174, 166, 110, texMetal), g.sign("STREETSIGN", -1220, 166, 110, -1100, 170, 140, "n"))
}

func (g *gen) snowman(sx, sy float64) {
	for _, b := range [][2]float64{{26, 28}, {72, 21}, {108, 15}} {
		g.w = append(g.w, g.Hull(spherePts(mapkit.V3{sx, sy, b[0]}, b[1]), "SNOWMAN", ""))
	}
	g.w = append(g.w, g.cyl(sx, sy, 19, 121, 124, texCoal), g.cyl(sx, sy, 11, 124, 150, texCoal))
	g.decor = append(g.decor, g.cyl(sx, sy, 12, 135, 139, texScarf), g.cyl(sx, sy, 17, 86, 92, texScarf),
		g.Beam(mapkit.V3{sx, sy - 13, 108}, mapkit.V3{sx, sy - 30, 106}, 4, "CARROT"))
	for _, dx := range []float64{-5, 5} {
		g.decor = append(g.decor, g.box(sx+dx-2, sy-15, 113, sx+dx+2, sy-12, 117, texCoal))
	}
	for _, dz := range []float64{60, 75, 90} {
		g.decor = append(g.decor, g.box(sx-2, sy-22, dz-2, sx+2, sy-19, dz+2, texCoal))
	}
	g.decor = append(g.decor, g.Beam(mapkit.V3{sx - 18, sy, 76}, mapkit.V3{sx - 50, sy - 6, 100}, 3, texBark),
		g.Beam(mapkit.V3{sx + 18, sy, 76}, mapkit.V3{sx + 48, sy + 4, 96}, 3, texBark))
}

// ------------------------------------------------------------------ ring tunnel dressing: faction colours

type factionColours struct {
	s1, s2 string
	c1     [3]int
}

func tunnelColour(x float64) factionColours {
	switch {
	case x < -500:
		return factionColours{texStripPink, texStripCoral, [3]int{255, 80, 170}}
	case x > 500:
		return factionColours{texStripCyan, texStripViolet, [3]int{60, 220, 255}}
	}
	return factionColours{texStripWhite, texStripWhite, [3]int{220, 225, 255}}
}

func (g *gen) tunnelDressing() {
	for xa := -1216.0; xa < 1216; xa += 152 {
		xb := xa + 152
		tc := tunnelColour((xa + xb) / 2)
		for _, yy := range [][2]float64{{512, 640}, {-640, -512}} {
			y0, y1 := yy[0], yy[1]
			g.decor = append(g.decor, g.box(xa, y0, bf+6, xb, y0+2, bf+10, tc.s1), g.box(xa, y1-2, bf+6, xb, y1, bf+10, tc.s2),
				g.box(xa, y0, bc-10, xb, y0+2, bc-6, tc.s2), g.box(xa, y1-2, bc-10, xb, y1, bc-6, tc.s1))
		}
	}
	for ya := -512.0; ya < 512; ya += 128 {
		for _, s := range []struct {
			x0, x1 float64
			c0, c1 string
		}{{-1088, -960, texStripPink, texStripCoral}, {960, 1088, texStripCyan, texStripViolet}} {
			g.decor = append(g.decor, g.box(s.x0, ya, bf+6, s.x0+2, ya+128, bf+10, s.c0), g.box(s.x1-2, ya, bf+6, s.x1, ya+128, bf+10, s.c1),
				g.box(s.x0, ya, bc-10, s.x0+2, ya+128, bc-6, s.c1), g.box(s.x1-2, ya, bc-10, s.x1, ya+128, bc-6, s.c0))
		}
	}
	for _, p := range tunnelLamps {
		x, y := p[0], p[1]
		g.decor = append(g.decor, g.box(x-6, y-6, bc-4, x+6, y+6, bc, texMetal, mapkit.Sides{Bottom: mapkit.T(texLamp)}))
		g.light(x, y, bc-24, tunnelColour(x).c1, 110)
	}
	g.decor = append(g.decor, g.sign(texEmbAxe, -1088, 120, bf+30, -1086, 184, bf+94, "e"),
		g.sign(texEmbAxe, -1088, -184, bf+30, -1086, -120, bf+94, "e"),
		g.sign(texEmbMoon, 1086, 120, bf+30, 1088, 184, bf+94, "w"),
		g.sign(texEmbMoon, 1086, -184, bf+30, 1088, -120, bf+94, "w"),
		g.sign(texTsAxe, -700, 638, bf+70, -580, 640, bf+100, "s"),
		g.sign(texTsMoon, 580, 638, bf+70, 700, 640, bf+100, "s"),
		g.sign(texTsAxe, -700, -640, bf+70, -580, -638, bf+100, "n"),
		g.sign(texTsMoon, 580, -640, bf+70, 700, -638, bf+100, "n"))
	g.light(-896, 0, bc-24, [3]int{255, 80, 170}, 120)
	g.light(896, 0, bc-24, [3]int{60, 220, 255}, 120)
	g.light(512, 740, -100, [3]int{180, 150, 255}, 70)
	g.light(512, 500, -230, [3]int{255, 90, 200}, 80)
}

// ------------------------------------------------------------------ the JustinCon hall

func (g *gen) hall() {
	g.w = append(g.w, g.wallY(hallY0, hallY1, hallX0, hallX0+lin, hf, hc, texHallwall, [][4]float64{{-48, 48, bf, bc}})...) // BattleAxe door
	g.w = append(g.w, g.wallY(hallY0, hallY1, hallX1-lin, hallX1, hf, hc, texHallwall, [][4]float64{{-48, 48, bf, bc}})...) // Crescent door
	g.w = append(g.w, g.wallX(ix0, ix1, hallY1-lin, hallY1, hf, hc, texHallwall, [][4]float64{{px0, px1, -320, -192}})...)
	g.w = append(g.w, g.wallX(ix0, ix1, hallY0, hallY0+lin, hf, hc, texHallwall, [][4]float64{{sdoorX0, sdoorX1, hf, -352}})...)
	stage, carpet := mapkit.Sides{Top: mapkit.T("STAGEFLOOR")}, mapkit.Sides{Top: mapkit.T(texArcadecarpet)}
	// BattleAxe entrance: landing + stairs down (north) along the west wall
	g.w = append(g.w, g.box(ix0, -48, hf, ix0+64, 48, bf, texHallwall, stage))
	for j := 1.0; j <= 34; j++ { // 8-unit risers
		g.w = append(g.w, g.box(ix0, 48+8*(j-1), hf, ix0+64, 48+8*j, bf-8*j, texHallwall, carpet))
	}
	// Crescent entrance: landing + stairs down (south) along the east wall
	g.w = append(g.w, g.box(ix1-64, -48, hf, ix1, 48, bf, texHallwall, stage))
	for j := 1.0; j <= 34; j++ {
		g.w = append(g.w, g.box(ix1-64, -48-8*j, hf, ix1, -48-8*(j-1), bf-8*j, texHallwall, carpet))
	}
	// blue-house reveal: mezzanine + stairs down (west) along the north wall
	g.w = append(g.w, g.box(px0-32, 264, hf, px1+32, iy1, -320, texHallwall, stage),
		g.box(px0-32, 260, -320, px1+32, 264, -288, texHallwall))
	g.decor = append(g.decor, g.box(px0-32, 259, -288, px1+32, 265, -284, texStripPink))
	for j := 1.0; j <= 16; j++ { // 8-unit risers
		g.w = append(g.w, g.box(px0-32-12*j, 296, hf, px0-32-12*(j-1), iy1, -320-8*j, texHallwall, carpet))
	}
	// stage, backdrop, match board, neon sign
	g.w = append(g.w, g.box(-320, iy0, hf, 320, -280, hf+32, texHallwall, stage))
	for _, x := range [][2]float64{{-140, 140}, {200, 300}} { // 8-unit steps up onto the stage, centre and teleport pad
		for i, z := range []float64{24, 16, 8} {
			fi := float64(i)
			g.w = append(g.w, g.box(x[0], -280+12*fi, hf, x[1], -268+12*fi, hf+z, texHallwall, stage))
		}
	}
	g.decor = append(g.decor, g.sign(texStageback, -232, iy0, hf+32, 232, iy0+2, -200, "n"),
		g.sign(texMatchboard, -168, iy0+2, -352, 168, iy0+6, -268, "n"),
		g.sign(texNeonJc, -240, iy0, -192, 240, iy0+6, -72, "n"),
		// faction banners facing each other
		g.sign("BANNER_AXE", ix0, -300, -400, ix0+3, -140, -80, "e"),
		g.sign("BANNER_MOON", ix1-3, 140, -400, ix1, 300, -80, "w"),
		g.box(ix0, -48, bc, ix0+3, 48, bc+8, texStripPink),
		g.box(ix1-3, -48, bc, ix1, 48, bc+8, texStripCyan))
	// glowing wall strips
	for _, z := range []float64{hf + 14, -150} {
		w, e := texStripCoral, texStripViolet
		if z < -200 {
			w, e = texStripPink, texStripCyan
		}
		g.decor = append(g.decor, g.box(ix0, iy0, z, ix0+2, iy1, z+4, w), g.box(ix1-2, iy0, z, ix1, iy1, z+4, e),
			g.box(ix0, iy1-2, z, 0, iy1, z+4, texStripCoral), g.box(0, iy1-2, z, ix1, iy1, z+4, texStripViolet),
			g.box(ix0, iy0, z, -320, iy0+2, z+4, texStripPink), g.box(320, iy0, z, ix1, iy0+2, z+4, texStripCyan))
	}
	for pi, p := range pods {
		g.pod(pi, p[0], p[1])
	}
	if g.handleSlots[0]+g.handleSlots[1] != 12 {
		panic("the LAN pods should show all 12 player handles")
	}
	g.light(0, -320, -150, [3]int{255, 120, 220}, 260)
	g.light(0, -200, -380, [3]int{120, 200, 255}, 160)
	g.light(0, 20, -120, [3]int{230, 220, 255}, 200)
	g.light(-700, -250, -260, [3]int{255, 110, 90}, 160)
	g.light(700, 250, -260, [3]int{150, 90, 255}, 160)
	g.light(px0+48, 300, -250, [3]int{255, 255, 255}, 90)
	for _, p := range [][2]float64{{-600, 150}, {-250, 0}, {250, 0}, {600, -150}, {0, 250}, {0, -250}} {
		g.point("ambient_generic", p[0], p[1], -400, "message", "justincon/fans_loop.wav", "health", 7, "spawnflags", 2, "pitch", 100, "pitchstart", 100)
	}
}

// pod is one LAN pod: a desk with four CRT workstations (some showing player handles), keyboards and chairs.
func (g *gen) pod(pi int, px, py float64) {
	side, pool := 1, handles[1:]
	if px < 0 {
		side, pool = 0, handles
	}
	g.w = append(g.w, g.box(px-56, py-56, hf, px+56, py+56, tableH, texDesk))
	fill := []string{texCrtGame, texCrtSaver, texCrtTerm}
	for k, f := range []string{"n", "e", "s", "w"} {
		d := mapkit.FacingNormal(f)
		dx, dy := d[0], d[1]
		cx, cy := px+dx*20, py+dy*20 // CRT body centred 20 units out from the pod centre, screen on its outer face
		bx0, bx1, by0, by1 := cx-15, cx+15, cy-18, cy+18
		if dx == 0 {
			bx0, bx1, by0, by1 = cx-18, cx+18, cy-15, cy+15
		}
		g.w = append(g.w, g.box(bx0, by0, tableH, bx1, by1, tableH+32, texCrt))
		tex := fill[(pi+k)%3]
		if order := (pi%4)*4 + k; order%3 == 0 && g.handleSlots[side] < 6 {
			tex = handleTex(pool[2*g.handleSlots[side]])
			g.handleSlots[side]++
		}
		// thin screen plate one unit in front of the body
		switch f {
		case "n":
			g.decor = append(g.decor, g.Sign(tex, bx0+3, by1, tableH+4, bx1-3, by1+1, tableH+28, "n", texCrt))
		case "s":
			g.decor = append(g.decor, g.Sign(tex, bx0+3, by0-1, tableH+4, bx1-3, by0, tableH+28, "s", texCrt))
		case "e":
			g.decor = append(g.decor, g.Sign(tex, bx1, by0+3, tableH+4, bx1+1, by1-3, tableH+28, "e", texCrt))
		default:
			g.decor = append(g.decor, g.Sign(tex, bx0-1, by0+3, tableH+4, bx0, by1-3, tableH+28, "w", texCrt))
		}
		kw, kd, cw, cd := 5.0, 14.0, 2.0, 12.0 // keyboard and chair-back half sizes across / along the facing
		if dx == 0 {
			kw, kd, cw, cd = 14, 5, 12, 2
		}
		kx, ky := px+dx*44, py+dy*44
		chx, chy := px+dx*86, py+dy*86
		bx, by := px+dx*98, py+dy*98
		g.decor = append(g.decor, g.box(kx-kw, ky-kd, tableH, kx+kw, ky+kd, tableH+2, "KEYBOARD"),
			g.box(chx-12, chy-12, hf, chx+12, chy+12, hf+18, texChair),
			g.box(bx-cw, by-cd, hf+18, bx+cw, by+cd, hf+50, texChair))
	}
	rgb := [3]int{60, 220, 255}
	switch {
	case math.Abs(px) > 300 && px < 0:
		rgb = [3]int{255, 90, 190}
	case math.Abs(px) <= 300 && px < 0:
		rgb = [3]int{255, 140, 110}
	case math.Abs(px) <= 300:
		rgb = [3]int{160, 110, 255}
	}
	g.light(px, py, -200, rgb, 230)
}

// ------------------------------------------------------------------ interactive: "Restore the Power"

func (g *gen) restoreThePower() {
	// breaker switches (3): north garage, south loop tunnel, far west end of the street
	breakers := []string{g.box(gx1-32, 560, 40, gx1-16, 600, 90, texMetal, mapkit.Sides{W: mapkit.T(texBreaker)}),
		g.box(-20, -640, -150, 20, -630, -110, texMetal, mapkit.Sides{N: mapkit.T(texBreaker)}),
		g.box(-2956, 184, 50, -2944, 216, 90, texMetal, mapkit.Sides{E: mapkit.T(texBreaker)})}
	g.w = append(g.w, g.box(-2964, 196, 0, -2956, 204, 200, texMetal)) // utility pole for breaker 3
	for _, b := range breakers {
		g.bent("func_button", []string{b}, "target", "breaker_mm", "wait", 45, "spawnflags", 1, "sounds", 21, "delay", 0)
	}
	g.point("multi_manager", -100, -560, -120, "targetname", "breaker_mm", "breaker_count", 0, "breaker_msg", 0)
	g.point("game_counter", -100, -560, -110, "targetname", "breaker_count", "target", "power_mm", "health", 3, "frags", 0, "spawnflags", 2)
	g.point("game_text", -100, -560, -100, "targetname", "breaker_msg", "message", "A breaker was switched on.  (Flip all 3 to restore power)",
		"x", -1, "y", 0.7, "effect", 0, "color", "255 220 90", "color2", "255 255 255", "fadein", 0.05, "fadeout", 0.5, "holdtime", 3,
		"channel", 2, "spawnflags", 1)
	g.point("multi_manager", -100, -560, -90, "targetname", "power_mm", "msg_power", 0, "boom", 0.2, "shake", 0.2, "alarm_snd", 0.6,
		"server_door", 1.5, "alarm_snd#2", 16)
	g.point("game_text", -100, -560, -80, "targetname", "msg_power", "message", "POWER RESTORED!  The JustinCon server room is open.",
		"x", -1, "y", 0.25, "effect", 0, "color", "255 90 200", "color2", "60 230 255", "fadein", 0.05, "fadeout", 0.6, "holdtime", 5,
		"channel", 1, "spawnflags", 1)
	// the transformer in the park blows when power comes back
	g.w = append(g.w, g.box(-870, 380, 0, -800, 440, 72, "TRANSFORMER", mapkit.Sides{Top: mapkit.T(texMetal)}))
	g.point("env_explosion", -835, 410, 84, "targetname", "boom", "iMagnitude", 150, "spawnflags", 2)
	g.point("env_shake", 0, 0, 0, "targetname", "shake", "amplitude", 8, "frequency", 30, "duration", 2.5, "radius", 5000, "spawnflags", 1)
	for _, p := range [][3]float64{{0, 0, -300}, {-835, 410, 100}, {-560, -470, -400}} {
		g.point("ambient_generic", p[0], p[1], p[2], "targetname", "alarm_snd", "message", "justincon/alarm.wav", "health", 9,
			"spawnflags", 4|16, "pitch", 100, "pitchstart", 100)
	}
	g.serverRoom()
	// stage marquee (animated) + teleport pad on the stage -> blue house roof
	g.decor = append(g.decor, g.sign(texMarquee0, -320, -280, hf, 320, -278, hf+32, "n"),
		g.box(220, -360, hf+32, 290, -300, hf+33, texTelepad))
	g.bent("trigger_teleport", []string{g.box(222, -358, hf+33, 288, -302, hf+90, "AAATRIGGER")}, "target", "tele_roof")
	g.point("info_teleport_destination", 384, 1150, roof, "targetname", "tele_roof", "angles", "0 270 0")
	g.light(255, -330, -380, [3]int{255, 120, 230}, 90)
	// breakable north-garage door + crates that drop weapons
	g.bent("func_breakable", []string{g.garageDoor()}, "material", 2, "health", 150, "gibmodel", "", "explodemagnitude", 0)
	for _, c := range [][3]int{{1400, 520, 14}, {1460, 640, 8}, {1520, 520, 2}} {
		cx, cy := float64(c[0]), float64(c[1])
		g.bent("func_breakable", []string{g.box(cx-20, cy-20, 1, cx+20, cy+20, 41, "CRATE")}, "material", 1, "health", 30, "spawnobject", c[2])
	}
	g.light(1440, 576, 110, [3]int{255, 230, 190}, 110)
	// steam-vent updraft behind S2 house: jump in to ride up to the roof
	g.w = append(g.w, g.box(344, -1356, 0, 424, -1284, 6, "VENT"))
	g.bent("trigger_push", []string{g.box(352, -1348, 6, 416, -1286, 176, "AAATRIGGER")}, "angles", "0 -1 0", "speed", 1100)
	g.point("env_sprite", 384, -1316, 40, "model", "sprites/prairie_steam.spr", "rendermode", 5, "renderamt", 160,
		"rendercolor", "255 255 255", "framerate", 12, "scale", 2.5, "spawnflags", 1)
	// rotating ceiling fans over the LAN hall
	for _, fx := range []float64{-320, 320} {
		g.w = append(g.w, g.box(fx-3, 13, -104, fx+3, 19, hc, texMetal))
		blades := []string{g.box(fx-112, 8, -112, fx+112, 24, -110, texFanblade), g.box(fx-8, 16-104, -112, fx+8, 16+104, -110, texFanblade),
			g.box(fx-10, 6, -116, fx+10, 26, -104, texMetal), g.box(fx-2, 14, -112, fx+2, 18, -108, "ORIGIN")}
		speed := -80
		if fx < 0 {
			speed = 80
		}
		g.bent("func_rotating", blades, "speed", speed, "spawnflags", 1|64, "fanfriction", 20)
	}
	// chargers: health in the BattleAxe tunnel, armour in the Crescent tunnel
	g.bent("func_healthcharger", []string{g.box(-1088, -420, -150, -1084, -372, -100, texMetal, mapkit.Sides{E: mapkit.T(texChargerH)})})
	g.bent("func_recharge", []string{g.box(1084, -420, -150, 1088, -372, -100, texMetal, mapkit.Sides{W: mapkit.T(texChargerA)})})
}

// serverRoom is the room behind a sliding door in the hall's south wall that opens when the power is back.
func (g *gen) serverRoom() {
	door := mapkit.Sides{N: g.FitBox(texServerdoor, "n", sdoorX0, hallY0, sdoorX1, hallY0+7, hf, -352),
		S: g.FitBox(texServerdoor, "s", sdoorX0, hallY0, sdoorX1, hallY0+7, hf, -352)}
	g.bent("func_door", []string{g.box(sdoorX0, hallY0+1, hf, sdoorX1, hallY0+7, -352, texMetal, door)},
		"targetname", "server_door", "angles", "0 -1 0", "speed", 110, "lip", 8, "wait", 14, "movesnd", 1, "stopsnd", 1)
	g.decor = append(g.decor, g.sign(texMarquee0, sdoorX0-20, hallY0+8, -344, sdoorX1+20, hallY0+10, -324, "n"))
	for _, r := range []struct {
		x0   float64
		face string
	}{{srvX0 + 4, "e"}, {srvX1 - 36, "w"}} {
		for _, y0 := range []float64{-544, -480} {
			g.w = append(g.w, g.box(r.x0, y0, hf, r.x0+32, y0+56, -372, texMetal, mapkit.SetSide(mapkit.Sides{}, r.face, mapkit.T(texServerrack))))
		}
	}
	g.bent("func_button", []string{g.box(-580, srvY0, -420, -540, srvY0+6, -380, texMetal,
		mapkit.Sides{N: g.FitBox(texArmory, "n", -580, srvY0, -540, srvY0+6, -420, -380)})}, "target", "armory", "wait", 3, "spawnflags", 1, "sounds", 9)
	g.point("game_player_equip", -560, -500, -440, "targetname", "armory", "spawnflags", 1,
		"weapon_crowbar", 1, "weapon_9mmhandgun", 1, "weapon_357", 1, "weapon_9mmAR", 1, "weapon_shotgun", 1, "weapon_crossbow", 1,
		"weapon_rpg", 1, "weapon_gauss", 1, "weapon_egon", 1, "weapon_hornetgun", 1, "weapon_handgrenade", 1, "weapon_satchel", 1,
		"weapon_tripmine", 1, "weapon_snark", 1, "ammo_buckshot", 2, "ammo_9mmAR", 2, "ammo_357", 2, "ammo_rpgclip", 2, "ammo_gaussclip", 2)
	g.item("weapon_gauss", -600, -460, hf+10)
	g.item("weapon_egon", -520, -460, hf+10)
	g.item("item_battery", -560, -420, hf+10)
	g.light(-560, -470, -360, [3]int{255, 70, 70}, 120)
	g.light(-560, -420, -380, [3]int{200, 200, 255}, 60)
}

// ------------------------------------------------------------------ weather + wind

// snowSpans splits the line x = fixed (or y = fixed when alongX) from lo to hi into (a, b, z_bottom) runs:
// over a roof a snow sheet starts at the roof, so it never snows indoors.
func snowSpans(fixed, lo, hi float64, alongX bool, roofs [][4]float64) [][3]float64 {
	cuts := []float64{lo, hi}
	for _, r := range roofs {
		c0, c1, f0, f1 := r[1], r[3], r[0], r[2]
		if alongX {
			c0, c1, f0, f1 = r[0], r[2], r[1], r[3]
		}
		if f0-8 < fixed && fixed < f1+8 {
			cuts = append(cuts, min(max(c0-8, lo), hi), min(max(c1+8, lo), hi))
		}
	}
	cuts = uniqSorted(cuts)
	var out [][3]float64
	for i := range len(cuts) - 1 {
		a, b := cuts[i], cuts[i+1]
		if b-a < 8 {
			continue
		}
		m := (a + b) / 2
		px, py := fixed, m
		if alongX {
			px, py = m, fixed
		}
		z0 := snowBot
		for _, r := range roofs {
			if r[0]-8 < px && px < r[2]+8 && r[1]-8 < py && py < r[3]+8 {
				z0 = roof + 2
				break
			}
		}
		if n := len(out); n > 0 && out[n-1][2] == z0 {
			out[n-1][1] = b
		} else {
			out = append(out, [3]float64{a, b, z0})
		}
	}
	return out
}

// weather is continuous snowfall: a crossing grid of see-through (additive) animated sheets over the whole outdoor
// area, so snow is everywhere instead of in sprite patches. Sheets are func_illusionary: no collision, no effect
// on the BSP.
func (g *gen) weather() {
	var roofs [][4]float64
	for _, h := range layout.Houses {
		roofs = append(roofs, layout.Footprint(h))
	}
	roofs = append(roofs, layout.Garages...)
	tex := mapkit.T("+0SNOWFALL").WithScale(4, 4)
	for x := xw + snowGap/2; x < xe; x += snowGap {
		for _, s := range snowSpans(x, ys, yn, false, roofs) {
			g.snow = append(g.snow, g.BoxT(x-1, s[0], s[2], x+1, s[1], snowTop, tex))
		}
	}
	for y := ys + snowGap/2; y < yn; y += snowGap {
		for _, s := range snowSpans(y, xw, xe, true, roofs) {
			g.snow = append(g.snow, g.BoxT(s[0], y-1, s[2], s[1], y+1, snowTop, tex))
		}
	}
	for _, part := range mapkit.Chunks(g.snow, 512) { // falling-snow sheets, in chunks so each has a small bbox
		g.bent("func_illusionary", part, "rendermode", 5, "renderamt", 200)
	}
	for _, b := range [][3]float64{{-2850, 0, 7}, {-2650, -160, 9}, {-2950, 200, 11}, {-2500, 120, 8}} {
		g.point("env_sprite", b[0], b[1], 380, "model", "sprites/prairie_burst.spr", "rendermode", 5, "renderamt", 210,
			"rendercolor", "255 255 255", "framerate", int(b[2]), "scale", 6, "spawnflags", 1)
	}
	for _, p := range [][2]float64{{-1700, 0}, {-2400, 0}, {-1000, 900}, {0, 880}, {1000, 900}, {-1000, -900}, {0, -880}, {1000, -900},
		{1700, 600}, {1700, -600}, {-1300, 1300}, {1300, 1300}} {
		g.point("ambient_generic", p[0], p[1], 120, "message", "justincon/wind_loop.wav", "health", 8, "spawnflags", 4, "pitch", 100, "pitchstart", 100)
	}
	// night sky fill: blue-grey storm light from above
	for _, p := range [][2]float64{{-2500, 0}, {-1200, 900}, {-1200, -900}, {300, 1000}, {300, -1000}, {1500, 0}, {0, 0}} {
		g.light(p[0], p[1], 950, [3]int{140, 155, 185}, 1700)
	}
}

// ------------------------------------------------------------------ weapons + spawns

var (
	weapons = []string{"weapon_shotgun", "weapon_9mmAR", "weapon_357", "weapon_crossbow", "weapon_rpg", "weapon_gauss", "weapon_egon",
		"weapon_hornetgun", "weapon_handgrenade", "weapon_satchel", "weapon_snark", "weapon_tripmine"}
	ammo = []string{"ammo_buckshot", "ammo_9mmAR", "ammo_357", "ammo_crossbow", "ammo_rpgclip", "ammo_gaussclip", "ammo_ARgrenades",
		"item_battery", "item_healthkit"}
)

func (g *gen) weapon(x, y, z float64) {
	g.item(weapons[g.weaponN%len(weapons)], x, y, z)
	g.weaponN++
}

func (g *gen) ammo(x, y, z float64) {
	g.item(ammo[g.weaponN%len(ammo)], x, y, z)
	g.weaponN++
}

func (g *gen) weaponsAndSpawns() {
	for _, p := range pods { // hall: between pods
		dy := -128.0
		if p[1] < 0 {
			dy = 128
		}
		g.weapon(p[0], p[1]+dy, hf+10)
	}
	for _, p := range [][2]float64{{-650, 0}, {650, 0}, {0, 10}, {-320, 10}, {320, 10}, {0, -220}} {
		g.ammo(p[0], p[1], hf+10)
	}
	g.item("weapon_gauss", 0, -330, hf+42)
	g.item("weapon_egon", 0, 330, hf+10)
	for i := 0; i < len(tunnelLamps); i += 2 {
		g.weapon(tunnelLamps[i][0], tunnelLamps[i][1], bf+10)
	}
	for _, h := range layout.Houses {
		x, y := layout.L2W(h, -120, -160)
		g.weapon(x, y, 10)
		x, y = layout.L2W(h, 100, -100)
		g.ammo(x, y, bf+10)
	}
	for _, p := range [][2]float64{{-2000, 0}, {-1400, -300}, {0, 620}, {0, -620}, {-1100, 0}, {500, 0}, {-500, 0}, {1150, 0}} {
		g.weapon(p[0], p[1], 10)
	}
	// roof weapons out on the bridges and open roofs, away from the roof spawns (strong weapons in risky spots)
	for _, p := range [][2]float64{{-768, 1056}, {-384, 1150}, {768, -1056}, {-1152, -1150}, {1440, 352}, {1440, 576}} {
		g.weapon(p[0], p[1], roof+10)
	}
	g.item("weapon_crossbow", 384, 1200, roof+10)
	g.item("weapon_rpg", -384, -1200, roof+10)
	for _, h := range layout.Houses {
		x, y := layout.L2W(h, 140, -60) // kitchen side, across the room from the house's weapon
		g.spawn(x, y, 0)
	}
	for _, s := range []struct {
		z   float64
		pts [][2]float64
	}{
		{bf, [][2]float64{{-700, 576}, {300, 576}, {-1024, -200}, {1024, 200}, {-300, -576}, {250, -576}}},
		{hf, [][2]float64{{-300, -240}, {650, 200}, {-720, -160}, {720, 160}, {-560, 330}, {-650, 300}}},
		{0, [][2]float64{{-1700, 60}, {-2400, 60}, {-760, -120}, {600, -300}, {1600, -100}}},
		{roof, [][2]float64{{-1152, 1150}, {1152, -1150}, {1504, -100}}},
	} {
		for _, p := range s.pts {
			g.spawn(p[0], p[1], s.z)
		}
	}
}
