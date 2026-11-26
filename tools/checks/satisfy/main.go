// satisfy: Go port of tools/checks/satisfaction.py (the 0-100 playability score). Same sampling, same
// random numbers (numpy's default_rng reproduced in rng.go), same walker and graph, so it prints the same
// score; the navigation graph is built on all cores.
//
//	satisfy build/<map>/<map>.bsp [--json out.json]      (seed: SATISFACTION_SEED, default 0)
package main

import (
	"container/heap"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	runUnits = 4800.0
	stepH    = 18
)

// ------------------------------------------------------------------ BSP (only what the walker needs)
type bspPlane struct {
	n   [3]float64
	d   float64
	typ int32
}
type clipNode struct {
	plane int32
	ch    [2]int16
}
type BSP struct {
	planes     []bspPlane
	clip       []clipNode
	mins, maxs [3]float64
	head1      int
	ents       []map[string]string
}

var blockRe = regexp.MustCompile(`\{([^{}]*)\}`)
var kvRe = regexp.MustCompile(`"([^"]*)" "([^"]*)"`)

func loadBSP(path string) *BSP {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	le := binary.LittleEndian
	lump := func(i int) []byte {
		o := int(int32(le.Uint32(data[4+8*i:])))
		n := int(int32(le.Uint32(data[8+8*i:])))
		return data[o : o+n]
	}
	f32 := func(b []byte) float64 { return float64(math.Float32frombits(le.Uint32(b))) }
	m := &BSP{}
	pl := lump(1)
	for k := 0; k+20 <= len(pl); k += 20 {
		m.planes = append(m.planes, bspPlane{[3]float64{f32(pl[k:]), f32(pl[k+4:]), f32(pl[k+8:])}, f32(pl[k+12:]), int32(le.Uint32(pl[k+16:]))})
	}
	cl := lump(9)
	for k := 0; k+8 <= len(cl); k += 8 {
		m.clip = append(m.clip, clipNode{int32(le.Uint32(cl[k:])), [2]int16{int16(le.Uint16(cl[k+4:])), int16(le.Uint16(cl[k+6:]))}})
	}
	md := lump(14)
	for i := 0; i < 3; i++ {
		m.mins[i] = f32(md[4*i:])
		m.maxs[i] = f32(md[12+4*i:])
	}
	m.head1 = int(int32(le.Uint32(md[36+4:])))
	for _, b := range blockRe.FindAllStringSubmatch(string(lump(0)), -1) {
		e := map[string]string{}
		for _, kv := range kvRe.FindAllStringSubmatch(b[1], -1) {
			e[kv[1]] = kv[2]
		}
		m.ents = append(m.ents, e)
	}
	return m
}

// free mirrors reader.player_free: point test in clip hull 1.
func (m *BSP) free(x, y, z float64) bool {
	p := [3]float64{x, y, z}
	n := m.head1
	for n >= 0 {
		c := m.clip[n]
		pl := m.planes[c.plane]
		var d float64
		if pl.typ < 3 {
			d = p[pl.typ] - pl.d
		} else {
			d = dot3(pl.n[0], pl.n[1], pl.n[2], x, y, z) - pl.d
		}
		if d >= 0 {
			n = int(c.ch[0])
		} else {
			n = int(c.ch[1])
		}
	}
	return n == -1
}

// walk mirrors tools/checks/walk.py Walker.walk for a single segment (step 4).
func (m *BSP) walk(floorZ, ax, ay, bx, by float64) (bool, float64) {
	z := floorZ + 36
	if !m.free(ax, ay, z) {
		return false, floorZ
	}
	dx, dy := bx-ax, by-ay
	n := int(norm2(dx, dy) / 4)
	if n < 1 {
		n = 1
	}
	for i := 1; i <= n; i++ {
		px := ax + float64(dx*float64(i))/float64(n)
		py := ay + float64(dy*float64(i))/float64(n)
		up := 0
		for ; up <= stepH; up++ {
			if m.free(px, py, z+float64(up)) {
				break
			}
		}
		if up > stepH {
			return false, z - 36
		}
		z += float64(up)
		for m.free(px, py, z-1) && z > -4000 {
			z -= 1
		}
	}
	return true, z - 36
}

// ------------------------------------------------------------------ graph
type pt [3]float64
type edge struct {
	to int
	d  float64
}

func arange(start, stop, step float64) []float64 {
	n := int(math.Ceil((stop - start) / step))
	if n <= 0 {
		return nil
	}
	out := []float64{start}
	delta := (start + step) - start
	for i := 1; i < n; i++ {
		out = append(out, start+float64(i)*delta)
	}
	return out
}

type item struct {
	cls     string
	x, y, z float64
}

func origin(e map[string]string) (float64, float64, float64) {
	f := strings.Fields(e["origin"])
	v := [3]float64{}
	for i := range v {
		v[i], _ = strconv.ParseFloat(f[i], 64)
	}
	return v[0], v[1], v[2]
}

func nearest(pts []pt, x, y, z float64) int {
	best, bi := 0.0, -1
	for i, p := range pts {
		d := hypot(p[0]-x, p[1]-y)
		if d <= 80 && math.Abs((p[2]-36)-z) <= 60 && (bi < 0 || d < best) {
			best, bi = d, i
		}
	}
	return bi
}

type hitem struct {
	d float64
	u int
}
type minHeap []hitem

func (h minHeap) Len() int { return len(h) }
func (h minHeap) Less(i, j int) bool {
	if h[i].d != h[j].d {
		return h[i].d < h[j].d
	}
	return h[i].u < h[j].u
}
func (h minHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)   { *h = append(*h, x.(hitem)) }
func (h *minHeap) Pop() any {
	o := *h
	x := o[len(o)-1]
	*h = o[:len(o)-1]
	return x
}

func dijkstra(edges [][]edge, srcs []int, limit float64) map[int]float64 {
	dist := map[int]float64{}
	h := &minHeap{}
	for _, s := range srcs {
		dist[s] = 0
		*h = append(*h, hitem{0, s})
	}
	heap.Init(h)
	for h.Len() > 0 {
		it := heap.Pop(h).(hitem)
		cur, ok := dist[it.u]
		if !ok {
			cur = math.Inf(1)
		}
		if it.d > cur || it.d > limit {
			continue
		}
		for _, e := range edges[it.u] {
			nd := it.d + e.d
			old, ok := dist[e.to]
			if !ok {
				old = math.Inf(1)
			}
			if nd < old && nd <= limit {
				dist[e.to] = nd
				heap.Push(h, hitem{nd, e.to})
			}
		}
	}
	return dist
}

func parallelFor(n int, f func(i int)) {
	var wg sync.WaitGroup
	next := make(chan int, 256)
	for w := 0; w < runtime.GOMAXPROCS(0); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				f(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		next <- i
	}
	close(next)
	wg.Wait()
}

func mean(xs []bool) float64 {
	s := 0.0
	for _, b := range xs {
		if b {
			s += 1
		}
	}
	return s / float64(len(xs))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: satisfy map.bsp [--json out.json]")
		os.Exit(2)
	}
	t0 := time.Now()
	seed := uint64(0)
	if s := os.Getenv("SATISFACTION_SEED"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "SATISFACTION_SEED must be a non-negative integer")
			os.Exit(2)
		}
		seed = v
	}
	rng := NewRNG(seed)
	m := loadBSP(os.Args[1])
	var spawns []pt
	var items []item
	for _, e := range m.ents {
		c := e["classname"]
		if strings.HasPrefix(c, "info_player_") {
			x, y, z := origin(e)
			spawns = append(spawns, pt{x, y, z})
		}
		if strings.HasPrefix(c, "weapon_") || strings.HasPrefix(c, "ammo_") || strings.HasPrefix(c, "item_") {
			x, y, z := origin(e)
			items = append(items, item{c, x, y, z})
		}
	}
	area := float64((m.maxs[0] - m.mins[0]) * (m.maxs[1] - m.mins[1]))
	spacing := math.Max(28.0, math.Sqrt(area/9000))
	link := spacing * 1.75

	// columns: same jitter sequence as Python (x then y per column, column-major over the grid)
	xs := arange(m.mins[0]+24, m.maxs[0]-24, spacing)
	ys := arange(m.mins[1]+24, m.maxs[1]-24, spacing)
	type col struct{ x, y float64 }
	var cols []col
	for _, gx := range xs {
		for _, gy := range ys {
			x := gx + float64(rng.Uniform(-0.3, 0.3)*spacing)
			y := gy + float64(rng.Uniform(-0.3, 0.3)*spacing)
			cols = append(cols, col{x, y})
		}
	}
	perCol := make([][]pt, len(cols))
	parallelFor(len(cols), func(i int) {
		x, y := cols[i].x, cols[i].y
		z, prev := m.maxs[2]-40, false
		var out []pt
		for z > m.mins[2]+36 {
			f := m.free(x, y, z)
			if prev && !f {
				zz := z + 6
				for zz > z && m.free(x, y, zz-1) {
					zz -= 1
				}
				out = append(out, pt{x, y, zz})
			}
			prev = f
			z -= 6
		}
		perCol[i] = out
	})
	var pts []pt
	for _, c := range perCol {
		pts = append(pts, c...)
	}
	base := len(pts)
	for _, s := range spawns {
		z := s[2]
		for m.free(s[0], s[1], z-1) && z > m.mins[2] {
			z -= 1
		}
		pts = append(pts, pt{s[0], s[1], z})
	}
	for _, it := range items {
		zz := it.z + 36
		for m.free(it.x, it.y, zz-1) && zz > m.mins[2] {
			zz -= 1
		}
		if m.free(it.x, it.y, zz) {
			pts = append(pts, pt{it.x, it.y, zz})
		}
	}

	// navigation graph: cells keyed like Python's (int(x // LINK), int(y // LINK))
	type key struct{ a, b int }
	cell := map[key][]int{}
	keys := make([]key, len(pts))
	for i, p := range pts {
		k := key{int(floorDiv(p[0], link)), int(floorDiv(p[1], link))}
		keys[i] = k
		cell[k] = append(cell[k], i)
	}
	edges := make([][]edge, len(pts))
	parallelFor(len(pts), func(i int) {
		p := pts[i]
		var es []edge
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				for _, j := range cell[key{keys[i].a + dx, keys[i].b + dy}] {
					if j == i {
						continue
					}
					q := pts[j]
					d := hypot(q[0]-p[0], q[1]-p[1])
					if d > link || q[2]-p[2] > 4*18+8 {
						continue
					}
					ok, end := m.walk(p[2]-36, p[0], p[1], q[0], q[1])
					if ok && math.Abs(end-(q[2]-36)) <= 18 {
						es = append(es, edge{j, d})
					}
				}
			}
		}
		edges[i] = es
	})

	// metrics
	var sp []int
	for i := range spawns {
		sp = append(sp, base+i)
	}
	inf := math.Inf(1)
	reach := dijkstra(edges, sp, inf)
	rev := make([][]edge, len(pts))
	for u, es := range edges {
		for _, e := range es {
			rev[e.to] = append(rev[e.to], edge{u, e.d})
		}
	}
	back := dijkstra(rev, sp, inf)
	var targets []int
	for _, it := range items {
		if strings.HasPrefix(it.cls, "weapon_") || strings.HasPrefix(it.cls, "ammo_") {
			if t := nearest(pts, it.x, it.y, it.z); t >= 0 {
				targets = append(targets, t)
			}
		}
	}
	rush := make([]bool, len(sp))
	parallelFor(len(sp), func(k int) {
		d := dijkstra(edges, []int{sp[k]}, runUnits)
		for _, t := range targets {
			if _, ok := d[t]; ok {
				rush[k] = true
				break
			}
		}
	})
	itemPts := make([]int, len(items))
	for i, it := range items {
		itemPts[i] = nearest(pts, it.x, it.y, it.z)
	}
	linksPer := make([][]bool, len(sp))
	parallelFor(len(sp), func(k int) {
		d := dijkstra(edges, []int{sp[k]}, inf)
		for _, t := range sp {
			if t != sp[k] {
				_, ok := d[t]
				linksPer[k] = append(linksPer[k], ok)
			}
		}
	})
	var links []bool
	for _, l := range linksPer {
		links = append(links, l...)
	}
	metric := func(xs []bool, empty float64) float64 {
		if len(xs) == 0 {
			return empty
		}
		return mean(xs)
	}
	var reachKeys []bool
	// Python iterates `u in back for u in reach` over dict insertion order; the mean doesn't depend on order
	for u := range reach {
		_, ok := back[u]
		reachKeys = append(reachKeys, ok)
	}
	var itemsOK []bool
	for _, p := range itemPts {
		_, ok := reach[p]
		itemsOK = append(itemsOK, p >= 0 && ok)
	}
	names := []string{"weapon_rush", "no_traps", "items", "spawn_links"}
	vals := []float64{metric(rush, 1), metric(reachKeys, 0), metric(itemsOK, 1), metric(links, 1)}
	total := ((vals[0] + vals[1]) + vals[2]) + vals[3]
	score := pyRound(100*(total/4), 1)
	metrics := map[string]float64{}
	for i, n := range names {
		metrics[n] = pyRound(vals[i], 3)
	}
	secs := pyRound(time.Since(t0).Seconds(), 1)
	fmt.Printf("satisfaction %5.1f  weapon_rush=%.2f no_traps=%.2f items=%.2f spawn_links=%.2f  (%d sampled points, %.0fs)\n",
		score, vals[0], vals[1], vals[2], vals[3], base, secs)
	for i, a := range os.Args {
		if a == "--json" && i+1 < len(os.Args) {
			out := map[string]any{"score": score, "metrics": metrics, "samples": base, "nodes": len(pts),
				"spawns": len(spawns), "items": len(items), "seconds": secs}
			b, _ := json.MarshalIndent(out, "", " ")
			if err := os.WriteFile(os.Args[i+1], b, 0o644); err != nil {
				panic(err)
			}
		}
	}
}
