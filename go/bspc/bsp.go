package main

import (
	"math"
	"runtime"
	"sort"
	"sync"
)

type SolidFn func(Vec) bool

// parallelFor runs f(i) for i in [0,n) on all cores. Results must be written to per-index slots.
func parallelFor(n int, f func(i int)) {
	workers := runtime.GOMAXPROCS(0)
	if n < 2 || workers < 2 {
		for i := 0; i < n; i++ {
			f(i)
		}
		return
	}
	var wg sync.WaitGroup
	next := make(chan int, n)
	for i := 0; i < n; i++ {
		next <- i
	}
	close(next)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				f(i)
			}
		}()
	}
	wg.Wait()
}

// ------------------------------------------------------------------ CSG
func visibleFaces(brushes []Brush, solid SolidFn) []Face {
	wins := make([][]Winding, len(brushes))
	parallelFor(len(brushes), func(i int) { wins[i] = brushWindings(brushes[i]) })
	BMN := make([]Vec, len(brushes))
	BMX := make([]Vec, len(brushes))
	for i, w := range wins {
		mn, mx := bounds(windingPoints(w))
		BMN[i] = mn.Sub(Vec{1, 1, 1})
		BMX[i] = mx.Add(Vec{1, 1, 1})
	}
	per := make([][]Face, len(brushes))
	parallelFor(len(brushes), func(bi int) {
		var out []Face
		for _, w := range wins[bi] {
			frags := []Poly{w.P}
			pmn, pmx := bounds(w.P)
			for bj := range brushes {
				if !(BMN[bj][0] <= pmx[0] && BMN[bj][1] <= pmx[1] && BMN[bj][2] <= pmx[2] &&
					BMX[bj][0] >= pmn[0] && BMX[bj][1] >= pmn[1] && BMX[bj][2] >= pmn[2]) {
					continue
				}
				if bj == bi {
					continue
				}
				other := brushes[bj]
				var nw []Poly
				for _, f := range frags {
					inside := f
					for _, pl := range other {
						fr, bk, st := split(inside, pl.N, pl.D)
						if st == stOn {
							bk = inside
						}
						if fr != nil {
							nw = append(nw, fr)
						}
						inside = bk
						if inside == nil {
							break
						}
					}
					if inside != nil {
						if !strictlyInside(other, mean(inside).Add(w.N.Mul(0.1)), EPS) {
							nw = append(nw, inside)
						}
					}
				}
				frags = nw
			}
			for _, f := range frags {
				if !solid(mean(f).Add(w.N.Mul(0.1))) {
					out = append(out, Face{f, w.N, w.D, w.Tex})
				}
			}
		}
		per[bi] = out
	})
	var out []Face
	for _, p := range per {
		out = append(out, p...)
	}
	return out
}

// ------------------------------------------------------------------ planes
type planeKey struct{ a, b, c, d float64 }
type DPlane struct {
	N   Vec
	D   float64
	Typ int
}

var dplanes []DPlane
var planeIdx = map[planeKey]int{}

func planeIndex(n Vec, d float64) (int, bool) {
	a := Vec{math.Abs(n[0]), math.Abs(n[1]), math.Abs(n[2])}
	ax := 0
	for i := 1; i < 3; i++ {
		if a[i] > a[ax] {
			ax = i
		}
	}
	flip := n[ax] < 0
	cn, cd := n, d
	if flip {
		cn, cd = n.Neg(), -d
	}
	typ := 3 + ax
	if a[ax] > 1-1e-6 {
		typ = ax
	}
	if typ < 3 {
		cn = Vec{}
		cn[ax] = 1
	}
	r := npRoundVec(cn, 5)
	key := planeKey{r[0] + 0, r[1] + 0, r[2] + 0, pyRound(cd, 3) + 0} // +0 folds -0 into 0, like Python's key equality
	if i, ok := planeIdx[key]; ok {
		return i, flip
	}
	planeIdx[key] = len(dplanes)
	dplanes = append(dplanes, DPlane{cn, cd, typ})
	return len(dplanes) - 1, flip
}

// ------------------------------------------------------------------ BSP build
func splitRegion(region []Poly, n Vec, d float64) ([]Poly, []Poly) {
	var fr, bk []Poly
	var caps []Vec
	for _, poly := range region {
		f, b, st := split(poly, n, d)
		if st == stOn {
			continue
		}
		if f != nil {
			fr = append(fr, f)
		}
		if b != nil {
			bk = append(bk, b)
		}
		if st == stSplit {
			for _, p := range f {
				if math.Abs(n.Dot(p)-d) < EPS {
					caps = append(caps, p)
				}
			}
		}
	}
	for _, poly := range region {
		for _, p := range poly {
			if math.Abs(n.Dot(p)-d) < EPS {
				caps = append(caps, p)
			}
		}
	}
	if len(caps) >= 3 {
		u := make([]Vec, len(caps))
		for i, p := range caps {
			u[i] = npRoundVec(p, 4)
		}
		sort.Slice(u, func(i, j int) bool {
			for k := 0; k < 3; k++ {
				if u[i][k] != u[j][k] {
					return u[i][k] < u[j][k]
				}
			}
			return false
		})
		var uniq []Vec
		for i, p := range u {
			if i == 0 || p != u[i-1] {
				uniq = append(uniq, p)
			}
		}
		if len(uniq) >= 3 {
			c := mean(uniq)
			up := Vec{0, 0, 1}
			if math.Abs(n[2]) >= 0.9 {
				up = Vec{1, 0, 0}
			}
			r := up.Cross(n)
			r = r.Div(r.Norm())
			uu := n.Cross(r)
			ang := make([]float64, len(uniq))
			for i, p := range uniq {
				q := p.Sub(c)
				ang[i] = math.Atan2(q.DotM(uu), q.DotM(r)) // numpy (caps - c) @ u (gemv)
			}
			var cp Poly
			for _, i := range argsortStable(ang) {
				cp = append(cp, uniq[i])
			}
			if cp = clean(cp); cp != nil {
				fr = append(fr, cp)
				bk = append(bk, append(Poly(nil), cp...))
			}
		}
	}
	return fr, bk
}

func boxRegion(mn, mx Vec) []Poly {
	x0, y0, z0 := mn[0], mn[1], mn[2]
	x1, y1, z1 := mx[0], mx[1], mx[2]
	V := func(x, y, z float64) Vec { return Vec{x, y, z} }
	return []Poly{
		{V(x0, y0, z0), V(x1, y0, z0), V(x1, y1, z0), V(x0, y1, z0)},
		{V(x0, y0, z1), V(x0, y1, z1), V(x1, y1, z1), V(x1, y0, z1)},
		{V(x0, y0, z0), V(x0, y1, z0), V(x0, y1, z1), V(x0, y0, z1)},
		{V(x1, y0, z0), V(x1, y0, z1), V(x1, y1, z1), V(x1, y1, z0)},
		{V(x0, y0, z0), V(x0, y0, z1), V(x1, y0, z1), V(x1, y0, z0)},
		{V(x0, y1, z0), V(x1, y1, z0), V(x1, y1, z1), V(x0, y1, z1)},
	}
}

func clipToBox(faces []Face, mn, mx Vec) []Face {
	bp := []struct {
		n Vec
		d float64
	}{{Vec{1, 0, 0}, mx[0]}, {Vec{-1, 0, 0}, -mn[0]}, {Vec{0, 1, 0}, mx[1]}, {Vec{0, -1, 0}, -mn[1]},
		{Vec{0, 0, 1}, mx[2]}, {Vec{0, 0, -1}, -mn[2]}}
	var out []Face
	for _, f := range faces {
		w := f.P
		for _, b := range bp {
			if w == nil {
				break
			}
			var st int
			_, w, st = split(w, b.n, b.d)
			if st == stOn {
				w = nil
			}
		}
		if w != nil {
			out = append(out, Face{w, f.N, f.D, f.Tex})
		}
	}
	return out
}

type candKey struct{ a, b, c, d float64 }

func chooseSplitter(polys []Face) (Vec, float64) {
	type cand struct {
		n Vec
		d float64
	}
	seen := map[candKey]bool{}
	var items []cand
	for _, f := range polys {
		r := npRoundVec(f.N, 4)
		k := candKey{r[0] + 0, r[1] + 0, r[2] + 0, pyRound(f.D, 2) + 0}
		rn := npRoundVec(f.N.Neg(), 4)
		nk := candKey{rn[0] + 0, rn[1] + 0, rn[2] + 0, pyRound(-f.D, 2) + 0}
		if seen[k] || seen[nk] {
			continue
		}
		seen[k] = true
		items = append(items, cand{f.N, f.D})
	}
	if len(items) > 48 {
		step := len(items) / 48
		if step < 1 {
			step = 1
		}
		var sub []cand
		for i := 0; i < len(items); i += step {
			sub = append(sub, items[i])
		}
		items = sub
	}
	var best cand
	bscore, have := 0, false
	for _, c := range items {
		sp, fr, bk := 0, 0, 0
		for _, f := range polys {
			hi, lo := math.Inf(-1), math.Inf(1)
			for _, p := range f.P {
				x := p.DotM(c.n) - c.d // numpy V @ n (gemv)
				hi = math.Max(hi, x)
				lo = math.Min(lo, x)
			}
			h, l := hi > EPS, lo < -EPS
			if h && l {
				sp++
			} else if h {
				fr++
			} else if l {
				bk++
			}
		}
		axial := math.Max(math.Abs(c.n[0]), math.Max(math.Abs(c.n[1]), math.Abs(c.n[2]))) > 1-1e-6
		d := fr - bk
		if d < 0 {
			d = -d
		}
		score := sp*8 + d
		if axial {
			score -= 6
		}
		if !have || score < bscore {
			best, bscore, have = c, score, true
		}
	}
	return best.n, best.d
}

type Node struct {
	Leaf, Solid bool
	Mins, Maxs  Vec
	N           Vec
	D           float64
	On          []Face
	Front, Back *Node
	LeafIndex   int
}

const parallelTreeMin = 400

func buildTree(polys []Face, region []Poly, solid SolidFn) *Node {
	node := &Node{}
	var pts []Vec
	for _, p := range region {
		pts = append(pts, p...)
	}
	if len(pts) == 0 {
		pts = []Vec{{0, 0, 0}}
	}
	node.Mins, node.Maxs = bounds(pts)
	if len(polys) == 0 {
		node.Leaf = true
		node.Solid = len(region) == 0 || solid(mean(pts))
		return node
	}
	n, d := chooseSplitter(polys)
	node.N, node.D = n, d
	var on, fr, bk []Face
	for _, f := range polys {
		a, b, st := split(f.P, n, d)
		if st == stOn {
			on = append(on, f)
			continue
		}
		if a != nil {
			fr = append(fr, Face{a, f.N, f.D, f.Tex})
		}
		if b != nil {
			bk = append(bk, Face{b, f.N, f.D, f.Tex})
		}
	}
	node.On = on
	rf, rb := splitRegion(region, n, d)
	if len(fr)+len(bk) > parallelTreeMin {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); node.Front = buildTree(fr, rf, solid) }()
		node.Back = buildTree(bk, rb, solid)
		wg.Wait()
	} else {
		node.Front = buildTree(fr, rf, solid)
		node.Back = buildTree(bk, rb, solid)
	}
	return node
}

func expandBrush(b Brush, mins, maxs Vec) Brush {
	bmn, bmx := bounds(windingPoints(brushWindings(b)))
	pl := append(Brush(nil), b...)
	for ax := 0; ax < 3; ax++ {
		for _, sgn := range []float64{1, -1} {
			var n Vec
			n[ax] = sgn
			dup := false
			for _, q := range pl {
				if allclose(n, q.N) {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			d := -bmn[ax]
			if sgn > 0 {
				d = bmx[ax]
			}
			pl = append(pl, Plane{n, d, nil})
		}
	}
	out := make(Brush, len(pl))
	for k, p := range pl {
		off := 0.0
		for i := 0; i < 3; i++ {
			if p.N[i] > 0 {
				off += p.N[i] * -mins[i]
			} else {
				off += p.N[i] * -maxs[i]
			}
		}
		out[k] = Plane{p.N, p.D + off, p.Tex}
	}
	return out
}

// brushSet answers "is p strictly inside any brush" with a bounding-box prefilter (same answers, faster).
type brushSet struct {
	b      []Brush
	mn, mx []Vec
}

func newBrushSet(bs []Brush) *brushSet {
	s := &brushSet{b: bs, mn: make([]Vec, len(bs)), mx: make([]Vec, len(bs))}
	for i, b := range bs {
		pts := windingPoints(brushWindings(b))
		if len(pts) == 0 {
			s.mn[i], s.mx[i] = Vec{1, 1, 1}, Vec{-1, -1, -1} // empty: never inside
			continue
		}
		mn, mx := bounds(pts)
		s.mn[i], s.mx[i] = mn.Sub(Vec{0.5, 0.5, 0.5}), mx.Add(Vec{0.5, 0.5, 0.5})
	}
	return s
}

func (s *brushSet) inside(p Vec) bool {
	for i, b := range s.b {
		if p[0] < s.mn[i][0] || p[1] < s.mn[i][1] || p[2] < s.mn[i][2] ||
			p[0] > s.mx[i][0] || p[1] > s.mx[i][1] || p[2] > s.mx[i][2] {
			continue
		}
		if strictlyInside(b, p, EPS) {
			return true
		}
	}
	return false
}
