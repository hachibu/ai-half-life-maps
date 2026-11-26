package main

// Numeric primitives that reproduce Python/numpy results bit for bit (each measured on 100,000 cases).

import (
	"math"
	"math/big"
	"strconv"
)

// dot3 = numpy.dot / @ on 3-vectors (OpenBLAS ddot): fma(a2,b2, fma(a1,b1, a0*b0)).
func dot3(a0, a1, a2, b0, b1, b2 float64) float64 {
	return math.FMA(a2, b2, math.FMA(a1, b1, float64(a0*b0)))
}

// norm2 = numpy.linalg.norm on a 2-vector: sqrt(fma(v1,v1, v0*v0)).
func norm2(v0, v1 float64) float64 { return math.Sqrt(math.FMA(v1, v1, float64(v0*v0))) }

// hypot = Python's math.hypot(x, y), which is correctly rounded. A double-double estimate settles almost
// every case; near-ties fall back to exact big-float arithmetic.
func hypot(x, y float64) float64 {
	x, y = math.Abs(x), math.Abs(y)
	if x < y {
		x, y = y, x
	}
	if y == 0 || math.IsInf(x, 0) || math.IsNaN(x) || math.IsNaN(y) {
		if math.IsNaN(x) || math.IsNaN(y) {
			return math.NaN()
		}
		return x
	}
	p1 := x * x
	e1 := math.FMA(x, x, -p1)
	p2 := y * y
	e2 := math.FMA(y, y, -p2)
	s := p1 + p2
	bv := s - p1
	es := (p1 - (s - bv)) + (p2 - bv) // TwoSum error
	lo := es + e1 + e2
	r := math.Sqrt(s + lo)
	resid := func(c float64) float64 { // c*c - (s + lo), with c*c exact as cc + ec
		cc := c * c
		ec := math.FMA(c, c, -cc)
		return ((cc - s) + ec) - lo
	}
	best, bd := r, math.Abs(resid(r))
	for _, c := range []float64{math.Nextafter(r, 0), math.Nextafter(r, math.Inf(1))} {
		if d := math.Abs(resid(c)); d < bd {
			best, bd = c, d
		}
	}
	// near a rounding midpoint the double-double residual isn't precise enough: decide exactly
	ulp := math.Nextafter(best, math.Inf(1)) - best
	if bd > 0.25*ulp*best {
		return exactHypot(x, y)
	}
	return best
}

func exactHypot(x, y float64) float64 {
	const prec = 400
	bx := new(big.Float).SetPrec(prec).SetFloat64(x)
	by := new(big.Float).SetPrec(prec).SetFloat64(y)
	s := new(big.Float).SetPrec(prec).Mul(bx, bx)
	s.Add(s, new(big.Float).SetPrec(prec).Mul(by, by))
	f, _ := new(big.Float).SetPrec(prec).Sqrt(s).Float64()
	return f
}

// floorDiv = Python's float // float (CPython float_floor_div).
func floorDiv(vx, wx float64) float64 {
	mod := math.Mod(vx, wx)
	div := (vx - mod) / wx
	if mod != 0 {
		if (wx < 0) != (mod < 0) {
			div -= 1.0
		}
	}
	var fd float64
	if div != 0 {
		fd = math.Floor(div)
		if div-fd > 0.5 {
			fd += 1.0
		}
	} else {
		fd = math.Copysign(0, vx/wx)
	}
	return fd
}

// pyRound = Python's round(x, n) (correctly rounded on the exact binary value).
func pyRound(x float64, n int) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return v
}
