package main

// Texture recipes for casaroble's zones. Each zone registers the ones it uses under its own prefixed names.

import (
	"math"
	"math/rand/v2"

	"halflifemaps/tools/mapkit/paint"
	tx "halflifemaps/tools/mapkit/textures"
)

type col = paint.Col

var rng = rand.New(rand.NewPCG(1922, 0)) //nolint:gosec // texture noise, not security

func rnd(w, h int) *paint.Field { return paint.Random(rng, w, h) }

func stucco(c0, c1 col) tx.Draw {
	return func() *paint.Canvas {
		return paint.Colorize(tx.Noise(128, 128, 0.6).Mul(0.45).Add(tx.Noise(128, 128, 3).Mul(0.55)), c0, c1).Canvas()
	}
}

var stuccoHouse = stucco(col{190, 168, 134}, col{224, 206, 176})

func plaster() *paint.Canvas {
	return paint.Colorize(tx.Noise(64, 64, 1.5).Mul(0.3).Plus(0.7), col{214, 208, 196}, col{238, 234, 226}).Canvas()
}

// rooftile is red clay barrel tiles seen side-on: rounded columns, staggered courses.
func rooftile() *paint.Canvas {
	n := tx.Noise(64, 64, 1)
	a := paint.NewField(64, 64)
	for y := range 64 {
		for x := range 64 {
			c := 0.55 + 0.45*math.Sin(float64(x%16)/16*math.Pi)
			a.V[y*64+x] = c * (0.85 + 0.15*n.At(x, y))
			if y%16 < 2 {
				a.V[y*64+x] *= 0.45
			}
		}
	}
	return paint.Colorize(a, col{95, 38, 22}, col{196, 92, 58}).Canvas()
}

func terracotta() *paint.Canvas {
	a := paint.Colorize(tx.Noise(64, 64, 1.2).Mul(0.6).Add(rnd(64, 64).Mul(0.2)), col{150, 72, 44}, col{198, 110, 70})
	a.SetRows(0, 32, col{170, 160, 145})
	a.SetCols(0, 32, col{170, 160, 145})
	return a.Canvas()
}

func copper() *paint.Canvas {
	a := paint.Colorize(tx.Noise(32, 32, 1.5), col{150, 82, 40}, col{210, 130, 75})
	v := tx.Noise(32, 32, 0.8)
	for y := range 32 {
		for x := range 32 {
			if v.At(x, y) > 0.72 { // verdigris patches
				a.SetPx(x, y, col{95, 160, 135})
			}
		}
	}
	return a.Canvas()
}

func mix(rk, blur, nk float64, c0, c1 col) tx.Draw {
	return func() *paint.Canvas {
		return paint.Colorize(rnd(128, 128).Mul(rk).Add(tx.Noise(128, 128, blur).Mul(nk)), c0, c1).Canvas()
	}
}

var (
	gravel  = mix(0.7, 2, 0.3, col{118, 106, 88}, col{205, 190, 160})
	lawn    = mix(0.55, 4, 0.45, col{52, 88, 34}, col{108, 146, 62})
	asphalt = mix(0.6, 2, 0.4, col{38, 38, 40}, col{82, 82, 86})
	tables  = tx.Planks(col{70, 42, 24}, col{120, 80, 48}, false, 128, 16)
)

func bark() *paint.Canvas {
	n := paint.ResizeField(tx.Noise(16, 64, 0.8), 64, 64)
	return paint.Colorize(n.Mul(0.7).Add(rnd(64, 64).Mul(0.3)), col{48, 38, 28}, col{112, 92, 70}).Canvas()
}

func leaves() *paint.Canvas {
	a := tx.Noise(64, 64, 0.7).Map(func(v float64) float64 {
		if v > 0.35 {
			return 0.6
		}
		return 0
	})
	return paint.Colorize(a.Add(rnd(64, 64).Mul(0.4)), col{34, 52, 22}, col{92, 118, 50}).Canvas()
}

func agave() *paint.Canvas {
	return paint.Colorize(tx.Noise(32, 32, 0.6), col{70, 120, 110}, col{140, 180, 160}).Canvas()
}

func bedspread(c0, c1 col) tx.Draw {
	return func() *paint.Canvas {
		a := tx.Noise(64, 64, 2).Mul(0.3).Plus(0.7)
		for y := 0; y < 64; y += 16 {
			for x := range 64 {
				a.V[y*64+x] *= 0.8
			}
		}
		return paint.Colorize(a, c0, c1).Canvas()
	}
}

func cardboard() *paint.Canvas {
	a := paint.Colorize(tx.Noise(32, 32, 1).Mul(0.4).Plus(0.6), col{140, 102, 60}, col{190, 150, 100})
	a.SetRect(0, 14, 32, 18, col{200, 190, 150}) // tape
	return a.Canvas()
}

func membrane() *paint.Canvas {
	return paint.Colorize(tx.Noise(64, 64, 1).Mul(0.5).Plus(0.5), col{120, 112, 100}, col{160, 150, 135}).Canvas()
}

func whitepaint() *paint.Canvas {
	return paint.Colorize(tx.Noise(32, 32, 1).Mul(0.15).Plus(0.85), col{215, 215, 212}, col{245, 245, 242}).Canvas()
}

func hedge() *paint.Canvas {
	return paint.Colorize(rnd(64, 64).Mul(0.5).Add(tx.Noise(64, 64, 1.5).Mul(0.5)), col{28, 58, 26}, col{74, 112, 52}).Canvas()
}
