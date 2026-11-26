package main

// Original textures for BattleAxe Combat League (BACL) Fight Night: an indoor arena look. Charcoal riveted panels,
// red/black branding with a double-bitted axe mark, dark chain-link cage. All names (league, fighters) are invented.

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"

	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/paint"
	tx "halflifemaps/tools/mapkit/textures"
)

type col = paint.Col

var (
	red   = col{225, 35, 45}
	redD  = col{120, 18, 26}
	blue  = col{45, 115, 255}
	white = col{245, 245, 245}
	ink   = col{14, 14, 17}
)

func seeded(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 0)) } //nolint:gosec // repeatable art

// ------------------------------------------------------------------ logo pieces

// axeIcon is a double-bitted battle axe, s = overall height in px.
func axeIcon(d *paint.Canvas, cx, cy, s float64, c col) {
	hw := math.Max(1, s*0.055)
	d.Rect(cx-hw, cy-s*0.5, cx+hw, cy+s*0.5, c)
	top, bot := cy-s*0.44, cy+s*0.02
	for _, sg := range []float64{-1, 1} {
		d.Polygon([]float64{cx + sg*hw, top + s*0.08, cx + sg*s*0.40, top - s*0.02, cx + sg*s*0.31, (top + bot) / 2,
			cx + sg*s*0.40, bot + s*0.02, cx + sg*hw, bot - s*0.08}, c)
	}
	d.Ellipse(cx-hw*1.8, cy-s*0.5-hw*1.5, cx+hw*1.8, cy-s*0.5+hw*2, c)
}

// textImg is text on a transparent layer cropped to its ink, optionally letter-spaced and slanted.
func textImg(text string, size float64, fill col, italic float64, spacing int) *paint.Canvas {
	fo := tx.Bold(size)
	x0, y0, x1, y1 := fo.BBox(text)
	h := int(y1 - y0 + 4)
	w := int(x1-x0+float64(spacing*len(text))+4) + int(italic*float64(h)) + 2
	lay := paint.NewTransparent(w, h)
	if spacing > 0 {
		x := 2 - x0
		for _, ch := range text {
			lay.Text(x, 2-y0, string(ch), fill, fo, "")
			x += fo.Length(string(ch)) + float64(spacing)
		}
	} else {
		lay.Text(2-x0, 2-y0, text, fill, fo, "")
	}
	if italic != 0 {
		lay = lay.Shear(italic)
	}
	return lay
}

// logo is the axe mark + slanted 'BattleAxe' + spaced 'COMBAT LEAGUE', transparent around it.
func logo(h int, axe, fg col) *paint.Canvas {
	fh := float64(h)
	big := textImg("BattleAxe", float64(int(fh*0.66)), fg, 0.2, 0)
	small := textImg("COMBAT LEAGUE", float64(int(fh*0.2)), fg, 0, max(1, int(fh*0.05)))
	aw := int(fh * 0.78)
	im := paint.NewTransparent(aw+max(big.W, small.W)+4, h)
	axeIcon(im, float64(aw)*0.45, fh*0.5, fh*0.92, axe)
	by := int(fh * 0.04)
	im.AlphaComposite(big, aw, max(0, by))
	im.AlphaComposite(small, aw+int(fh*0.1), min(h-small.H, by+big.H+1))
	return im
}

// pasteFit scales src to fit the box (keeping its shape) and composites it centred.
func pasteFit(dst, src *paint.Canvas, x0, y0, x1, y1 float64) {
	k := math.Min((x1-x0)/float64(src.W), (y1-y0)/float64(src.H))
	s := src.ResizeLanczos(max(1, int(float64(src.W)*k)), max(1, int(float64(src.H)*k)))
	dst.AlphaComposite(s, int(x0+(x1-x0-float64(s.W))/2), int(y0+(y1-y0-float64(s.H))/2))
}

func slashes(d *paint.Canvas, x, y float64, n int, w, h float64, c col, gap float64) {
	if gap == 0 {
		gap = w * 0.6
	}
	for i := range n {
		xi := x + float64(i)*(w+gap)
		d.Polygon([]float64{xi + h*0.5, y, xi + h*0.5 + w, y, xi + w, y + h, xi, y + h}, c)
	}
}

// led darkens alternate columns and rows like an LED board.
func led(im *paint.Canvas, strength float64) *paint.Canvas {
	a := im.Arr()
	a.MulCols(0, 2, strength)
	a.MulRows(0, 2, (strength+1)/2)
	return a.Canvas()
}

// ------------------------------------------------------------------ building surfaces

func panelsArr(w, h int, c0, c1 col, cell int) *paint.Arr {
	a := paint.Colorize(tx.Noise(w, h, 2).Mul(0.22).Add(tx.Rand(w, h).Mul(0.05)), c0, c1)
	for k := 0; k < h; k += cell {
		a.MulRect(0, k, w, k+1, 0.55)
		a.MulRect(0, (k+1)%h, w, (k+1)%h+1, 1.12)
	}
	for k := 0; k < w; k += cell {
		a.MulRect(k, 0, k+1, h, 0.55)
		a.MulRect((k+1)%w, 0, (k+1)%w+1, h, 1.12)
	}
	for y := 0; y < h; y += cell {
		for x := 0; x < w; x += cell {
			for _, d := range [][2]int{{4, 4}, {cell - 5, 4}, {4, cell - 5}, {cell - 5, cell - 5}} {
				yy, xx := (y+d[1])%h, (x+d[0])%w
				p := a.Px(xx, yy)
				a.SetPx(xx, yy, col{min(p[0]*1.6, 255), min(p[1]*1.6, 255), min(p[2]*1.6, 255)})
			}
		}
	}
	return a
}

func panels(w, h int, c0, c1 col, cell int) tx.Draw {
	return func() *paint.Canvas { return panelsArr(w, h, c0, c1, cell).Canvas() }
}

func noiseTex(w int, blur, k float64, c0, c1 col) tx.Draw {
	return func() *paint.Canvas { return paint.Colorize(tx.Noise(w, w, blur).Mul(k), c0, c1).Canvas() }
}

// roof is the dark ceiling deck with a beam grid and strip-light fixtures (drawn for scale 4).
func roof() *paint.Canvas {
	a := paint.Colorize(tx.Noise(128, 128, 2).Mul(0.25), col{30, 32, 37}, col{42, 44, 50})
	for k := 0; k < 128; k += 64 {
		a.SetRect(0, k, 128, k+3, col{62, 64, 70})
		a.SetRect(k, 0, k+3, 128, col{62, 64, 70})
	}
	a.MulRows(0, 16, 0.8)
	im := a.Canvas()
	for _, cy := range []float64{32, 96} {
		for _, cx := range []float64{32, 96} {
			im.Rect(cx-14, cy-3, cx+14, cy+3, col{70, 70, 74})
			im.Rect(cx-12, cy-2, cx+12, cy+2, col{235, 235, 225})
		}
	}
	return im
}

func truss() *paint.Canvas {
	im := paint.NewCanvas(32, 32, col{52, 54, 60})
	im.Rect(0, 0, 31, 3, col{120, 122, 130})
	im.Rect(0, 28, 31, 31, col{120, 122, 130})
	im.Line([]float64{0, 3, 31, 28}, col{105, 107, 115}, 2)
	im.Line([]float64{0, 28, 31, 3}, col{105, 107, 115}, 2)
	return im
}

// suites is the glass-fronted suite level: lit rooms with seated silhouettes, red trim under the sill.
func suites() *paint.Canvas {
	r := seeded(14)
	im := paint.NewCanvas(256, 64, col{86, 90, 98})
	im.Rect(0, 0, 255, 7, col{40, 42, 48})
	people := []col{{60, 40, 38}, {40, 40, 52}, {70, 55, 45}, {30, 30, 34}}
	for i := range 8 {
		x0 := float64(i*32 + 3)
		x1 := x0 + 27
		for y := 10; y < 53; y++ {
			t := float64(y-10) / 43
			im.Line([]float64{x0, float64(y), x1, float64(y)}, col{float64(int(150 - 60*t)), float64(int(118 - 48*t)), float64(int(84 - 30*t))}, 1)
		}
		im.Rect(x0, 11, x1, 13, col{250, 240, 210}) // ceiling light
		for x := x0 + 1; x < x1-4; {
			w := float64(5 + r.IntN(3))
			hy := float64(30 + r.IntN(6))
			c := people[r.IntN(4)]
			skin := col{90, 65, 50}
			if r.Float64() < 0.7 {
				skin = col{150, 110, 85}
			}
			im.Ellipse(x, hy, x+4, hy+5, skin)
			im.RoundedRect(x-1, hy+5, x+w-1, 53, 2, c)
			x += w
		}
		im.Line([]float64{x0 + 4, 52, x0 + 14, 14}, col{170, 160, 150}, 1) // reflection
	}
	im.Rect(0, 53, 255, 59, col{110, 114, 122})
	im.Rect(0, 60, 255, 63, red)
	return im
}

// ------------------------------------------------------------------ cage and canvas

func fence() *paint.Canvas {
	a := paint.Full(64, 64, col{0, 0, 255})
	for y := range 64 {
		for x := range 64 {
			if mapkit.PyMod(x+y, 16) < 2 || mapkit.PyMod(x-y, 16) < 2 {
				s := 46 + 26*float64(mapkit.PyMod(x+y, 16))/16
				a.SetPx(x, y, col{s, s, s + 4})
			}
		}
	}
	return a.Canvas()
}

func cagepost() *paint.Canvas {
	im := paint.NewCanvas(32, 128, ink)
	im.Rect(0, 0, 31, 5, red)
	axeIcon(im, 16, 26, 22, white)
	im.Text(16, 62, "B", white, tx.Bold(22), "mm")
	im.Text(16, 92, "A", white, tx.Bold(22), "mm")
	return im
}

func canvas() *paint.Canvas {
	const w = 256
	im := paint.Colorize(tx.Noise(w, w, 3).Mul(0.10).Plus(0.9), col{214, 214, 218}, col{238, 238, 240}).Canvas()
	var pts []float64
	for i := range 8 {
		a := (22.5 + 45*float64(i)) * math.Pi / 180
		pts = append(pts, float64(int(128+120*math.Cos(a))), float64(int(128+120*math.Sin(a))))
	}
	im.PolygonOutline(pts, col{170, 170, 176}, 1)
	pasteFit(im, logo(80, red, col{38, 38, 44}), 24, 92, 232, 164)
	im.Text(128, 200, "FIGHT NIGHT", col{120, 120, 128}, tx.Bold(11), "mm")
	return im
}

func apronside() *paint.Canvas {
	im := paint.NewCanvas(256, 64, ink)
	im.Rect(0, 0, 255, 5, red)
	axeIcon(im, 30, 36, 40, white)
	pasteFit(im, logo(48, red, white), 60, 16, 210, 56)
	slashes(im, 214, 26, 3, 6, 18, red, 0)
	return im
}

func axeLogo() *paint.Canvas {
	im := paint.NewCanvas(64, 64, ink)
	axeIcon(im, 32, 32, 48, red)
	return im
}

// ------------------------------------------------------------------ crowd

func personRow(width int, seed uint64, phones, signs float64) *paint.Canvas {
	r := seeded(seed)
	im := paint.NewCanvas(width, 64, col{36, 36, 42})
	shirts := []col{{120, 45, 45}, {72, 62, 52}, {150, 120, 90}, {58, 68, 92}, {92, 92, 98}, {40, 40, 46},
		{130, 95, 70}, {100, 40, 42}, {168, 150, 122}, {56, 74, 60}, {30, 50, 90}, {180, 180, 182}}
	skins := []col{{232, 196, 166}, {210, 160, 120}, {172, 118, 84}, {122, 82, 58}, {92, 62, 44}}
	hairs := []col{{30, 20, 15}, {70, 45, 25}, {180, 150, 95}, {15, 15, 15}, {110, 110, 110}}
	for x := 1; x < width-12; {
		pw := 15 + r.IntN(5)
		cx := float64(x + pw/2)
		top := float64(12 + r.IntN(8))
		sc, sk := shirts[r.IntN(len(shirts))], skins[r.IntN(len(skins))]
		fx := float64(x)
		im.RoundedRect(fx, top+12, fx+float64(pw)-1, 64, 5, sc)
		im.Ellipse(cx-5, top, cx+5, top+13, sk)
		im.Chord(cx-5, top-1, cx+5, top+8, 180, 360, hairs[r.IntN(len(hairs))])
		switch q := r.Float64(); {
		case q < phones:
			im.Rect(cx+5, top-6, cx+8, top, col{235, 240, 255})
		case q < phones+signs:
			im.Line([]float64{cx, top + 14, cx, top - 6}, col{170, 170, 170}, 1)
			im.Rect(cx-9, top-16, cx+9, top-6, ink)
			im.Line([]float64{cx - 6, top - 11, cx + 6, top - 11}, red, 2)
		case q < phones+signs+0.1:
			im.Line([]float64{fx + 2, top + 16, fx - 2, top - 4}, sk, 3)
			im.Line([]float64{fx + float64(pw) - 2, top + 16, fx + float64(pw) + 2, top - 4}, sk, 3)
		}
		x += pw
	}
	return im
}

func crowd(seed uint64, phones, signs float64, w, h int) tx.Draw {
	return func() *paint.Canvas { return personRow(256, seed, phones, signs).ResizeLanczos(w, h) }
}

func crowdtop() *paint.Canvas {
	r := seeded(9)
	a := paint.Full(64, 64, col{36, 36, 42})
	cols := []col{{30, 20, 15}, {70, 45, 25}, {180, 150, 95}, {15, 15, 15}, {160, 115, 80}, {110, 110, 110}}
	for y := 0; y < 64; y += 3 {
		for x := r.IntN(3); x < 64; x += 3 {
			if r.Float64() < 0.8 {
				a.SetRect(x, y, x+2, y+2, cols[r.IntN(len(cols))])
			}
		}
	}
	return a.Canvas()
}

// ------------------------------------------------------------------ boards and signs

func ribbon(kind int) tx.Draw {
	return func() *paint.Canvas {
		im := paint.NewCanvas(256, 32, ink)
		im.Rect(0, 0, 255, 1, red)
		im.Rect(0, 30, 255, 31, red)
		switch kind {
		case 1:
			pasteFit(im, logo(40, red, white), 6, 4, 120, 28)
			im.Line([]float64{128, 8, 128, 24}, col{120, 120, 128}, 1)
			im.Ellipse(138, 13, 144, 19, red)
			im.Text(150, 16, "LIVE  FIGHT NIGHT", white, tx.Bold(11), "lm")
		case 2:
			slashes(im, 8, 9, 3, 5, 14, red, 0)
			slashes(im, 222, 9, 3, 5, 14, red, 0)
			pasteFit(im, logo(44, red, white), 48, 3, 208, 29)
		default:
			for _, x := range []float64{14, 242} {
				axeIcon(im, x, 16, 20, red)
			}
			im.Text(128, 16, "CHAMPIONSHIP SERIES", white, tx.Bold(13), "mm")
		}
		return led(im, 0.85)
	}
}

func scoreboard(variant int) tx.Draw {
	return func() *paint.Canvas {
		im := paint.NewCanvas(256, 160, col{16, 16, 20})
		im.RectOutline(0, 0, 255, 159, red, 4)
		grey := col{170, 170, 178}
		im.Text(12, 16, "BACL  /  LIVE", grey, tx.Mono(10), "lm")
		im.Text(244, 16, "FIGHT NIGHT", grey, tx.Mono(10), "rm")
		if variant == 1 {
			pasteFit(im, logo(80, red, white), 20, 30, 236, 104)
		} else {
			im.Text(128, 58, "MORROW  vs  REYES", white, tx.Bold(20), "mm")
			im.Rect(40, 78, 127, 84, redD)
			im.Rect(129, 78, 216, 84, col{20, 40, 110})
			im.Text(128, 98, "MIDDLEWEIGHT  CHAMPIONSHIP", col{200, 200, 206}, tx.Mono(10), "mm")
		}
		const y0 = 114.0
		for _, c := range []struct {
			x0, x1 float64
			t      string
		}{{10, 128, "MAIN EVENT"}, {132, 178, "01"}, {182, 246, "05:00"}} {
			im.Rect(c.x0, y0, c.x1, y0+34, col{34, 34, 40})
			fg := white
			if c.t == "05:00" {
				fg = red
			}
			im.Text((c.x0+c.x1)/2, y0+17, c.t, fg, tx.Bold(16), "mm")
		}
		return led(im, 0.9)
	}
}

// walkout is the red-corner entrance wall, 512x144 across ~1420 x 400 units; gate opening at the bottom centre.
func walkout() *paint.Canvas {
	im := panelsArr(512, 144, col{66, 70, 78}, col{88, 92, 100}, 48).Canvas()
	const gx0, gx1, gy = 238.0, 274.0, 104.0 // gate opening (cut in the wall itself)
	lit := col{255, 45, 55}
	im.Rect(gx0-14, gy-12, gx1+14, 143, col{112, 116, 124})
	im.Rect(gx0, gy, gx1, 143, col{10, 10, 12})
	for _, x := range []float64{gx0 - 20, gx1 + 14} {
		im.Rect(x, gy-20, x+6, 143, lit)
	}
	im.Rect(gx0-20, gy-20, gx1+20, gy-15, lit)
	im.RectFillOutline(186, 52, 326, 80, ink, red, 2)
	pasteFit(im, logo(48, red, white), 192, 55, 320, 77)
	for _, bx := range []float64{24, 348} {
		im.Rect(bx, 20, bx+140, 100, ink)
		im.Rect(bx, 94, bx+140, 100, red)
		pasteFit(im, logo(64, red, white), bx+8, 30, bx+132, 86)
	}
	for x := 8.0; x < 512; x += 32 {
		c := col{10, 10, 12}
		if math.Abs(x+8-256) > 40 {
			c = lit
		}
		im.Rect(x, 136, x+16, 139, c)
	}
	return im
}

func walkway(c col, flip bool) tx.Draw {
	return func() *paint.Canvas {
		im := paint.Colorize(tx.Noise(128, 128, 2).Mul(0.3), col{12, 12, 14}, col{28, 28, 32}).Canvas()
		for _, x := range []float64{0, 122} {
			im.Rect(x, 0, x+5, 127, c)
		}
		for y := 4.0; y < 128; y += 16 {
			im.Rect(10, y, 12, y+6, col{230, 230, 230})
			im.Rect(115, y, 117, y+6, col{230, 230, 230})
		}
		for _, cy := range []float64{40, 104} {
			im.Line([]float64{36, cy + 14, 64, cy - 6, 92, cy + 14}, c, 7)
		}
		if flip {
			return im.FlipV()
		}
		return im
	}
}

func cornerSign(word string, base, accent col, gate string) tx.Draw {
	return func() *paint.Canvas {
		im := paint.NewCanvas(256, 128, col{22, 22, 28})
		im.Polygon([]float64{0, 0, 150, 0, 104, 127, 0, 127}, base)
		im.Rect(0, 0, 255, 5, accent)
		im.Rect(0, 122, 255, 127, accent)
		grey := col{220, 220, 226}
		im.Text(14, 20, "BACL  /  ATHLETE ENTRANCE", grey, tx.Mono(9), "lm")
		im.Text(242, 20, gate, grey, tx.Mono(9), "rm")
		big := textImg(word, 40, white, 0, 0)
		if big.W > 228 {
			big = big.ResizeLanczos(228, big.H)
		}
		im.AlphaComposite(big, 12, 34)
		im.Rect(14, 84, 70, 86, accent)
		im.Text(14, 100, "FIGHTER WALKOUT", grey, tx.Mono(10), "lm")
		slashes(im, 176, 92, 4, 8, 18, accent, 5)
		return im
	}
}

func lockers(c0, c1 col, prefix string) tx.Draw {
	return func() *paint.Canvas {
		a := paint.Colorize(tx.Noise(64, 128, 1).Mul(0.3).Plus(0.7), c0, c1)
		a.MulCols(0, 32, 0.35)
		a.MulCols(1, 32, 0.6)
		for x := 0; x < 64; x += 32 {
			for _, y := range []int{10, 15, 20, 25} {
				a.MulRect(x+8, y, x+24, y+2, 0.45)
			}
			a.SetRect(x+26, 64, x+28, 72, col{200, 200, 205})
		}
		im := a.Canvas()
		for i, x := range []float64{0, 32} {
			im.Rect(x+7, 34, x+25, 42, col{225, 225, 228})
			im.Text(x+16, 38, prefix+"-0"+strconv.Itoa(i+1), col{20, 20, 24}, tx.Mono(7), "mm")
		}
		return im
	}
}

func broadcast() *paint.Canvas {
	im := paint.NewCanvas(64, 48, col{10, 10, 14})
	im.Rect(0, 0, 63, 34, col{40, 40, 46})
	var pts []float64
	for i := range 8 {
		a := (22.5 + 45*float64(i)) * math.Pi / 180
		pts = append(pts, 32+22*math.Cos(a), 18+13*math.Sin(a))
	}
	im.Polygon(pts, col{210, 210, 215})
	im.PolygonOutline(pts, col{20, 20, 20}, 1)
	im.Rect(25, 12, 28, 22, col{170, 20, 20})
	im.Rect(36, 13, 39, 23, col{20, 40, 170})
	im.Rect(0, 36, 63, 47, col{150, 10, 20})
	im.Text(32, 41, "LIVE  R1 05:00", col{255, 255, 255}, tx.Mono(7), "mm")
	return im
}

func press() *paint.Canvas {
	im := paint.NewCanvas(128, 32, ink)
	im.Rect(0, 0, 127, 2, red)
	axeIcon(im, 14, 17, 20, red)
	im.Text(72, 17, "BACL  MEDIA", col{210, 210, 218}, tx.Bold(12), "mm")
	return im
}

func lightbank() *paint.Canvas {
	im := paint.NewCanvas(64, 32, col{20, 20, 24})
	for x := 4.0; x < 64; x += 15 {
		for _, y := range []float64{3, 17} {
			im.Ellipse(x, y, x+11, y+11, col{255, 250, 230})
		}
	}
	return im
}

// textureList is every texture in the map's WAD, in order.
func textureList() []mapkit.Texture {
	var t []mapkit.Texture
	add := func(name string, d tx.Draw) { t = append(t, mapkit.Texture{Name: name, Draw: d}) }
	for _, k := range []string{texConcrete, texMetal, "TRIM", texLamp, texTable, texTvblack, texCeiltile, "STEP", "AAATRIGGER"} {
		add(k, tx.Base(k))
	}
	add(texStadwall, panels(128, 128, col{66, 70, 78}, col{88, 92, 100}, 32))
	add("TUNNELWALL", panels(64, 64, col{52, 55, 62}, col{68, 71, 78}, 32))
	add(texArenafloor, func() *paint.Canvas {
		return paint.Colorize(tx.Rand(64, 64).Mul(0.35).Add(tx.Noise(64, 64, 2).Mul(0.65)), col{34, 36, 40}, col{56, 58, 63}).Canvas()
	})
	add(texStands, func() *paint.Canvas {
		return paint.Colorize(tx.Rand(64, 64).Mul(0.4).Add(tx.Noise(64, 64, 1).Mul(0.6)), col{76, 78, 84}, col{100, 102, 108}).Canvas()
	})
	add(texSeats, noiseTex(64, 1, 0.3, col{46, 48, 54}, col{64, 66, 72}))
	add(texStage, func() *paint.Canvas {
		return paint.Colorize(tx.Noise(64, 64, 3).Mul(0.3).Add(tx.Rand(64, 64).Mul(0.1)), col{12, 12, 14}, col{34, 34, 38}).Canvas()
	})
	add(texApron, noiseTex(64, 2, 0.3, col{10, 10, 12}, col{30, 30, 34}))
	add(texCagepad, noiseTex(32, 1, 0.4, col{10, 10, 12}, col{36, 36, 40}))
	add(texCouch, noiseTex(32, 1, 0.4, col{26, 26, 30}, col{44, 44, 50}))
	add(texCamera, noiseTex(32, 1, 0.4, col{20, 20, 22}, col{60, 60, 66}))
	add(texRedstrip, tx.Solid(16, 16, col{255, 45, 55}))
	add(texBluestrip, tx.Solid(16, 16, col{60, 130, 255}))
	add(texRoof, roof)
	add(texTruss, truss)
	add(texSuites, suites)
	add("{CAGEFENCE", fence)
	add("CAGEPOST", cagepost)
	add("CANVAS", canvas)
	add("APRONSIDE", apronside)
	add(texAxelogo, axeLogo)
	rows := []struct{ phones, signs float64 }{{0.04, 0.02}, {0.08, 0.02}, {0.04, 0.06}}
	for i, r := range rows {
		add(fmt.Sprintf("CROWD%d", i+1), crowd(uint64(i+1), r.phones, r.signs, 128, 32))
	}
	// The stands' crowd (CROWDF*): the same rows at half resolution, drawn at twice the scale, so they cover the same
	// area with a quarter of the light samples (lighting is most of the compile). Only the floor rows next to the
	// octagon, which players see up close, use the full-resolution CROWD*.
	for i, r := range rows {
		full := crowd(uint64(i+1), r.phones, r.signs, 128, 32)
		add(fmt.Sprintf("CROWDF%d", i+1), func() *paint.Canvas { return full().ResizeLanczos(64, 16) })
	}
	add(texCrowdtop, crowdtop)
	add(texRibbon1, ribbon(1))
	add(texRibbon2, ribbon(2))
	add(texRibbon3, ribbon(3))
	add(texJumbo1, scoreboard(1))
	add(texJumbo2, scoreboard(2))
	add(texWalkout, walkout)
	add("WALKWAY_R", walkway(red, false))
	add("WALKWAY_B", walkway(blue, true))
	add(texCornerR, cornerSign("RED CORNER", col{110, 20, 28}, red, "GATE 01"))
	add(texCornerB, cornerSign("BLUE CORNER", col{20, 40, 110}, blue, "GATE 02"))
	add(texLockerR, lockers(col{88, 30, 28}, col{118, 44, 40}, "A"))
	add(texLockerB, lockers(col{38, 48, 68}, col{54, 66, 90}, "B"))
	add(texBroadcast, broadcast)
	add(texPress, press)
	add(texLightbank, lightbank)
	// a smooth, untextured glow (any repeat pattern would show as stripes on a big cone)
	add(texLightbeam, tx.Solid(16, 16, col{255, 245, 225}))
	return t
}

var emissive = []mapkit.Glow{{Name: texRibbon1, Glow: 200}, {Name: texRibbon2, Glow: 200}, {Name: texRibbon3, Glow: 200},
	{Name: texJumbo1, Glow: 230}, {Name: texJumbo2, Glow: 230}, {Name: texWalkout, Glow: 90}, {Name: texBroadcast, Glow: 220},
	{Name: texLightbank, Glow: 255}, {Name: texLamp, Glow: 255}, {Name: texCornerR, Glow: 170}, {Name: texCornerB, Glow: 170},
	{Name: texRedstrip, Glow: 255}, {Name: texBluestrip, Glow: 255}, {Name: texSuites, Glow: 105}, {Name: texAxelogo, Glow: 200}}
