package mapkit

import "math"

// Plane is N·p = D with N a unit normal pointing out of the solid.
type Plane struct {
	N V3
	D float64
}

// HullPlanes returns the outward planes of the convex hull of pts, one per face (coplanar facets merged). The
// point sets are small (a few dozen points), so every triple is tried; for each face the triple with the
// largest spread decides the normal, which keeps it accurate.
func HullPlanes(pts []V3) []Plane {
	scale := 0.0
	for _, p := range pts {
		scale = max(scale, math.Abs(p[0]), math.Abs(p[1]), math.Abs(p[2]))
	}
	eps := 1e-7 * max(scale, 1)
	type found struct {
		p      Plane
		weight float64
	}
	var out []found
	n := len(pts)
	for i := range n {
		for j := i + 1; j < n; j++ {
			for k := j + 1; k < n; k++ {
				c := pts[j].Sub(pts[i]).Cross(pts[k].Sub(pts[i]))
				w := c.Norm()
				if w < 1e-9 {
					continue
				}
				nrm := c.Div(w)
				d := nrm.Dot(pts[i])
				above, below := false, false
				for _, p := range pts {
					s := nrm.Dot(p) - d
					above = above || s > eps*10
					below = below || s < -eps*10
				}
				if above && below {
					continue
				}
				if above {
					nrm, d = nrm.Neg(), -d
				}
				merged := false
				for fi := range out {
					f := &out[fi]
					if f.p.N.Dot(nrm) > 1-1e-9 && math.Abs(f.p.D-d) < 1e-3 {
						if w > f.weight {
							f.p, f.weight = Plane{nrm, d}, w
						}
						merged = true
						break
					}
				}
				if !merged {
					out = append(out, found{Plane{nrm, d}, w})
				}
			}
		}
	}
	planes := make([]Plane, len(out))
	for i, f := range out {
		planes[i] = f.p
	}
	return planes
}
