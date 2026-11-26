package main

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/paint"
	tx "halflifemaps/tools/mapkit/textures"
)

// All artwork here is generated and original.

var prng = rand.New(rand.NewPCG(2026, 0)) //nolint:gosec // texture noise, not security

type col = paint.Col

func seeded(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 0)) } //nolint:gosec // repeatable art

func snow(seed uint64) tx.Draw {
	return func() *paint.Canvas {
		r := seeded(seed)
		a := tx.Noise(128, 128, 3).Mul(0.5).Add(paint.Random(r, 128, 128).Mul(0.5))
		return paint.Colorize(a, col{196, 204, 220}, col{240, 244, 252}).Canvas()
	}
}

func snowroad() *paint.Canvas {
	a := tx.Noise(128, 128, 3).Mul(0.5).Add(paint.Random(prng, 128, 128).Mul(0.5))
	c := paint.Colorize(a, col{170, 176, 190}, col{225, 230, 240})
	for _, x0 := range []int{22, 40, 86, 104} { // tyre ruts
		c.BlendRect(x0, 0, x0+8, 128, 0.72, col{40, 40, 44})
	}
	return c.Canvas()
}

func ice() *paint.Canvas {
	im := paint.Colorize(tx.Noise(128, 128, 6).Mul(0.6).Plus(0.4), col{150, 185, 210}, col{205, 228, 242}).Canvas()
	for range 40 {
		x, y := float64(prng.IntN(128)), float64(prng.IntN(128))
		l := float64(10 + prng.IntN(30))
		ang := prng.Float64() * math.Pi
		im.Line([]float64{x, y, x + l*math.Cos(ang), y + l*math.Sin(ang)}, col{235, 245, 252}, 1)
	}
	return im
}

func rinkboard() *paint.Canvas {
	a := paint.Colorize(tx.Noise(64, 32, 1).Mul(0.2).Plus(0.8), col{220, 222, 225}, col{245, 246, 248})
	a.SetRect(0, 20, 64, 26, col{40, 70, 170})
	a.SetRect(0, 0, 64, 3, col{230, 60, 60})
	return a.Canvas()
}

func garagedoor() *paint.Canvas {
	a := paint.Colorize(tx.Noise(64, 64, 1).Mul(0.2).Plus(0.8), col{190, 192, 195}, col{225, 226, 228})
	a.MulRows(0, 16, 0.6)
	a.MulRows(1, 16, 0.6)
	return a.Canvas()
}

func noiseTex(w, h int, blur, k, base float64, c0, c1 col) tx.Draw {
	return func() *paint.Canvas {
		return paint.Colorize(tx.Noise(w, h, blur).Mul(k).Plus(base), c0, c1).Canvas()
	}
}

func mixTex(rk, nk, blur float64, c0, c1 col) tx.Draw {
	return func() *paint.Canvas {
		return paint.Colorize(paint.Random(prng, 64, 64).Mul(rk).Add(tx.Noise(64, 64, blur).Mul(nk)), c0, c1).Canvas()
	}
}

func pine() *paint.Canvas {
	a := paint.Colorize(paint.Random(prng, 64, 64).Mul(0.6).Add(tx.Noise(64, 64, 1).Mul(0.4)), col{14, 38, 24}, col{40, 80, 50})
	m := paint.Random(prng, 64, 64)
	for y := range 64 {
		for x := range 64 {
			if m.At(x, y) > 0.86 {
				a.SetPx(x, y, col{215, 222, 235}) // snow on the needles
			}
		}
	}
	return a.Canvas()
}

func plate(text string, fg, bg col, size float64) tx.Draw {
	return func() *paint.Canvas {
		im := paint.NewCanvas(128, 32, bg)
		im.Text(64, 16, text, fg, tx.Bold(size), "mm")
		return im
	}
}

func strip(c col) tx.Draw { return tx.Solid(16, 16, c) }

// axeEmblem is an original angular axe: diagonal haft + faceted double-bit head.
func axeEmblem(d *paint.Canvas, cx, cy, s float64, fill, edge col) {
	d.Line([]float64{cx - 0.9*s, cy + 0.9*s, cx + 0.55*s, cy - 0.55*s}, col{70, 40, 30}, max(2, int(s*0.16)))
	hx, hy := cx+0.45*s, cy-0.45*s
	head := []float64{hx - 0.15*s, hy - 0.62*s, hx + 0.5*s, hy - 0.35*s, hx + 0.62*s, hy + 0.15*s, hx + 0.15*s, hy + 0.05*s, hx - 0.05*s, hy - 0.15*s}
	d.Polygon(head, fill)
	d.PolygonOutline(head, edge, 1)
	back := []float64{hx - 0.05*s, hy - 0.15*s, hx - 0.45*s, hy - 0.2*s, hx - 0.3*s, hy + 0.05*s, hx - 0.1*s, hy + 0.12*s}
	d.Polygon(back, fill)
	d.PolygonOutline(back, edge, 1)
}

func moonEmblem(d *paint.Canvas, cx, cy, s float64, fill, bg col) {
	d.Ellipse(cx-s, cy-s, cx+s, cy+s, fill)
	d.Ellipse(cx-s*0.55, cy-s*1.05, cx+s*1.25, cy+s*0.75, bg)
}

func emblemPlate(axe bool) tx.Draw {
	return func() *paint.Canvas {
		bg := col{8, 14, 34}
		if axe {
			bg = col{25, 8, 22}
		}
		im := paint.NewCanvas(64, 64, bg)
		if axe {
			im.RectOutline(1, 1, 62, 62, col{255, 110, 90}, 2)
			axeEmblem(im, 32, 34, 26, col{255, 70, 180}, col{255, 170, 150})
		} else {
			im.RectOutline(1, 1, 62, 62, col{150, 90, 255}, 2)
			moonEmblem(im, 30, 32, 20, col{60, 230, 255}, bg)
		}
		return im
	}
}

func arcadeCarpet() *paint.Canvas {
	r := seeded(98)
	im := paint.NewCanvas(128, 128, col{14, 10, 40})
	cols := []col{{255, 60, 190}, {40, 230, 255}, {255, 210, 60}, {150, 80, 255}, {90, 255, 140}}
	for range 26 {
		x, y := float64(r.IntN(128)), float64(r.IntN(128))
		c := cols[r.IntN(len(cols))]
		switch r.IntN(4) {
		case 0:
			im.PolygonOutline([]float64{x, y, x + 10, y + 3, x + 3, y + 11}, c, 2)
		case 1:
			im.Arc(x, y, x+12, y+12, 0, 270, c, 2)
		case 2:
			pts := make([]float64, 0, 10)
			for i := range 5 {
				pts = append(pts, x+float64(i*4), y+float64(i%2*5))
			}
			im.Line(pts, c, 2)
		default:
			im.Ellipse(x, y, x+5, y+5, c)
		}
	}
	return im
}

func gridTex(w int, blur, k float64, c0, c1 col, step int, rows, cols col) tx.Draw {
	return func() *paint.Canvas {
		a := paint.Colorize(tx.Noise(w, w, blur).Mul(k), c0, c1)
		a.SetRows(0, step, rows)
		a.SetCols(0, step, cols)
		return a.Canvas()
	}
}

func stageback() *paint.Canvas {
	const w, h = 256, 128
	im := paint.NewCanvas(w, h, col{})
	for y := range 64 { // vaporwave sunset sky
		t := float64(y) / 63
		im.Line([]float64{0, float64(y), w, float64(y)}, col{float64(int(40 + 215*t)), float64(int(10 + 70*t)), float64(int(90 + 60*(1-t)))}, 1)
	}
	for i, y := 0, 20; y < 60; i, y = i+1, y+6 { // striped sun
		hw := float64(int(math.Sqrt(math.Max(0, 34*34-float64((y-50)*(y-50))))))
		wd := 2
		if i%2 == 0 {
			wd = 4
		}
		im.Line([]float64{128 - hw, float64(y), 128 + hw, float64(y)}, col{255, float64(200 - i*12), float64(80 + i*10)}, wd)
	}
	im.Rect(0, 64, w, h, col{20, 6, 40})
	for x := -512; x < 768; x += 32 { // perspective grid
		im.Line([]float64{128, 64, float64(x), h}, col{255, 60, 190}, 1)
	}
	for i := range 7 {
		y := 64 + float64(int(64*math.Pow(float64(i)/6, 1.8)))
		im.Line([]float64{0, y, w, y}, col{40, 230, 255}, 1)
	}
	return im
}

func neonSign() *paint.Canvas {
	glow := paint.NewCanvas(256, 64, col{})
	glow.Text(128, 30, "JustinCon", col{255, 70, 200}, tx.Bold(44), "mm")
	glow = glow.Blur(5)
	a := paint.Full(256, 64, col{6, 2, 14})
	g := glow.Arr()
	for i := range a.P {
		a.P[i] = min(a.P[i]+g.P[i]*2, 255)
	}
	im := a.Canvas()
	im.Text(128, 30, "JustinCon", col{255, 225, 250}, tx.Bold(44), "mm")
	im.RectOutline(2, 2, 253, 61, col{40, 230, 255}, 3)
	return im
}

func matchboard() *paint.Canvas {
	im := paint.NewCanvas(256, 64, col{5, 5, 12})
	im.Rect(0, 0, 127, 63, col{45, 8, 30})
	im.Rect(128, 0, 255, 63, col{8, 18, 50})
	axeEmblem(im, 20, 32, 14, col{255, 70, 180}, col{255, 170, 150})
	moonEmblem(im, 236, 32, 12, col{60, 230, 255}, col{8, 18, 50})
	im.Text(74, 20, texBattleaxe, col{255, 130, 200}, tx.Bold(15), "mm")
	im.Text(184, 20, texCrescent, col{110, 235, 255}, tx.Bold(15), "mm")
	im.Text(128, 20, "vs", col{255, 255, 255}, tx.Bold(12), "mm")
	im.Text(74, 46, "7", col{255, 210, 60}, tx.Bold(20), "mm")
	im.Text(184, 46, "7", col{255, 210, 60}, tx.Bold(20), "mm")
	im.Text(128, 46, "MAP 3", col{200, 200, 220}, tx.Mono(9), "mm")
	return im
}

func banner(axe bool) tx.Draw {
	return func() *paint.Canvas {
		const w, h = 128, 256
		bg, a1, a2, name := col{6, 12, 44}, col{40, 230, 255}, col{150, 80, 255}, texCrescent
		if axe {
			bg, a1, a2, name = col{40, 6, 30}, col{255, 70, 180}, col{255, 120, 90}, texBattleaxe
		}
		im := paint.NewCanvas(w, h, bg)
		im.RectOutline(4, 4, w-5, h-5, a2, 4)
		im.Polygon([]float64{4, h - 40, 64, h - 5, 124, h - 40, 124, h - 5, 4, h - 5}, col{})
		if axe {
			axeEmblem(im, 64, 92, 46, a1, col{255, 200, 180})
		} else {
			moonEmblem(im, 60, 92, 40, a1, bg)
		}
		im.Text(64, 170, name, col{255, 255, 255}, tx.Bold(17), "mm")
		im.Line([]float64{20, 186, 108, 186}, a2, 3)
		return im
	}
}

// handles are invented player handles.
var handles = []string{"FragTaco", "PixelMoose", "LagWizard", "SnowCrab", "QuadToast", "NullHornet", "ByteKnight",
	"CampKing", "RocketRita", "GibMaster", "Wavedash", "MidnightMod"}

func handleTex(name string) string {
	s := "H_" + strings.ToUpper(name)
	return s[:min(len(s), 15)]
}

func handleScreen(name string, axe bool) tx.Draw {
	return func() *paint.Canvas {
		const w, h = 128, 96
		bg, acc, faction := col{4, 14, 34}, col{60, 225, 255}, texCrescent
		if axe {
			bg, acc, faction = col{16, 6, 30}, col{255, 90, 190}, texBattleaxe
		}
		im := paint.NewCanvas(w, h, bg)
		im.Rect(0, 0, w-1, 15, acc)
		im.Text(4, 8, "PLAYER", bg, tx.Mono(10), "lm")
		im.Text(w-4, 8, faction, bg, tx.Bold(9), "rm")
		size := 30.0
		fnt := tx.Bold(size)
		for size > 10 {
			fnt = tx.Bold(size)
			if fnt.Length(name) <= w-10 {
				break
			}
			size--
		}
		im.Text(w/2, 50, name, col{255, 255, 255}, fnt, "mm")
		im.Line([]float64{10, 70, w - 10, 70}, acc, 2)
		im.Text(w/2, 83, "READY  >_", acc, tx.Mono(11), "mm")
		a := im.Arr()
		a.MulRows(0, 2, 0.85) // scanlines
		return a.Canvas()
	}
}

func crtGame() *paint.Canvas {
	im := paint.NewCanvas(128, 96, col{})
	for y := range 48 {
		im.Line([]float64{0, float64(y), 127, float64(y)}, col{float64(20 + y), 10, float64(50 + y*2)}, 1)
	}
	im.Rect(0, 48, 127, 95, col{30, 20, 40})
	for x := -200; x < 330; x += 24 {
		im.Line([]float64{64, 48, float64(x), 95}, col{255, 60, 190}, 1)
	}
	for i := range 5 {
		y := float64(48 + 6*i*i/2)
		im.Line([]float64{0, y, 127, y}, col{40, 230, 255}, 1)
	}
	im.Line([]float64{60, 40, 68, 40}, col{255, 255, 255}, 1)
	im.Line([]float64{64, 36, 64, 44}, col{255, 255, 255}, 1)
	im.Text(4, 4, "FRAGS 12", col{255, 255, 255}, tx.Mono(9), "")
	return im
}

func crtSaver() *paint.Canvas {
	im := paint.NewCanvas(128, 96, col{0, 0, 8})
	im.Text(64, 40, "JustinCon", col{255, 90, 200}, tx.Bold(20), "mm")
	im.Text(64, 64, "LAN '99", col{60, 225, 255}, tx.Mono(12), "mm")
	return im
}

func crtTerm() *paint.Canvas {
	im := paint.NewCanvas(128, 96, col{0, 10, 4})
	for i, t := range []string{`C:\> ping 10.0.0.1`, "Reply: time<1ms", `C:\> net use`, "OK", `C:\> _`} {
		im.Text(4, float64(4+i*17), t, col{70, 255, 120}, tx.Mono(10), "")
	}
	return im
}

func poster98() *paint.Canvas {
	im := paint.NewCanvas(64, 96, col{20, 8, 40})
	im.RectOutline(0, 0, 63, 95, col{255, 70, 200}, 2)
	im.Text(32, 22, "Justin", col{255, 255, 255}, tx.Bold(14), "mm")
	im.Text(32, 38, "Con", col{255, 90, 200}, tx.Bold(14), "mm")
	im.Text(32, 62, "'98", col{60, 225, 255}, tx.Bold(18), "mm")
	im.Text(32, 84, "BYOPC", col{255, 220, 90}, tx.Mono(9), "mm")
	return im
}

func newscast() *paint.Canvas {
	im := paint.NewCanvas(128, 64, col{10, 20, 60})
	im.Text(64, 20, "WINTER STORM", col{255, 255, 255}, tx.Bold(15), "mm")
	im.Rect(0, 40, 127, 63, col{180, 20, 30})
	im.Text(64, 51, "STAY INSIDE. PLAY LAN.", col{255, 255, 255}, tx.Bold(9), "mm")
	return im
}

func breaker() *paint.Canvas {
	im := paint.NewCanvas(64, 64, col{110, 112, 118})
	im.RectOutline(2, 2, 61, 61, col{60, 62, 66}, 3)
	for i := 0; i < 64; i += 8 { // hazard band
		im.Line([]float64{float64(i), 0, float64(i + 8), 8}, col{230, 190, 30}, 3)
	}
	im.Rect(22, 20, 42, 50, col{40, 40, 44})
	im.Rect(27, 24, 37, 36, col{220, 40, 40}) // lever
	im.Text(32, 56, texBreaker, col{20, 20, 20}, tx.Bold(8), "mm")
	return im
}

func transformer() *paint.Canvas {
	im := paint.Colorize(tx.Noise(64, 64, 1).Mul(0.3).Plus(0.7), col{60, 90, 60}, col{90, 125, 90}).Canvas()
	im.Rect(8, 18, 56, 44, col{240, 200, 30})
	im.Text(32, 26, "DANGER", col{}, tx.Bold(9), "mm")
	im.Text(32, 37, "HIGH VOLTAGE", col{}, tx.Bold(6), "mm")
	return im
}

func telepad() *paint.Canvas {
	im := paint.NewCanvas(64, 64, col{10, 6, 24})
	for _, rc := range []struct {
		r float64
		c col
	}{{30, col{255, 70, 200}}, {22, col{60, 230, 255}}, {14, col{255, 70, 200}}, {6, col{255, 255, 255}}} {
		im.EllipseOutline(32-rc.r, 32-rc.r, 32+rc.r, 32+rc.r, rc.c, 3)
	}
	return im
}

func vent() *paint.Canvas {
	a := paint.Colorize(tx.Noise(64, 64, 1).Mul(0.3).Plus(0.5), col{40, 42, 46}, col{90, 92, 98})
	a.MulRows(0, 8, 0.3)
	return a.Canvas()
}

func serverdoor() *paint.Canvas {
	im := paint.Colorize(tx.Noise(64, 64, 2).Mul(0.2).Plus(0.8), col{70, 72, 82}, col{110, 112, 125}).Canvas()
	im.RectOutline(4, 4, 59, 59, col{40, 40, 48}, 2)
	im.Rect(6, 22, 57, 36, col{30, 10, 40})
	im.Text(32, 29, "SERVER ROOM", col{255, 100, 210}, tx.Bold(7), "mm")
	for _, y := range []float64{46, 52} {
		im.Line([]float64{10, y, 54, y}, col{50, 50, 58}, 2)
	}
	return im
}

func armory() *paint.Canvas {
	im := paint.NewCanvas(64, 64, col{30, 30, 36})
	im.Ellipse(14, 10, 50, 46, col{200, 20, 30})
	im.EllipseOutline(14, 10, 50, 46, col{255, 120, 120}, 3)
	im.Text(32, 55, texArmory, col{255, 220, 90}, tx.Bold(10), "mm")
	return im
}

func charger(health bool) tx.Draw {
	return func() *paint.Canvas {
		bg := col{40, 70, 140}
		if health {
			bg = col{200, 205, 210}
		}
		im := paint.NewCanvas(64, 64, bg)
		im.RectOutline(3, 3, 60, 60, col{70, 70, 80}, 3)
		if health {
			im.Rect(27, 14, 37, 46, col{40, 180, 80}) // green plus
			im.Rect(16, 25, 48, 35, col{40, 180, 80})
			im.Text(32, 55, "HEALTH", col{30, 30, 30}, tx.Bold(8), "mm")
		} else {
			im.Polygon([]float64{36, 10, 22, 36, 31, 36, 27, 54, 44, 26, 34, 26}, col{255, 220, 60}) // bolt
			im.Text(32, 58, "ARMOR", col{255, 255, 255}, tx.Bold(7), "mm")
		}
		return im
	}
}

func crate() *paint.Canvas {
	a := paint.Colorize(tx.Noise(64, 64, 0.8), col{110, 80, 45}, col{165, 125, 75})
	a.MulRect(0, 0, 64, 4, 0.5)
	a.MulRect(0, -4, 64, 64, 0.5)
	a.MulRect(0, 0, 4, 64, 0.5)
	a.MulRect(-4, 0, 64, 64, 0.5)
	im := a.Canvas()
	im.Line([]float64{4, 4, 60, 60}, col{90, 60, 30}, 4)
	return im
}

func marqueeFrame(k int) tx.Draw {
	return func() *paint.Canvas {
		const w, h = 256, 32
		im := paint.NewCanvas(w, h, col{12, 4, 24})
		im.Text(w/2, h/2, "BATTLEAXE  vs  CRESCENT   * LIVE *", col{255, 240, 255}, tx.Bold(12), "mm")
		cols := []col{{255, 70, 200}, {60, 230, 255}}
		var pts [][2]int
		for x := 2; x < w; x += 8 {
			pts = append(pts, [2]int{x, 2})
		}
		for y := 2; y < h; y += 8 {
			pts = append(pts, [2]int{w - 3, y})
		}
		for x := w - 3; x > 0; x -= 8 {
			pts = append(pts, [2]int{x, h - 3})
		}
		for y := h - 3; y > 0; y -= 8 {
			pts = append(pts, [2]int{2, y})
		}
		for i, p := range pts { // chasing lights: three bulbs in ten lit, shifting each frame
			c := col{40, 20, 50}
			if mapkit.PyMod(i-k, 10) < 3 {
				c = cols[(i/10)%2]
			}
			x, y := float64(p[0]), float64(p[1])
			im.Ellipse(x-2, y-2, x+2, y+2, c)
		}
		return im
	}
}

// snowfallFrame is one frame of the continuous snowfall sheet (additive: black = clear). Each flake falls 1 px
// (4 units at scale 4) per frame and fades in and out over its own 10-frame life, phases staggered, so the
// 10-frame loop (1 s in game) has no visible restart.
func snowfallFrame(k int) tx.Draw {
	return func() *paint.Canvas {
		const n, frames, s = 110, 10, 128
		r := seeded(21)
		x0, y0, ph, drift, peak := make([]float64, n), make([]float64, n), make([]int, n), make([]float64, n), make([]float64, n)
		for i := range n {
			x0[i] = r.Float64() * s
		}
		for i := range n {
			y0[i] = r.Float64() * s
		}
		for i := range n {
			ph[i] = r.IntN(frames)
		}
		for i := range n {
			drift[i] = []float64{0, 0.3, 0.5}[r.IntN(3)]
		}
		for i := range n {
			peak[i] = 120 + r.Float64()*90
		}
		a := paint.NewField(s, s)
		put := func(x, y int, v float64) { a.V[y*s+x] = max(a.V[y*s+x], v) }
		for i := range n {
			age := mapkit.PyMod(k-ph[i], frames)
			b := peak[i] * math.Sin(math.Pi*(float64(age)+0.5)/frames)
			x := mapkit.PyMod(int(x0[i]+drift[i]*float64(age)), s)
			y := mapkit.PyMod(int(y0[i]+float64(age)), s)
			put(x, y, b)
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				put(mapkit.PyMod(x+d[0], s), mapkit.PyMod(y+d[1], s), b*0.15)
			}
		}
		out := paint.NewArr(s, s)
		for i, v := range a.V {
			out.P[3*i], out.P[3*i+1], out.P[3*i+2] = v*0.92, v*0.95, v
		}
		return out.Canvas()
	}
}

// textureList is every texture in the map's WAD, in order.
func textureList() []mapkit.Texture {
	var t []mapkit.Texture
	add := func(name string, d tx.Draw) { t = append(t, mapkit.Texture{Name: name, Draw: d}) }
	for _, k := range []string{texWallpaper, texWoodfloor, texCeiling, texBasewall, texCeiltile, texCouch, "COUNTER", "FRIDGE",
		"TVBLACK", texStep, texTrim, texMetal, "BRICK", "CARPET", texLamp, "GLASS", texCarglass, texTire, texCarred, texCarblue,
		"CARSILVER", texWoodfence, "sky"} {
		add(k, tx.Base(k))
	}
	// outdoor
	add(texSnow, snow(1))
	add(texRoofsnow, snow(2))
	add(texSnowroad, snowroad)
	add(texIce, ice)
	add(texRinkboard, rinkboard)
	for _, s := range []struct {
		name   string
		c0, c1 col
	}{{"SIDING_BLUE", col{50, 85, 150}, col{85, 125, 195}}, {"SIDING_CREAM", col{190, 175, 140}, col{230, 218, 185}},
		{"SIDING_RED", col{120, 40, 35}, col{165, 65, 55}}, {"SIDING_GREEN", col{70, 100, 75}, col{110, 145, 115}},
		{texSidingGray, col{110, 112, 118}, col{155, 158, 165}}, {"SIDING_YELLOW", col{185, 160, 80}, col{225, 200, 120}},
		{"SIDING_WHITE", col{200, 202, 205}, col{238, 240, 242}}, {"SIDING_BROWN", col{100, 75, 55}, col{145, 110, 80}},
		{"SIDING_SAGE", col{120, 135, 110}, col{160, 175, 150}}} {
		add(s.name, tx.Siding(s.c0, s.c1))
	}
	add(texGaragedoor, garagedoor)
	add(texBark, noiseTex(32, 64, 0.7, 1, 0, col{45, 32, 22}, col{90, 65, 45}))
	add("PINE", pine)
	add("SNOWMAN", noiseTex(32, 32, 2, 0.3, 0.7, col{215, 220, 232}, col{245, 247, 252}))
	add(texCoal, noiseTex(16, 16, 1, 1, 0, col{10, 10, 12}, col{40, 40, 44}))
	add("CARROT", noiseTex(16, 16, 1, 1, 0, col{200, 90, 20}, col{240, 130, 40}))
	add(texScarf, noiseTex(16, 16, 1, 1, 0, col{150, 20, 30}, col{200, 40, 50}))
	add(texPlank, tx.Planks(col{95, 70, 45}, col{150, 115, 80}, false, 128, 16))
	add("STREETSIGN", plate("PRAIRIE LOOP", col{255, 255, 255}, col{30, 110, 60}, 14))
	add("DEADEND", plate("DEAD END", col{20, 20, 20}, col{240, 200, 40}, 16))
	// tunnels
	add(texTunnelwall, mixTex(0.4, 0.6, 2, col{60, 60, 66}, col{100, 100, 108}))
	add(texTunnelfloor, mixTex(0.5, 0.5, 1, col{45, 45, 50}, col{80, 80, 88}))
	add(texStripPink, strip(col{255, 60, 190}))
	add(texStripCoral, strip(col{255, 120, 90}))
	add(texStripCyan, strip(col{40, 230, 255}))
	add(texStripViolet, strip(col{150, 80, 255}))
	add(texStripWhite, strip(col{235, 235, 255}))
	add(texEmbAxe, emblemPlate(true))
	add(texEmbMoon, emblemPlate(false))
	add(texTsAxe, plate("BATTLEAXE  >>", col{255, 120, 200}, col{25, 8, 22}, 13))
	add(texTsMoon, plate("<<  CRESCENT", col{90, 235, 255}, col{8, 14, 34}, 13))
	add(texTsJc, plate("JUSTINCON  v", col{255, 240, 255}, col{30, 10, 45}, 13))
	// the hall
	add(texArcadecarpet, arcadeCarpet)
	add(texHallwall, gridTex(128, 2, 0.3, col{26, 14, 44}, col{50, 28, 78}, 64, col{90, 50, 140}, col{90, 50, 140}))
	add(texHallceil, gridTex(64, 1, 0.2, col{10, 8, 20}, col{25, 20, 40}, 32, col{60, 40, 90}, col{60, 40, 90}))
	add("STAGEFLOOR", gridTex(64, 2, 0.2, col{8, 8, 14}, col{30, 26, 44}, 16, col{255, 60, 190}, col{40, 230, 255}))
	add(texStageback, stageback)
	add(texNeonJc, neonSign)
	add(texMatchboard, matchboard)
	add("BANNER_AXE", banner(true))
	add("BANNER_MOON", banner(false))
	add(texDesk, noiseTex(32, 32, 2, 0.2, 0.8, col{60, 55, 70}, col{85, 80, 95}))
	add(texCrt, noiseTex(32, 32, 2, 0.2, 0.8, col{175, 165, 140}, col{215, 205, 180}))
	add(texChair, noiseTex(16, 16, 1, 0.4, 0, col{20, 20, 30}, col{55, 50, 80}))
	add("KEYBOARD", noiseTex(16, 16, 1, 0.3, 0.7, col{170, 165, 150}, col{210, 205, 190}))
	for i, h := range handles {
		add(handleTex(h), handleScreen(h, i%2 == 0))
	}
	add(texCrtGame, crtGame)
	add(texCrtSaver, crtSaver)
	add(texCrtTerm, crtTerm)
	add("POSTER98", poster98)
	add(texNewscast, newscast)
	// interactive elements
	add(texServerrack, tx.Base(texServerrack))
	add(texBreaker, breaker)
	add("TRANSFORMER", transformer)
	add(texTelepad, telepad)
	add("VENT", vent)
	add(texServerdoor, serverdoor)
	add(texArmory, armory)
	add(texChargerH, charger(true))
	add(texChargerA, charger(false))
	add(texFanblade, noiseTex(32, 32, 1, 0.3, 0, col{30, 24, 40}, col{70, 60, 90}))
	add("ORIGIN", tx.Solid(16, 16, col{255, 140, 0}))
	add("CRATE", crate)
	for k := range 10 {
		add("+"+strconv.Itoa(k)+"MARQUEE", marqueeFrame(k))
	}
	for k := range 10 {
		add("+"+strconv.Itoa(k)+"SNOWFALL", snowfallFrame(k))
	}
	return t
}

// emissive lists the glowing textures (0..255).
func emissive() []mapkit.Glow {
	g := []mapkit.Glow{{Name: texStripPink, Glow: 255}, {Name: texStripCoral, Glow: 255}, {Name: texStripCyan, Glow: 255},
		{Name: texStripViolet, Glow: 255}, {Name: texStripWhite, Glow: 230}, {Name: texNeonJc, Glow: 255},
		{Name: texMatchboard, Glow: 240}, {Name: texStageback, Glow: 200}, {Name: texCrtGame, Glow: 220},
		{Name: texCrtSaver, Glow: 220}, {Name: texCrtTerm, Glow: 220}, {Name: texLamp, Glow: 255},
		{Name: texNewscast, Glow: 220}, {Name: texEmbAxe, Glow: 140}, {Name: texEmbMoon, Glow: 140},
		{Name: texTsAxe, Glow: 200}, {Name: texTsMoon, Glow: 200}, {Name: texTsJc, Glow: 200}}
	for _, h := range handles {
		g = append(g, mapkit.Glow{Name: handleTex(h), Glow: 230})
	}
	return append(g, mapkit.Glow{Name: texMarquee0, Glow: 240}, mapkit.Glow{Name: texTelepad, Glow: 230},
		mapkit.Glow{Name: texChargerH, Glow: 120}, mapkit.Glow{Name: texChargerA, Glow: 120},
		mapkit.Glow{Name: texArmory, Glow: 150}, mapkit.Glow{Name: texBreaker, Glow: 60})
}
