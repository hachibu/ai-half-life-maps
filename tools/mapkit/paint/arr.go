// Package paint is the small image toolkit the map textures are drawn with: float colour arrays for noise and
// shading (Arr, Field), an 8-bit canvas with aliased shapes and anti-aliased TrueType text (Canvas), and the
// filters the textures use (blur, resize, shear). It needs only the standard library.
package paint

import (
	"math"
	"math/rand/v2"
)

// Col is an RGB colour with 0..255 channels (floats, so shading can overshoot before clipping).
type Col [3]float64

// Scale multiplies every channel.
func (c Col) Scale(f float64) Col { return Col{c[0] * f, c[1] * f, c[2] * f} }

// Field is a grey float image, h rows of w values (numpy's (h, w) array).
type Field struct {
	W, H int
	V    []float64
}

// NewField is an all-zero field.
func NewField(w, h int) *Field { return &Field{w, h, make([]float64, w*h)} }

// Random is uniform noise in [0, 1) from r (numpy's rng.random((h, w))).
func Random(r *rand.Rand, w, h int) *Field {
	f := NewField(w, h)
	for i := range f.V {
		f.V[i] = r.Float64()
	}
	return f
}

// At is the value at (x, y).
func (f *Field) At(x, y int) float64 { return f.V[y*f.W+x] }

// Map returns g(v) for every value.
func (f *Field) Map(g func(float64) float64) *Field {
	o := NewField(f.W, f.H)
	for i, v := range f.V {
		o.V[i] = g(v)
	}
	return o
}

// Mul returns f * s.
func (f *Field) Mul(s float64) *Field { return f.Map(func(v float64) float64 { return v * s }) }

// Plus returns f + s.
func (f *Field) Plus(s float64) *Field { return f.Map(func(v float64) float64 { return v + s }) }

// Pow returns f ** p.
func (f *Field) Pow(p float64) *Field {
	return f.Map(func(v float64) float64 { return math.Pow(v, p) })
}

// Add returns f + g (same size).
func (f *Field) Add(g *Field) *Field {
	o := NewField(f.W, f.H)
	for i := range f.V {
		o.V[i] = f.V[i] + g.V[i]
	}
	return o
}

// Arr is a float RGB image, h rows of w pixels (numpy's (h, w, 3) array). Slicing helpers follow numpy's
// rules: ranges clip to the image.
type Arr struct {
	W, H int
	P    []float64
}

// NewArr is an all-black array.
func NewArr(w, h int) *Arr { return &Arr{w, h, make([]float64, w*h*3)} }

// Full is an array of one colour.
func Full(w, h int, c Col) *Arr {
	a := NewArr(w, h)
	a.SetRect(0, 0, w, h, c)
	return a
}

// Colorize maps a field to colours: c0 + (c1 - c0) * v, clipped to 0..255.
func Colorize(f *Field, c0, c1 Col) *Arr {
	a := NewArr(f.W, f.H)
	for i, v := range f.V {
		for c := range 3 {
			a.P[3*i+c] = clip255(c0[c] + (c1[c]-c0[c])*v)
		}
	}
	return a
}

func clip255(v float64) float64 { return min(max(v, 0), 255) }

// Px is the colour at (x, y).
func (a *Arr) Px(x, y int) Col {
	i := 3 * (y*a.W + x)
	return Col{a.P[i], a.P[i+1], a.P[i+2]}
}

// SetPx sets the colour at (x, y).
func (a *Arr) SetPx(x, y int, c Col) {
	i := 3 * (y*a.W + x)
	a.P[i], a.P[i+1], a.P[i+2] = c[0], c[1], c[2]
}

func span(lo, hi, n int) (int, int) {
	if lo < 0 {
		lo = max(lo+n, 0)
	}
	if hi < 0 {
		hi += n
	}
	return min(lo, n), min(hi, n)
}

// MulRect is a[y0:y1, x0:x1] *= f.
func (a *Arr) MulRect(x0, y0, x1, y1 int, f float64) {
	x0, x1 = span(x0, x1, a.W)
	y0, y1 = span(y0, y1, a.H)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := 3 * (y*a.W + x)
			a.P[i] *= f
			a.P[i+1] *= f
			a.P[i+2] *= f
		}
	}
}

// SetRect is a[y0:y1, x0:x1] = c.
func (a *Arr) SetRect(x0, y0, x1, y1 int, c Col) {
	x0, x1 = span(x0, x1, a.W)
	y0, y1 = span(y0, y1, a.H)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			a.SetPx(x, y, c)
		}
	}
}

// BlendRect is a[y0:y1, x0:x1] = a * k + c * (1 - k).
func (a *Arr) BlendRect(x0, y0, x1, y1 int, k float64, c Col) {
	x0, x1 = span(x0, x1, a.W)
	y0, y1 = span(y0, y1, a.H)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			p := a.Px(x, y)
			a.SetPx(x, y, Col{p[0]*k + c[0]*(1-k), p[1]*k + c[1]*(1-k), p[2]*k + c[2]*(1-k)})
		}
	}
}

// MulRows is a[start::step] *= f (whole rows).
func (a *Arr) MulRows(start, step int, f float64) {
	for y := start; y < a.H; y += step {
		a.MulRect(0, y, a.W, y+1, f)
	}
}

// MulCols is a[:, start::step] *= f (whole columns).
func (a *Arr) MulCols(start, step int, f float64) {
	for x := start; x < a.W; x += step {
		a.MulRect(x, 0, x+1, a.H, f)
	}
}

// SetRows is a[start::step] = c.
func (a *Arr) SetRows(start, step int, c Col) {
	for y := start; y < a.H; y += step {
		a.SetRect(0, y, a.W, y+1, c)
	}
}

// SetCols is a[:, start::step] = c.
func (a *Arr) SetCols(start, step int, c Col) {
	for x := start; x < a.W; x += step {
		a.SetRect(x, 0, x+1, a.H, c)
	}
}

// Canvas converts to an 8-bit RGB canvas: clip to 0..255, then truncate (numpy's astype(uint8)).
func (a *Arr) Canvas() *Canvas {
	c := NewCanvas(a.W, a.H, Col{})
	for i := range a.W * a.H {
		for k := range 3 {
			c.Pix[4*i+k] = uint8(clip255(a.P[3*i+k]))
		}
	}
	return c
}

// Arr converts a canvas back to floats (alpha dropped).
func (c *Canvas) Arr() *Arr {
	a := NewArr(c.W, c.H)
	for i := range c.W * c.H {
		for k := range 3 {
			a.P[3*i+k] = float64(c.Pix[4*i+k])
		}
	}
	return a
}
