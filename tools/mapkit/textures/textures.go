// Package textures is the shared texture library (originally the Justin Con set): noise and colour helpers,
// the fonts, and base textures maps reuse by name. Every texture is drawn by code; nothing is copied.
package textures

import (
	"fmt"
	"math/rand/v2"
	"sync"

	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/paint"
)

// RNG feeds the base textures (and the maps that share it), seeded so builds are repeatable.
var RNG = rand.New(rand.NewPCG(42, 0)) //nolint:gosec // texture noise, not security

// Draw draws one texture.
type Draw = func() *paint.Canvas

// Noise is blurred noise stretched to 0..1 from RNG.
func Noise(w, h int, blur float64) *paint.Field { return paint.Noise(RNG, w, h, blur, 1) }

// Rand is uniform noise in [0, 1) from RNG.
func Rand(w, h int) *paint.Field { return paint.Random(RNG, w, h) }

// IntN is a random integer in [0, n) from RNG.
func IntN(n int) int { return RNG.IntN(n) }

var fonts = sync.OnceValues(func() (map[string]*paint.Font, error) {
	out := map[string]*paint.Font{}
	for key, file := range map[string]string{"bold": "Poppins-Bold.ttf", "mono": "DejaVuSansMono.ttf"} {
		p, err := mapkit.FontPath(file)
		if err != nil {
			return nil, err
		}
		if out[key], err = paint.LoadFont(p); err != nil {
			return nil, fmt.Errorf("font %s: %w", file, err)
		}
	}
	return out, nil
})

func face(key string, size float64) *paint.Face {
	f, err := fonts()
	if err != nil {
		panic(err) // the fonts ship with the repo; without them no texture with text can be drawn
	}
	return f[key].Face(size)
}

// Bold is Poppins Bold at size pixels.
func Bold(size float64) *paint.Face { return face("bold", size) }

// Mono is DejaVu Sans Mono at size pixels.
func Mono(size float64) *paint.Face { return face("mono", size) }

// Slabs is concrete slabs with dark joints every step pixels.
func Slabs(c0, c1 paint.Col, step int) Draw {
	return func() *paint.Canvas {
		a := paint.Colorize(Noise(128, 128, 1.2).Mul(0.5).Add(Rand(128, 128).Mul(0.5)), c0, c1)
		a.MulRows(0, step, 0.55)
		a.MulCols(0, step, 0.55)
		return a.Canvas()
	}
}

// Siding is horizontal lap siding.
func Siding(c0, c1 paint.Col) Draw {
	return func() *paint.Canvas {
		a := paint.Colorize(Noise(128, 128, 1.5).Mul(0.3).Plus(0.7), c0, c1)
		for y := 0; y < 128; y += 16 {
			a.MulRect(0, y, 128, y+2, 0.55)
			a.MulRect(0, y+2, 128, y+4, 1.08)
		}
		return a.Canvas()
	}
}

func brick() *paint.Canvas {
	a := paint.Colorize(Noise(128, 128, 0.7), paint.Col{110, 45, 35}, paint.Col{165, 75, 55})
	mortar := paint.Col{150, 145, 135}
	for r, y := 0, 0; y < 128; r, y = r+1, y+16 {
		a.SetRect(0, y, 128, y+3, mortar)
		off := 16 * (r % 2)
		for x := off; x < 128; x += 32 {
			a.SetRect(x, y, x+3, y+16, mortar)
		}
	}
	return a.Canvas()
}

func wallpaper() *paint.Canvas {
	a := paint.Colorize(Noise(128, 128, 1).Mul(0.2).Plus(0.8), paint.Col{170, 150, 115}, paint.Col{215, 195, 160})
	for x := 0; x < 128; x += 32 {
		a.MulRect(x, 0, x+3, 128, 0.85)
		a.MulRect(x+14, 0, x+16, 128, 1.05)
	}
	return a.Canvas()
}

// Planks is wooden boards gap pixels wide, size x size, turned upright if vertical.
func Planks(c0, c1 paint.Col, vertical bool, size, gap int) Draw {
	return func() *paint.Canvas {
		n := paint.ResizeField(Noise(size, size/4, 0.6), size, size)
		a := paint.Colorize(n.Mul(0.6).Add(Rand(size, size).Mul(0.15)), c0, c1)
		for y := 0; y < size; y += gap {
			a.MulRect(0, y, size, y+2, 0.45)
			j := IntN(size)
			a.MulRect(j, y, j+2, y+gap, 0.5)
			a.MulRect(0, y, size, y+gap, 0.85+0.3*RNG.Float64())
		}
		im := a.Canvas()
		if vertical {
			return im.Rotate90()
		}
		return im
	}
}

// Flat is a texture of plain noise between two colours: noise(w, w, blur) * k + base.
func Flat(w int, blur, k, base float64, c0, c1 paint.Col) Draw {
	return func() *paint.Canvas {
		return paint.Colorize(Noise(w, w, blur).Mul(k).Plus(base), c0, c1).Canvas()
	}
}

// Solid is a texture of one colour.
func Solid(w, h int, c paint.Col) Draw {
	return func() *paint.Canvas { return paint.Full(w, h, c).Canvas() }
}

func ceiltile() *paint.Canvas {
	a := paint.Colorize(Rand(128, 128).Mul(0.3).Plus(0.7), paint.Col{175, 172, 165}, paint.Col{215, 212, 205})
	for range 300 {
		x, y := IntN(128), IntN(128)
		a.MulRect(x, y, x+1, y+1, 0.7)
	}
	a.SetRows(0, 64, paint.Col{120, 120, 120})
	a.SetCols(0, 64, paint.Col{120, 120, 120})
	return a.Canvas()
}

func fridge() *paint.Canvas {
	a := paint.Colorize(Noise(64, 64, 2).Mul(0.1).Plus(0.9), paint.Col{200, 205, 205}, paint.Col{235, 238, 238})
	a.MulRect(0, 20, 64, 22, 0.6)
	a.SetRect(54, 28, 57, 52, paint.Col{120, 120, 125})
	return a.Canvas()
}

func carpaint(c paint.Col) Draw {
	return func() *paint.Canvas {
		a := paint.Colorize(Noise(64, 64, 3).Mul(0.4).Plus(0.6), c.Scale(0.5), c)
		a.MulRect(0, 30, 64, 32, 0.6)
		return a.Canvas()
	}
}

func crt() *paint.Canvas {
	a := paint.Colorize(Noise(64, 64, 2).Mul(0.2).Plus(0.8), paint.Col{170, 160, 135}, paint.Col{215, 205, 180})
	a.MulRect(0, 54, 64, 58, 0.7)
	return a.Canvas()
}

func serverrack() *paint.Canvas {
	im := paint.NewCanvas(64, 128, paint.Col{15, 15, 18})
	leds := []paint.Col{{0, 255, 80}, {0, 160, 255}, {255, 170, 0}}
	for y := 4; y < 128; y += 12 {
		im.Rect(4, float64(y), 59, float64(y+9), paint.Col{35, 35, 40})
		for x := 8; x < 30; x += 5 {
			c := leds[IntN(3)]
			im.Set(x, y+4, c)
			im.Set(x+1, y+4, c)
		}
	}
	return im
}

var lib = map[string]Draw{
	"CONCRETE":  Slabs(paint.Col{95, 95, 92}, paint.Col{150, 148, 142}, 128),
	"BRICK":     brick,
	"TRIM":      Flat(64, 1, 0.25, 0.75, paint.Col{200, 200, 195}, paint.Col{240, 240, 235}),
	"WALLPAPER": wallpaper,
	"WOODFLOOR": Planks(paint.Col{95, 60, 30}, paint.Col{165, 115, 65}, false, 128, 16),
	"BASEWALL":  Planks(paint.Col{70, 45, 25}, paint.Col{135, 90, 50}, true, 128, 32),
	"WOODFENCE": Planks(paint.Col{90, 70, 50}, paint.Col{150, 120, 85}, true, 128, 16),
	"STEP":      Planks(paint.Col{80, 55, 30}, paint.Col{140, 100, 60}, false, 128, 16),
	"CEILING":   Flat(64, 0.8, 0.15, 0.85, paint.Col{195, 192, 185}, paint.Col{230, 228, 222}),
	"CEILTILE":  ceiltile,
	"CARPET": func() *paint.Canvas {
		return paint.Colorize(Rand(64, 64).Mul(0.7).Add(Noise(64, 64, 2).Mul(0.3)), paint.Col{30, 35, 55}, paint.Col{75, 80, 105}).Canvas()
	},
	"TABLE": Flat(64, 2, 0.2, 0.8, paint.Col{170, 170, 165}, paint.Col{205, 205, 200}),
	"METAL": Flat(64, 0.7, 0.5, 0.5, paint.Col{45, 47, 52}, paint.Col{110, 112, 120}),
	"TIRE": func() *paint.Canvas {
		return paint.Colorize(Rand(32, 32), paint.Col{12, 12, 12}, paint.Col{35, 35, 35}).Canvas()
	},
	"COUCH": func() *paint.Canvas {
		return paint.Colorize(Rand(64, 64).Mul(0.4).Add(Noise(64, 64, 1).Mul(0.6)), paint.Col{70, 45, 35}, paint.Col{120, 80, 60}).Canvas()
	},
	"COUNTER": func() *paint.Canvas {
		return paint.Colorize(Rand(64, 64).Pow(3), paint.Col{40, 40, 45}, paint.Col{190, 185, 180}).Canvas()
	},
	"TVBLACK":    Flat(32, 1, 0.3, 0, paint.Col{8, 8, 10}, paint.Col{40, 40, 45}),
	"GLASS":      Flat(64, 4, 0.4, 0.6, paint.Col{120, 150, 170}, paint.Col{190, 215, 230}),
	"CARGLASS":   Flat(64, 4, 1, 0, paint.Col{15, 20, 30}, paint.Col{60, 75, 95}),
	"FRIDGE":     fridge,
	"CARRED":     carpaint(paint.Col{170, 25, 25}),
	"CARBLUE":    carpaint(paint.Col{30, 70, 170}),
	"CARSILVER":  carpaint(paint.Col{170, 175, 180}),
	"LAMP":       Flat(32, 2, 0.2, 0.8, paint.Col{230, 200, 140}, paint.Col{255, 240, 200}),
	"CRT":        crt,
	"sky":        Solid(64, 64, paint.Col{10, 12, 30}),
	"AAATRIGGER": Solid(16, 16, paint.Col{255, 140, 0}),
	"SERVERRACK": serverrack,
}

// Base is the shared texture called name (it panics on an unknown name: a typo in a map is a bug).
func Base(name string) Draw {
	d, ok := lib[name]
	if !ok {
		panic("textures: no base texture " + name)
	}
	return d
}

// Emissive lists the base textures that glow (lit through mapkit.Texlights).
var Emissive = []mapkit.Glow{{Name: "LAMP", Glow: 255}, {Name: "SERVERRACK", Glow: 170}}
