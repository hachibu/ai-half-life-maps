// satisfy: the 0-100 playability score, and the holdout scenarios, on a navigation graph of sampled standable
// points (built on all cores). Its output is pinned by `make golden`; see README.md for the metrics.
//
//	satisfy build/<map>/<map>.bsp [--json out.json]                          (seed: SATISFACTION_SEED, default 0)
//	satisfy build/<map>/<map>.bsp --holdout scenarios.json|- [--json out.json]   (pass rate only)
package main

import (
	"container/heap"
	"encoding/json"
	"errors"
	"fmt"
	"halflifemaps/tools/bsp/bspfile"
	"halflifemaps/tools/checks/walker"
	"io"
	"math"
	"os"
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

// BSP is the scorer's view of a map: the shared reader (tools/bsp/bspfile) and walker (tools/checks/walker),
// plus the world bounds and entities in the form the sampling code uses.
type BSP struct {
	w          walker.Walker
	mins, maxs [3]float64
	ents       []map[string]string
}

func loadBSP(path string) *BSP {
	b, err := bspfile.Load(path)
	if err != nil {
		fail("%v", err)
	}
	m := &BSP{w: walker.Walker{M: b}}
	for i := range 3 {
		m.mins[i] = float64(b.Models[0].Mins[i])
		m.maxs[i] = float64(b.Models[0].Maxs[i])
	}
	for _, e := range b.Entities {
		m.ents = append(m.ents, e.Val)
	}
	return m
}

func (m *BSP) free(x, y, z float64) bool { return m.w.Free(x, y, z) }

// walk follows one straight segment; it returns whether it is passable and the final floor height.
func (m *BSP) walk(floorZ, ax, ay, bx, by float64) (bool, float64) {
	ok, _, end := m.w.Walk(floorZ, [][2]float64{{ax, ay}, {bx, by}})
	return ok, end
}

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
func (h *minHeap) Push(x any) {
	it, ok := x.(hitem)
	if !ok {
		panic("minHeap: pushed a non-hitem")
	}
	*h = append(*h, it)
}
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
		it, ok := heap.Pop(h).(hitem)
		if !ok {
			panic("minHeap: popped a non-hitem")
		}
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
	for range runtime.GOMAXPROCS(0) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				f(i)
			}
		}()
	}
	for i := range n {
		next <- i
	}
	close(next)
	wg.Wait()
}

func mean(xs []bool) float64 {
	s := 0.0
	for _, b := range xs {
		if b {
			s++
		}
	}
	return s / float64(len(xs))
}

// graph is the navigation graph both the score and the holdout scenarios use.
type graph struct {
	pts    []pt
	edges  [][]edge
	base   int // pts[:base] are sampled; then one node per spawn, then one per standable item
	spawns []pt
	items  []item
	sp     []int // spawn nodes
}

func buildGraph(m *BSP, seed uint64) *graph {
	rng := NewRNG(seed)
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

	// columns: one jitter pair (x then y) per column, column-major over the grid
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
					zz--
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
			z--
		}
		pts = append(pts, pt{s[0], s[1], z})
	}
	for _, it := range items {
		zz := it.z + 36
		for m.free(it.x, it.y, zz-1) && zz > m.mins[2] {
			zz--
		}
		if m.free(it.x, it.y, zz) {
			pts = append(pts, pt{it.x, it.y, zz})
		}
	}

	// navigation graph: cells keyed by (floor(x / link), floor(y / link))
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
	var sp []int
	for i := range spawns {
		sp = append(sp, base+i)
	}
	return &graph{pts: pts, edges: edges, base: base, spawns: spawns, items: items, sp: sp}
}

// score computes the four metrics and the 0-100 score.
func score(g *graph) (float64, []float64) {
	pts, edges, items, sp := g.pts, g.edges, g.items, g.sp
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
	var reachKeys []bool // the mean doesn't depend on map iteration order
	for u := range reach {
		_, ok := back[u]
		reachKeys = append(reachKeys, ok)
	}
	var itemsOK []bool
	for _, p := range itemPts {
		_, ok := reach[p]
		itemsOK = append(itemsOK, p >= 0 && ok)
	}
	vals := []float64{metric(rush, 1), metric(reachKeys, 0), metric(itemsOK, 1), metric(links, 1)}
	total := ((vals[0] + vals[1]) + vals[2]) + vals[3]
	return pyRound(100*(total/4), 1), vals
}

// scenario is one holdout check (see tools/checks/holdout.py for the format).
type scenario struct {
	From   json.RawMessage `json:"from"`   // "spawns" or [x, y, z_floor]
	To     json.RawMessage `json:"to"`     // an entity class or [x, y, z_floor]
	Within *float64        `json:"within"` // running distance limit; none = any distance
}

// Holdout scenario errors: they name the rule a scenario broke, never its contents.
var (
	errPointShape = errors.New("a point must be [x, y, z_floor]")
	errFromShape  = errors.New(`"from" must be "spawns" or [x, y, z_floor]`)
)

// point resolves [x, y, z_floor] to the nearest graph node, or -1.
func (g *graph) point(raw json.RawMessage) (int, error) {
	var v []float64
	if err := json.Unmarshal(raw, &v); err != nil || len(v) != 3 {
		return -1, errPointShape
	}
	return nearest(g.pts, v[0], v[1], v[2]), nil
}

// passes: from every start, some target is reachable within the limit (and there is at least one target).
func (g *graph) passes(s scenario) (bool, error) {
	var starts []int
	var from string
	if json.Unmarshal(s.From, &from) == nil {
		if from != "spawns" {
			return false, errFromShape
		}
		starts = g.sp
	} else {
		p, err := g.point(s.From)
		if err != nil {
			return false, err
		}
		starts = []int{p}
	}
	tgt := map[int]bool{}
	var cls string
	if json.Unmarshal(s.To, &cls) == nil {
		for _, it := range g.items {
			if it.cls == cls {
				if t := nearest(g.pts, it.x, it.y, it.z); t >= 0 {
					tgt[t] = true
				}
			}
		}
	} else {
		p, err := g.point(s.To)
		if err != nil {
			return false, err
		}
		if p >= 0 {
			tgt[p] = true
		}
	}
	if len(tgt) == 0 {
		return false, nil
	}
	limit := math.Inf(1)
	if s.Within != nil {
		limit = *s.Within
	}
	for _, st := range starts {
		if st < 0 {
			return false, nil
		}
		d := dijkstra(g.edges, []int{st}, limit)
		found := false
		for t := range tgt {
			if _, ok := d[t]; ok {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "satisfy: "+format+"\n", a...)
	os.Exit(2)
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fail("usage: satisfy map.bsp [--json out.json] [--holdout scenarios.json|-]")
	}
	var jsonOut, holdoutIn string
	for i, a := range os.Args {
		if i+1 < len(os.Args) {
			switch a {
			case "--json":
				jsonOut = os.Args[i+1]
			case "--holdout":
				holdoutIn = os.Args[i+1]
			}
		}
	}
	t0 := time.Now()
	seed := uint64(0)
	if s := os.Getenv("SATISFACTION_SEED"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			fail("SATISFACTION_SEED must be a non-negative integer")
		}
		seed = v
	}
	m := loadBSP(os.Args[1])
	g := buildGraph(m, seed)

	if holdoutIn != "" {
		// The scenarios are secret: read them (usually from stdin), print and write only the pass rate.
		var raw []byte
		var err error
		if holdoutIn == "-" {
			raw, err = io.ReadAll(os.Stdin)
		} else {
			raw, err = os.ReadFile(holdoutIn)
		}
		if err != nil {
			fail("reading holdout scenarios: %v", err)
		}
		var sc []scenario
		if err := json.Unmarshal(raw, &sc); err != nil {
			fail("holdout scenarios are not a JSON list: %v", err)
		}
		ok := 0
		for i, s := range sc {
			p, err := g.passes(s)
			if err != nil {
				fail("holdout scenario %d: %v", i, err) // the index and the rule only: never print a scenario's contents
			}
			if p {
				ok++
			}
		}
		out := map[string]any{"rate": nil, "n": len(sc)}
		if len(sc) > 0 {
			out["rate"] = pyRound(100*float64(ok)/float64(len(sc)), 1)
		}
		fmt.Printf("holdout: %v of %d scenarios\n", out["rate"], len(sc))
		if jsonOut != "" {
			writeJSON(jsonOut, out)
		}
		return
	}

	sc, vals := score(g)
	names := []string{"weapon_rush", "no_traps", "items", "spawn_links"}
	metrics := map[string]float64{}
	for i, n := range names {
		metrics[n] = pyRound(vals[i], 3)
	}
	secs := pyRound(time.Since(t0).Seconds(), 1)
	fmt.Printf("satisfaction %5.1f  weapon_rush=%.2f no_traps=%.2f items=%.2f spawn_links=%.2f  (%d sampled points, %.0fs)\n",
		sc, vals[0], vals[1], vals[2], vals[3], g.base, secs)
	if jsonOut != "" {
		writeJSON(jsonOut, map[string]any{"score": sc, "metrics": metrics, "samples": g.base, "nodes": len(g.pts),
			"spawns": len(g.spawns), "items": len(g.items), "seconds": secs})
	}
}
