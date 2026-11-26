package mapkit

import "math"

// V3 is a 3-D vector. The arithmetic below keeps numpy's operation order (fused multiply-adds included), so
// brush planes come out bit for bit the same as the original Python generator wrote them.
type V3 [3]float64

// Add returns a + b.
func (a V3) Add(b V3) V3 { return V3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }

// Sub returns a - b.
func (a V3) Sub(b V3) V3 { return V3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

// Mul returns a * s.
func (a V3) Mul(s float64) V3 { return V3{a[0] * s, a[1] * s, a[2] * s} }

// Div returns a / s.
func (a V3) Div(s float64) V3 { return V3{a[0] / s, a[1] / s, a[2] / s} }

// Neg returns -a.
func (a V3) Neg() V3 { return V3{-a[0], -a[1], -a[2]} }

// Dot is numpy's 1-D a @ b: fma(a2,b2, fma(a1,b1, a0*b0)).
func (a V3) Dot(b V3) float64 {
	return math.FMA(a[2], b[2], math.FMA(a[1], b[1], float64(a[0]*b[0])))
}

// Norm is numpy.linalg.norm of a 3-vector: sqrt(a @ a).
func (a V3) Norm() float64 { return math.Sqrt(a.Dot(a)) }

// Unit returns a / |a|.
func (a V3) Unit() V3 { return a.Div(a.Norm()) }

// Cross is numpy.cross.
func (a V3) Cross(b V3) V3 {
	return V3{float64(a[1]*b[2]) - float64(a[2]*b[1]), float64(a[2]*b[0]) - float64(a[0]*b[2]),
		float64(a[0]*b[1]) - float64(a[1]*b[0])}
}

// V2 is a 2-D vector (plan view).
type V2 [2]float64

// Dot is numpy's 1-D a @ b for 2-vectors: fma(a1,b1, a0*b0).
func (a V2) Dot(b V2) float64 { return math.FMA(a[1], b[1], float64(a[0]*b[0])) }

// Norm is numpy.linalg.norm of a 2-vector.
func (a V2) Norm() float64 { return math.Sqrt(a.Dot(a)) }

// Sub returns a - b.
func (a V2) Sub(b V2) V2 { return V2{a[0] - b[0], a[1] - b[1]} }

// Add returns a + b.
func (a V2) Add(b V2) V2 { return V2{a[0] + b[0], a[1] + b[1]} }

// Mul returns a * s.
func (a V2) Mul(s float64) V2 { return V2{a[0] * s, a[1] * s} }
