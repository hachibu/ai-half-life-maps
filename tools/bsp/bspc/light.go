package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	kLight  = 30000.0
	ambient = 14.0
)

type Light struct {
	P   Vec
	Col Vec
}

func collectLights(ents []*Entity) []Light {
	var out []Light
	for _, e := range ents {
		if e.Get("classname") != "light" {
			continue
		}
		f := strings.Fields(e.Get("_light"))
		var v [4]float64
		for i := range v {
			v[i], _ = strconv.ParseFloat(f[i], 64)
		}
		o := floats(e.Get("origin"))
		col := Vec{v[0] / 255.0 * v[3], v[1] / 255.0 * v[3], v[2] / 255.0 * v[3]}
		out = append(out, Light{Vec{o[0], o[1], o[2]}, col})
	}
	return out
}

func overlaps(amn, amx, bmn, bmx Vec) bool {
	return amn[0] <= bmx[0] && amn[1] <= bmx[1] && amn[2] <= bmx[2] &&
		amx[0] >= bmn[0] && amx[1] >= bmn[1] && amx[2] >= bmn[2]
}

func finalizeLighting(world []Brush, lights []Light) {
	t0 := time.Now()
	F := len(pending)
	bmin := make([]Vec, len(world))
	bmax := make([]Vec, len(world))
	for i, b := range world {
		bmin[i], bmax[i] = bounds(windingPoints(brushWindings(b)))
	}
	start := make([][]Vec, F)
	cent := make([]Vec, F)
	for fi, pd := range pending {
		cent[fi] = mean(pd.P)
		s := make([]Vec, len(pd.P))
		for i, p := range pd.P {
			s[i] = p.Add(pd.N.Mul(1.0))
		}
		start[fi] = s
	}

	// 1) pull luxels that sit inside solid towards their face centre
	for range 6 {
		fmn := make([]Vec, F)
		fmx := make([]Vec, F)
		for fi := range start {
			fmn[fi], fmx[fi] = bounds(start[fi])
		}
		var moved int64
		parallelFor(F, func(fi int) {
			var cand []int
			for k := range world {
				if bmin[k][0] <= fmx[fi][0] && bmin[k][1] <= fmx[fi][1] && bmin[k][2] <= fmx[fi][2] &&
					bmax[k][0] >= fmn[fi][0] && bmax[k][1] >= fmn[fi][1] && bmax[k][2] >= fmn[fi][2] {
					cand = append(cand, k)
				}
			}
			if len(cand) == 0 {
				return
			}
			n := pending[fi].N
			var idx []int
			for i, x := range start[fi] {
				for _, k := range cand {
					in := true
					for _, pl := range world[k] {
						if !(x.DotE(pl.N)-pl.D < -0.01) { // numpy einsum
							in = false
							break
						}
					}
					if in {
						idx = append(idx, i)
						break
					}
				}
			}
			for _, i := range idx {
				x := start[fi][i]
				start[fi][i] = x.Add(cent[fi].Add(n).Sub(x).Mul(0.5))
			}
			atomic.AddInt64(&moved, int64(len(idx)))
		})
		if moved == 0 {
			break
		}
	}
	fmt.Printf("  luxel nudge done %.1fs\n", time.Since(t0).Seconds())

	// 2) per light: radius, and the brushes near it
	type lightInfo struct {
		R    float64
		near []int
	}
	info := make([]lightInfo, len(lights))
	for li, l := range lights {
		R := math.Sqrt(kLight * math.Max(l.Col[0], math.Max(l.Col[1], l.Col[2])) / 2.0)
		lo, hi := l.P.Sub(Vec{R, R, R}), l.P.Add(Vec{R, R, R})
		var near []int
		for k := range world {
			if bmin[k][0] <= hi[0] && bmin[k][1] <= hi[1] && bmin[k][2] <= hi[2] &&
				bmax[k][0] >= lo[0] && bmax[k][1] >= lo[1] && bmax[k][2] >= lo[2] {
				near = append(near, k)
			}
		}
		info[li] = lightInfo{R, near}
	}

	// 3) every face independently: ambient, then each light in order (same sum order as the Python version)
	var done atomic.Int64
	total := 0
	parallelFor(F, func(fi int) {
		S := start[fi]
		N := pending[fi].N
		acc := make([]Vec, len(S))
		for i := range acc {
			acc[i] = Vec{ambient, ambient, ambient}
		}
		fmn, fmx := bounds(S)
		dvec := make([]Vec, len(S))
		dist := make([]float64, len(S))
		cosv := make([]float64, len(S))
		ok := make([]bool, len(S))
		for li, l := range lights {
			R := info[li].R
			// whole face out of range? (conservative: 1 unit margin keeps this identical to per-luxel tests)
			var gap Vec
			for a := range 3 {
				if l.P[a] < fmn[a] {
					gap[a] = fmn[a] - l.P[a]
				} else if l.P[a] > fmx[a] {
					gap[a] = l.P[a] - fmx[a]
				}
			}
			if gap.Norm() > R+1 {
				continue
			}
			anyOK := false
			for i, s := range S {
				d := l.P.Sub(s)
				dvec[i] = d
				dist[i] = math.Sqrt(float64(float64(d[0]*d[0])+float64(d[1]*d[1])) + float64(d[2]*d[2])) // numpy norm(axis=1)
				cosv[i] = d.DotE(N) / math.Max(dist[i], 1e-6)                                            // numpy einsum
				ok[i] = cosv[i] > 0 && dist[i] < R
				anyOK = anyOK || ok[i]
			}
			if !anyOK {
				continue
			}
			smn, smx := fmn, fmx
			for a := range 3 {
				smn[a] = math.Min(fmn[a], l.P[a])
				smx[a] = math.Max(fmx[a], l.P[a])
			}
			var cand []int
			for _, k := range info[li].near {
				if overlaps(bmin[k], bmax[k], smn, smx) {
					cand = append(cand, k)
				}
			}
			for i := range S {
				if !ok[i] {
					continue
				}
				hit := false
				for _, k := range cand {
					t0, t1 := 0.0, 1.0
					parOut := false
					for _, pl := range world[k] {
						sn := S[i].DotE(pl.N) - pl.D // numpy einsum
						dn := dvec[i].DotE(pl.N)
						switch {
						case dn < 0:
							t0 = math.Max(t0, -sn/dn)
						case dn > 0:
							t1 = math.Min(t1, -sn/dn)
						case sn > 0:
							parOut = true
						}
					}
					if t0 < t1-1e-4 && !parOut {
						hit = true
						break
					}
				}
				if hit {
					continue
				}
				m := math.Max(dist[i], 64)
				val := float64(kLight*cosv[i]) / float64(m*m)
				acc[i] = Vec{acc[i][0] + float64(val*l.Col[0]), acc[i][1] + float64(val*l.Col[1]), acc[i][2] + float64(val*l.Col[2])}
			}
		}
		pd := pending[fi]
		e, hasE := emissive[pd.Name]
		for i, a := range acc {
			for c := range 3 {
				v := a[c]
				if hasE {
					v = math.Max(v, e)
				}
				v = math.Min(math.Max(v, 0), 255)
				lightdata[pd.Ofs+3*i+c] = byte(v)
			}
		}
		if n := done.Add(1); n%2000 == 0 {
			fmt.Printf("  faces lit %d/%d  %.1fs\n", n, F, time.Since(t0).Seconds())
		}
	})
	for _, pd := range pending {
		total += len(pd.P)
	}
	fmt.Printf("luxels lit: %d in %.1fs\n", total, time.Since(t0).Seconds())
}
