package main

// Geometry helpers. The floating-point operation order is what produced the golden output (`make golden`):
// (a*b + c*d) + e*f is not the same number as a*b + (c*d + e*f), so reordering is an output change, not a
// refactor. See README.md.

import (
	"math"
	"sort"
	"strconv"
)

const EPS = 0.01

type Vec [3]float64

func (a Vec) Add(b Vec) Vec     { return Vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a Vec) Sub(b Vec) Vec     { return Vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a Vec) Mul(s float64) Vec { return Vec{float64(a[0] * s), float64(a[1] * s), float64(a[2] * s)} }
func (a Vec) Div(s float64) Vec { return Vec{a[0] / s, a[1] / s, a[2] / s} }
func (a Vec) Neg() Vec          { return Vec{-a[0], -a[1], -a[2]} }

// Dot products, each matching the numpy expression it replaces bit for bit (measured; see tools/bsp/bspc/README.md):
//
//	Dot  = numpy 1-D a @ b (and A @ M.T)  fma(a2,b2, fma(a1,b1, a0*b0))
//	DotM = numpy 2-D rows @ vector        fma(a2,b2, fma(a0,b0, a1*b1))   (OpenBLAS gemv)
//	DotE = numpy einsum over 3            (a0*b0 + a2*b2) + a1*b1
//
// float64() conversions stop the Go compiler from fusing plain products into FMAs on arm64.
func (a Vec) Dot(b Vec) float64 {
	return math.FMA(a[2], b[2], math.FMA(a[1], b[1], float64(a[0]*b[0])))
}
func (a Vec) DotM(b Vec) float64 {
	return math.FMA(a[2], b[2], math.FMA(a[0], b[0], float64(a[1]*b[1])))
}
func (a Vec) DotE(b Vec) float64 {
	return float64(float64(float64(a[0]*b[0])+float64(a[2]*b[2])) + float64(a[1]*b[1]))
}
func (a Vec) Norm() float64 { return math.Sqrt(a.Dot(a)) }
func (a Vec) Cross(b Vec) Vec {
	return Vec{float64(a[1]*b[2]) - float64(a[2]*b[1]), float64(a[2]*b[0]) - float64(a[0]*b[2]), float64(a[0]*b[1]) - float64(a[1]*b[0])}
}

type Tex struct {
	Name string
	S, T [4]float64
}

type Plane struct {
	N   Vec
	D   float64
	Tex *Tex // nil for bevel planes added to clip hulls
}

type Brush []Plane
type Poly []Vec

type Face struct {
	P   Poly
	N   Vec
	D   float64
	Tex *Tex
}

const (
	stOn = iota
	stFront
	stBack
	stSplit
)

// npRound mirrors numpy.round(x, dec) for dec > 0: rint(x * 10**dec) / 10**dec.
func npRound(x float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.RoundToEven(x*p) / p
}

// pyRound mirrors Python's round(x, n): correctly rounded on the exact binary value.
func pyRound(x float64, n int) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return v
}

func npRoundVec(v Vec, dec int) Vec {
	return Vec{npRound(v[0], dec), npRound(v[1], dec), npRound(v[2], dec)}
}

func allclose(a, b Vec) bool {
	for i := range 3 {
		if math.Abs(a[i]-b[i]) > 1e-8+1e-5*math.Abs(b[i]) {
			return false
		}
	}
	return true
}

func baseWinding(n Vec, d float64) Poly {
	const size = 1e5
	up := Vec{0, 0, 1}
	if math.Abs(n[2]) >= 0.9 {
		up = Vec{1, 0, 0}
	}
	r := up.Cross(n)
	r = r.Div(r.Norm())
	u := n.Cross(r)
	o := n.Mul(d)
	return Poly{o.Add(r.Neg().Add(u).Mul(size)), o.Add(r.Add(u).Mul(size)), o.Add(r.Sub(u).Mul(size)), o.Add(r.Neg().Sub(u).Mul(size))}
}

func split(poly Poly, n Vec, d float64) (Poly, Poly, int) {
	k := len(poly)
	dists := make([]float64, k)
	sides := make([]int, k)
	allOn, allGE, allLE := true, true, true
	for i, p := range poly {
		x := n.Dot(p) - d
		dists[i] = x
		s := 0
		if x > EPS {
			s = 1
		} else if x < -EPS {
			s = -1
		}
		sides[i] = s
		if s != 0 {
			allOn = false
		}
		if s < 0 {
			allGE = false
		}
		if s > 0 {
			allLE = false
		}
	}
	if allOn {
		return nil, nil, stOn
	}
	if allGE {
		return poly, nil, stFront
	}
	if allLE {
		return nil, poly, stBack
	}
	var f, b Poly
	for i := range k {
		j := (i + 1) % k
		p, q := poly[i], poly[j]
		sp, sq := sides[i], sides[j]
		if sp >= 0 {
			f = append(f, p)
		}
		if sp <= 0 {
			b = append(b, p)
		}
		if sp*sq < 0 {
			t := dists[i] / (dists[i] - dists[j])
			m := p.Add(q.Sub(p).Mul(t)) // elementwise numpy ops: no fusion
			f = append(f, m)
			b = append(b, m)
		}
	}
	return clean(f), clean(b), stSplit
}

func clean(poly Poly) Poly {
	var out Poly
	for _, p := range poly {
		if len(out) == 0 || p.Sub(out[len(out)-1]).Norm() > EPS {
			out = append(out, p)
		}
	}
	if len(out) > 1 && out[0].Sub(out[len(out)-1]).Norm() < EPS {
		out = out[:len(out)-1]
	}
	if len(out) >= 3 && area(out) > 0.05 {
		return out
	}
	return nil
}

func area(poly Poly) float64 {
	var s Vec
	for i := 1; i < len(poly)-1; i++ {
		s = s.Add(poly[i].Sub(poly[0]).Cross(poly[i+1].Sub(poly[0])))
	}
	return s.Norm() / 2
}

// mean mirrors numpy's mean over axis 0 of an (k,3) array: running sum, then divide.
func mean(pts []Vec) Vec {
	var s Vec
	for _, p := range pts {
		s = s.Add(p)
	}
	return s.Div(float64(len(pts)))
}

func bounds(pts []Vec) (Vec, Vec) {
	mn, mx := pts[0], pts[0]
	for _, p := range pts[1:] {
		for i := range 3 {
			mn[i] = math.Min(mn[i], p[i])
			mx[i] = math.Max(mx[i], p[i])
		}
	}
	return mn, mx
}

type Winding struct {
	P   Poly
	N   Vec
	D   float64
	Tex *Tex
}

func brushWindings(planes Brush) []Winding {
	var out []Winding
	for i, pl := range planes {
		w := baseWinding(pl.N, pl.D)
		for j, p2 := range planes {
			if i == j || w == nil {
				continue
			}
			if allclose(pl.N, p2.N) && math.Abs(pl.D-p2.D) < EPS {
				continue
			}
			var st int
			_, w, st = split(w, p2.N, p2.D)
			if st == stOn {
				w = nil
			}
		}
		if w != nil {
			out = append(out, Winding{w, pl.N, pl.D, pl.Tex})
		}
	}
	return out
}

func windingPoints(ws []Winding) []Vec {
	var pts []Vec
	for _, w := range ws {
		pts = append(pts, w.P...)
	}
	return pts
}

func strictlyInside(planes Brush, p Vec, margin float64) bool {
	for _, pl := range planes {
		if !(pl.N.Dot(p)-pl.D < -margin) {
			return false
		}
	}
	return true
}

// argsortStable: numpy's argsort uses insertion sort (stable) for the small arrays we sort.
func argsortStable(keys []float64) []int {
	idx := make([]int, len(keys))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return keys[idx[a]] < keys[idx[b]] })
	return idx
}
