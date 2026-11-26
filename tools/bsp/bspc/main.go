// bspc: GoldSrc (BSP v30) compiler for the generated maps in this repo. A Go port of tools/bsp/compile.py
// with the same algorithms and output, parallelised with goroutines (CSG per brush, BSP subtrees, the four
// hulls, lighting per face).
//
//	bspc in.map in.wad out.bsp
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Entity struct {
	KV      [][2]string
	Brushes []Brush
}

func (e *Entity) Get(k string) string {
	v := ""
	for _, kv := range e.KV { // dict(kv): the last value wins
		if kv[0] == k {
			v = kv[1]
		}
	}
	return v
}

var faceRe = regexp.MustCompile(`^\(\s*([-\d.e ]+)\)\s*\(\s*([-\d.e ]+)\)\s*\(\s*([-\d.e ]+)\)\s*(\S+)\s*\[\s*([-\d.e ]+)\]\s*\[\s*([-\d.e ]+)\]\s*([-\d.e]+)\s+([-\d.e]+)\s+([-\d.e]+)`)
var kvRe = regexp.MustCompile(`"([^"]*)"`)

func floats(s string) []float64 {
	var out []float64
	for _, f := range strings.Fields(s) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			panic(err)
		}
		out = append(out, v)
	}
	return out
}

func parseMap(path string) []*Entity {
	fh, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	var ents []*Entity
	depth := 0
	var cur *Entity
	var brush Brush
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "{":
			depth++
			if depth == 1 {
				cur = &Entity{}
			} else {
				brush = Brush{}
			}
		case line == "}":
			if depth == 2 {
				cur.Brushes = append(cur.Brushes, brush)
			} else {
				ents = append(ents, cur)
			}
			depth--
		case strings.HasPrefix(line, `"`):
			m := kvRe.FindAllStringSubmatch(line, -1)
			cur.KV = append(cur.KV, [2]string{m[0][1], m[1][1]})
		case strings.HasPrefix(line, "("):
			m := faceRe.FindStringSubmatch(line)
			var p [3]Vec
			for i := 0; i < 3; i++ {
				f := floats(m[i+1])
				p[i] = Vec{f[0], f[1], f[2]}
			}
			n := p[0].Sub(p[1]).Cross(p[2].Sub(p[1]))
			n = n.Div(n.Norm())
			u, v := floats(m[5]), floats(m[6])
			sx, _ := strconv.ParseFloat(m[8], 64)
			sy, _ := strconv.ParseFloat(m[9], 64)
			t := &Tex{Name: m[4], S: [4]float64{u[0] / sx, u[1] / sx, u[2] / sx, u[3]}, T: [4]float64{v[0] / sy, v[1] / sy, v[2] / sy, v[3]}}
			brush = append(brush, Plane{n, n.Dot(p[1]), t})
		}
	}
	return ents
}

// ------------------------------------------------------------------ output state
type DNode struct {
	Plane               int
	Children            [2]int
	Mins, Maxs          Vec
	FirstFace, NumFaces int
}
type DLeaf struct {
	Contents   int
	Mins, Maxs Vec
	First, Num int
	Faces      []int
}
type DFace struct {
	Plane, Side, FirstEdge, NumEdges, Texinfo, LightOfs int
	Sky                                                 bool
}
type DClip struct {
	Plane    int
	Children [2]int
}
type TexInfo struct {
	S, T         [4]float64
	Miptex, Flag int
}
type Pending struct {
	Ofs  int
	P    []Vec
	N    Vec
	Name string
}

var (
	dnodes         []*DNode
	dleafs         []*DLeaf
	dfaces         []DFace
	dclip          []*DClip
	texinfos       []TexInfo
	texinfoIdx     = map[string]int{}
	verts          []Vec
	vertIdx        = map[Vec]int{}
	edges          = [][2]int{{0, 0}}
	edgeIdx        = map[[2]int]int{}
	surfedges      []int
	marksurf       []int
	lightdata      []byte
	wadtex         = map[string][]byte{}
	texNames       []string
	texLookup      = map[string]int{}
	pending        []Pending
	lightOfs       Vec
	emptyShared    = -1
	extentMismatch int
	emissive       = map[string]float64{}
)

func texIndex(name string) int {
	name = strings.ToUpper(name)
	if i, ok := texLookup[name]; ok {
		return i
	}
	texLookup[name] = len(texNames)
	texNames = append(texNames, name)
	return len(texNames) - 1
}

func texinfoIndex(t *Tex) int {
	f := func(x float64) string { return strconv.FormatFloat(npRound(x, 5)+0, 'g', -1, 64) }
	key := strings.ToUpper(t.Name)
	for _, x := range t.S {
		key += "|" + f(x)
	}
	for _, x := range t.T {
		key += "|" + f(x)
	}
	if i, ok := texinfoIdx[key]; ok {
		return i
	}
	flag := 0
	if strings.HasPrefix(strings.ToUpper(t.Name), "SKY") {
		flag = 1
	}
	texinfoIdx[key] = len(texinfos)
	texinfos = append(texinfos, TexInfo{t.S, t.T, texIndex(t.Name), flag})
	return len(texinfos) - 1
}

func vertIndex(p Vec) int {
	k := npRoundVec(p, 2)
	k = Vec{k[0] + 0, k[1] + 0, k[2] + 0}
	if i, ok := vertIdx[k]; ok {
		return i
	}
	vertIdx[k] = len(verts)
	verts = append(verts, npRoundVec(p, 2))
	return len(verts) - 1
}

func edgeRef(a, b int) int {
	if i, ok := edgeIdx[[2]int{b, a}]; ok {
		return -i
	}
	if i, ok := edgeIdx[[2]int{a, b}]; ok {
		return i
	}
	edgeIdx[[2]int{a, b}] = len(edges)
	edges = append(edges, [2]int{a, b})
	return len(edges) - 1
}

func subdivide(poly Poly, t *Tex) []Poly {
	const limit = 240.0
	var out []Poly
	stack := []Poly{poly}
	for len(stack) > 0 {
		w := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		done := true
		for _, ax := range [][4]float64{t.S, t.T} {
			v := Vec{ax[0], ax[1], ax[2]}
			off := ax[3]
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, p := range w {
				x := p.Dot(v) + off
				lo, hi = math.Min(lo, x), math.Max(hi, x)
			}
			base := math.Floor(lo/16) * 16
			if base+limit < hi-0.01 {
				cut := base + limit
				L := v.Norm()
				f, b, _ := split(w, v.Div(L), (cut-off)/L)
				if f != nil && b != nil {
					stack = append(stack, f, b)
					done = false
					break
				}
			}
		}
		if done {
			out = append(out, w)
		}
	}
	return out
}

func pyRoundInt(x float64) float64 { return math.RoundToEven(x) }

func inv3(m [3][3]float64) [3][3]float64 {
	// LU with partial pivoting, like LAPACK's dgesv(A, I) used by numpy.linalg.inv.
	a := m
	piv := [3]int{0, 1, 2}
	for k := 0; k < 3; k++ {
		p := k
		for i := k + 1; i < 3; i++ {
			if math.Abs(a[i][k]) > math.Abs(a[p][k]) {
				p = i
			}
		}
		a[k], a[p] = a[p], a[k]
		piv[k], piv[p] = piv[p], piv[k]
		for i := k + 1; i < 3; i++ {
			a[i][k] /= a[k][k]
			for j := k + 1; j < 3; j++ {
				a[i][j] -= a[i][k] * a[k][j]
			}
		}
	}
	var inv [3][3]float64
	for col := 0; col < 3; col++ {
		var b [3]float64
		for i := 0; i < 3; i++ {
			if piv[i] == col {
				b[i] = 1
			}
		}
		for i := 0; i < 3; i++ {
			for j := 0; j < i; j++ {
				b[i] -= a[i][j] * b[j]
			}
		}
		for i := 2; i >= 0; i-- {
			for j := i + 1; j < 3; j++ {
				b[i] -= a[i][j] * b[j]
			}
			b[i] /= a[i][i]
		}
		for i := 0; i < 3; i++ {
			inv[i][col] = b[i]
		}
	}
	return inv
}

func emitFace(poly Poly, n Vec, d float64, tex *Tex) int {
	c := mean(poly)
	up := Vec{0, 0, 1}
	if math.Abs(n[2]) >= 0.9 {
		up = Vec{1, 0, 0}
	}
	r := up.Cross(n)
	r = r.Div(r.Norm())
	u := n.Cross(r)
	ang := make([]float64, len(poly))
	for i, p := range poly {
		q := p.Sub(c)
		ang[i] = math.Atan2(q.Dot(u), q.Dot(r))
	}
	order := argsortStable(ang)
	sorted := make(Poly, len(poly))
	for i, j := range order {
		sorted[len(poly)-1-i] = poly[j] // CCW -> CW
	}
	vids := make([]int, len(sorted))
	for i, p := range sorted {
		vids[i] = vertIndex(p)
	}
	var dedup []int
	for i, v := range vids {
		prev := vids[(i-1+len(vids))%len(vids)]
		if v != prev {
			dedup = append(dedup, v)
		}
	}
	pidx, flip := planeIndex(n, d)
	side := 0
	if flip {
		side = 1
	}
	first := len(surfedges)
	for i := range dedup {
		surfedges = append(surfedges, edgeRef(dedup[i], dedup[(i+1)%len(dedup)]))
	}
	if strings.HasPrefix(strings.ToUpper(tex.Name), "SKY") {
		dfaces = append(dfaces, DFace{pidx, side, first, len(dedup), texinfoIndex(tex), -1, true})
		return len(dfaces) - 1
	}
	vs := make([]Vec, len(dedup))
	for i, v := range dedup {
		vs[i] = verts[v]
	}
	base := tex
	var t *Tex
	var ex [2][2]int
	var sv, tv Vec
	var so, to float64
	ok := false
	for _, nudge := range []float64{0, 0.05, -0.05, 0.11, -0.11, 0.23, -0.23, 0.37} {
		t = &Tex{Name: base.Name, S: base.S, T: base.T}
		t.S[3] += nudge
		t.T[3] += nudge
		sv, tv = Vec{t.S[0], t.S[1], t.S[2]}, Vec{t.T[0], t.T[1], t.T[2]}
		so, to = t.S[3], t.T[3]
		ok = true
		for k, ax := range []struct {
			v   Vec
			off float64
		}{{sv, so}, {tv, to}} {
			mn, mx := math.Inf(1), math.Inf(-1)
			mn32, mx32 := float32(math.Inf(1)), float32(math.Inf(-1))
			v32 := [3]float32{float32(ax.v[0]), float32(ax.v[1]), float32(ax.v[2])}
			for _, p := range vs {
				x := p.DotM(ax.v) + ax.off // numpy vs @ vec (gemv)
				mn, mx = math.Min(mn, x), math.Max(mx, x)
				x32 := float32(float32(float32(float32(p[0])*v32[0])+float32(float32(p[1])*v32[1]))+float32(float32(p[2])*v32[2])) + float32(ax.off)
				if x32 < mn32 {
					mn32 = x32
				}
				if x32 > mx32 {
					mx32 = x32
				}
			}
			a, b := int(math.Floor(mn/16)), int(math.Ceil(mx/16))
			a32, b32 := int(math.Floor(float64(mn32)/16)), int(math.Ceil(float64(mx32)/16))
			margin := math.Min(math.Abs(mn/16-pyRoundInt(mn/16)), math.Abs(mx/16-pyRoundInt(mx/16)))
			if a != a32 || b != b32 || (margin > 0 && margin < 1e-3) {
				ok = false
			}
			ex[k] = [2]int{a, b}
		}
		if ok {
			break
		}
	}
	if !ok {
		extentMismatch++
	}
	ti := texinfoIndex(t)
	smn, smx, tmn, tmx := ex[0][0], ex[0][1], ex[1][0], ex[1][1]
	if (smx-smn)*16 > 256 || (tmx-tmn)*16 > 256 {
		panic("bad extents")
	}
	w, h := smx-smn+1, tmx-tmn+1
	Minv := inv3([3][3]float64{{sv[0], sv[1], sv[2]}, {tv[0], tv[1], tv[2]}, {n[0], n[1], n[2]}})
	P := make([]Vec, 0, w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			rhs := Vec{float64(i+smn)*16 - so, float64(j+tmn)*16 - to, d}
			p := Vec{rhs.Dot(Vec(Minv[0])), rhs.Dot(Vec(Minv[1])), rhs.Dot(Vec(Minv[2]))} // numpy rhs @ Minv.T
			p = p.Add(c.Sub(p).Mul(0.02))
			P = append(P, p.Add(lightOfs))
		}
	}
	lofs := len(lightdata)
	lightdata = append(lightdata, make([]byte, w*h*3)...)
	pending = append(pending, Pending{lofs, P, n, strings.ToUpper(t.Name)})
	dfaces = append(dfaces, DFace{pidx, side, first, len(dedup), ti, lofs, false})
	return len(dfaces) - 1
}

// ------------------------------------------------------------------ writing trees
func writeHull0(root *Node, isWorld bool) int {
	type fn struct {
		idx  int
		node *Node
	}
	var faceNodes []fn
	var rec func(*Node) int
	rec = func(node *Node) int {
		if node.Leaf {
			if node.Solid {
				return -1
			}
			if !isWorld {
				return -(emptyShared + 1)
			}
			dleafs = append(dleafs, &DLeaf{Contents: -1, Mins: node.Mins, Maxs: node.Maxs})
			node.LeafIndex = len(dleafs) - 1
			return -(node.LeafIndex + 1)
		}
		idx := len(dnodes)
		dnodes = append(dnodes, nil)
		pidx, flip := planeIndex(node.N, node.D)
		cf, cb := rec(node.Front), rec(node.Back)
		ch := [2]int{cf, cb}
		if flip {
			ch = [2]int{cb, cf}
		}
		dnodes[idx] = &DNode{Plane: pidx, Children: ch, Mins: node.Mins, Maxs: node.Maxs}
		faceNodes = append(faceNodes, fn{idx, node})
		return idx
	}
	root0 := rec(root)
	sort.SliceStable(faceNodes, func(a, b int) bool { return faceNodes[a].idx < faceNodes[b].idx })
	for _, x := range faceNodes {
		first := len(dfaces)
		for _, f := range x.node.On {
			for _, piece := range subdivide(f.P, f.Tex) {
				fi := emitFace(piece, f.N, f.D, f.Tex)
				if isWorld {
					side := x.node.Back
					if f.N.Dot(x.node.N) > 0 {
						side = x.node.Front
					}
					markLeaves(side, piece, fi)
				}
			}
		}
		dnodes[x.idx].FirstFace, dnodes[x.idx].NumFaces = first, len(dfaces)-first
	}
	return root0
}

func markLeaves(node *Node, poly Poly, fi int) {
	if node.Leaf {
		if !node.Solid {
			dleafs[node.LeafIndex].Faces = append(dleafs[node.LeafIndex].Faces, fi)
		}
		return
	}
	f, b, st := split(poly, node.N, node.D)
	if st == stOn {
		markLeaves(node.Front, poly, fi)
		return
	}
	if f != nil {
		markLeaves(node.Front, f, fi)
	}
	if b != nil {
		markLeaves(node.Back, b, fi)
	}
}

func writeClip(root *Node) int {
	var rec func(*Node) int
	rec = func(node *Node) int {
		if node.Leaf {
			if node.Solid {
				return -2
			}
			return -1
		}
		idx := len(dclip)
		dclip = append(dclip, nil)
		pidx, flip := planeIndex(node.N, node.D)
		cf, cb := rec(node.Front), rec(node.Back)
		ch := [2]int{cf, cb}
		if flip {
			ch = [2]int{cb, cf}
		}
		dclip[idx] = &DClip{pidx, ch}
		return idx
	}
	r := rec(root)
	if r < 0 {
		pidx, _ := planeIndex(Vec{0, 0, 1}, 1e6)
		dclip = append(dclip, &DClip{pidx, [2]int{r, r}})
		r = len(dclip) - 1
	}
	return r
}

// ------------------------------------------------------------------ models
var HULLS = [3][2]Vec{{{-16, -16, -36}, {16, 16, 36}}, {{-32, -32, -32}, {32, 32, 32}}, {{-16, -16, -18}, {16, 16, 18}}}

type Model struct {
	Mins, Maxs                    Vec
	Heads                         [4]int
	Visleafs, FirstFace, NumFaces int
}

var WMIN, WMAX Vec

func outsideWorld(p Vec) bool {
	for i := 0; i < 3; i++ {
		if p[i] < WMIN[i]+EPS || p[i] > WMAX[i]-EPS {
			return true
		}
	}
	return false
}

func compileModel(brushes []Brush, isWorld, noclip bool) Model {
	var mn, mx Vec
	var solid SolidFn
	set := newBrushSet(brushes)
	if isWorld {
		mn, mx = WMIN, WMAX
		solid = func(p Vec) bool { return outsideWorld(p) || set.inside(p) }
	} else {
		var pts []Vec
		for _, b := range brushes {
			pts = append(pts, windingPoints(brushWindings(b))...)
		}
		mn, mx = bounds(pts)
		solid = set.inside
	}
	hasFaces := false
	for _, b := range brushes {
		for _, p := range b {
			if strings.ToUpper(p.Tex.Name) != "AAATRIGGER" {
				hasFaces = true
			}
		}
	}
	bmn, bmx := mn, mx
	if !isWorld {
		bmn, bmx = mn.Sub(Vec{64, 64, 64}), mx.Add(Vec{64, 64, 64})
	}
	// build all hull trees concurrently (pure); write them afterwards in the original order
	trees := make([]*Node, 4)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vis := clipToBox(visibleFaces(brushes, solid), bmn, bmx)
		trees[0] = buildTree(vis, boxRegion(bmn, bmx), solid)
	}()
	if !noclip {
		for h := 0; h < 3; h++ {
			wg.Add(1)
			go func(h int) {
				defer wg.Done()
				eb := make([]Brush, len(brushes))
				for i, b := range brushes {
					eb[i] = expandBrush(b, HULLS[h][0], HULLS[h][1])
				}
				es := newBrushSet(eb)
				var hs SolidFn
				var hmn, hmx Vec
				if isWorld {
					hs = func(p Vec) bool { return outsideWorld(p) || es.inside(p) }
					hmn, hmx = WMIN, WMAX
				} else {
					hs = es.inside
					hmn, hmx = mn.Sub(Vec{128, 128, 128}), mx.Add(Vec{128, 128, 128})
				}
				hv := clipToBox(visibleFaces(eb, hs), hmn, hmx)
				trees[h+1] = buildTree(hv, boxRegion(hmn, hmx), hs)
			}(h)
		}
	}
	wg.Wait()
	tree := trees[0]
	if !hasFaces {
		var strip func(*Node)
		strip = func(nd *Node) {
			if !nd.Leaf {
				nd.On = nil
				strip(nd.Front)
				strip(nd.Back)
			}
		}
		strip(tree)
	}
	firstFace := len(dfaces)
	leafStart := len(dleafs)
	m := Model{Mins: mn, Maxs: mx, FirstFace: firstFace}
	m.Heads[0] = writeHull0(tree, isWorld)
	m.NumFaces = len(dfaces) - firstFace
	if noclip {
		pidx, _ := planeIndex(Vec{0, 0, 1}, 1e6)
		dclip = append(dclip, &DClip{pidx, [2]int{-1, -1}})
		for h := 1; h < 4; h++ {
			m.Heads[h] = len(dclip) - 1
		}
		return m
	}
	for h := 1; h < 4; h++ {
		m.Heads[h] = writeClip(trees[h])
	}
	if isWorld {
		m.Visleafs = len(dleafs) - leafStart
	}
	return m
}

// pyG mirrors Python's "%g" formatting.
func pyG(x float64) string { return strconv.FormatFloat(x, 'g', 6, 64) }

// ------------------------------------------------------------------ main
func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: bspc in.map in.wad out.bsp")
		os.Exit(2)
	}
	MAP, WAD, OUT := os.Args[1], os.Args[2], os.Args[3]
	t0 := time.Now()
	ents := parseMap(MAP)
	world := ents[0]
	for _, kv := range world.KV {
		if kv[0] == "_emissive" && kv[1] != "" {
			for _, item := range strings.Split(kv[1], ",") {
				p := strings.SplitN(item, ":", 2)
				v, _ := strconv.ParseFloat(p[1], 64)
				emissive[strings.ToUpper(p[0])] = v
			}
		}
	}
	var all []Vec
	for _, b := range world.Brushes {
		all = append(all, windingPoints(brushWindings(b))...)
	}
	WMIN, WMAX = bounds(all)

	dleafs = append(dleafs, &DLeaf{Contents: -2})
	wad, err := os.ReadFile(WAD)
	if err != nil {
		panic(err)
	}
	num := int(binary.LittleEndian.Uint32(wad[4:8]))
	ofs := int(binary.LittleEndian.Uint32(wad[8:12]))
	for i := 0; i < num; i++ {
		e := wad[ofs+32*i : ofs+32*(i+1)]
		pos := int(binary.LittleEndian.Uint32(e[0:4]))
		dsz := int(binary.LittleEndian.Uint32(e[4:8]))
		name := string(bytes.SplitN(e[16:32], []byte{0}, 2)[0])
		wadtex[strings.ToUpper(name)] = wad[pos : pos+dsz]
	}

	fmt.Println("world...")
	models := []Model{compileModel(world.Brushes, true, false)}
	dleafs = append(dleafs, &DLeaf{Contents: -1})
	emptyShared = len(dleafs) - 1

	var entText strings.Builder
	for _, e := range ents {
		var kv [][2]string
		for _, p := range e.KV {
			if p[0] != "wad" {
				kv = append(kv, p)
			}
		}
		if e == world {
			kv = append(kv, [2]string{"wad", ""})
		} else if len(e.Brushes) > 0 {
			cls := e.Get("classname")
			fmt.Println(cls, "...")
			brs := e.Brushes
			var org, rest []Brush
			for _, b := range brs {
				isOrg := true
				for _, p := range b {
					if strings.ToUpper(p.Tex.Name) != "ORIGIN" {
						isOrg = false
					}
				}
				if isOrg {
					org = append(org, b)
				} else {
					rest = append(rest, b)
				}
			}
			if len(org) > 0 {
				var pts []Vec
				for _, b := range org {
					pts = append(pts, windingPoints(brushWindings(b))...)
				}
				mn, mx := bounds(pts)
				oc := npRoundVec(mn.Add(mx).Div(2), 3)
				brs = nil
				for _, b := range rest {
					nb := make(Brush, len(b))
					for i, p := range b {
						t := *p.Tex
						t.S[3] = t.S[3] + Vec{t.S[0], t.S[1], t.S[2]}.Dot(oc)
						t.T[3] = t.T[3] + Vec{t.T[0], t.T[1], t.T[2]}.Dot(oc)
						nb[i] = Plane{p.N, p.D - p.N.Dot(oc), &t}
					}
					brs = append(brs, nb)
				}
				var nkv [][2]string
				for _, p := range kv {
					if p[0] != "origin" {
						nkv = append(nkv, p)
					}
				}
				kv = append(nkv, [2]string{"origin", pyG(oc[0]) + " " + pyG(oc[1]) + " " + pyG(oc[2])})
				lightOfs = oc
			}
			noclip := cls == "func_illusionary" || cls == "func_rotating"
			models = append(models, compileModel(brs, false, noclip))
			lightOfs = Vec{}
			kv = append(kv, [2]string{"model", fmt.Sprintf("*%d", len(models)-1)})
		}
		entText.WriteString("{\n")
		for _, p := range kv {
			entText.WriteString(`"` + p[0] + `" "` + p[1] + `"` + "\n")
		}
		entText.WriteString("}\n")
	}
	entities := append([]byte(entText.String()), 0)

	lights := collectLights(ents)
	fmt.Printf("geometry done %.1fs, %d faces, %d lights\n", time.Since(t0).Seconds(), len(dfaces), len(lights))
	finalizeLighting(world.Brushes, lights)

	for _, lf := range dleafs {
		lf.First, lf.Num = len(marksurf), len(lf.Faces)
		marksurf = append(marksurf, lf.Faces...)
	}
	out := pack(entities, models)
	if err := os.WriteFile(OUT, out, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("planes", len(dplanes), "nodes", len(dnodes), "leafs", len(dleafs), "faces", len(dfaces),
		"clipnodes", len(dclip), "texinfo", len(texinfos), "verts", len(verts), "edges", len(edges),
		"light bytes", len(lightdata), "models", len(models))
	fmt.Println("extent float32 mismatches:", extentMismatch)
	fmt.Println("bytes", len(out))
	fmt.Printf("total %.1fs\n", time.Since(t0).Seconds())
}

func s16(v float64) int16 {
	if v < -32768 {
		return -32768
	}
	if v > 32767 {
		return 32767
	}
	return int16(v)
}

func pack(entities []byte, models []Model) []byte {
	le := binary.LittleEndian
	var L [15][]byte
	w := func(i int, vals ...any) {
		var b bytes.Buffer
		for _, v := range vals {
			binary.Write(&b, le, v)
		}
		L[i] = append(L[i], b.Bytes()...)
	}
	L[0] = entities
	for _, p := range dplanes {
		w(1, float32(p.N[0]), float32(p.N[1]), float32(p.N[2]), float32(p.D), int32(p.Typ))
	}
	for _, nm := range append([]string(nil), texNames...) {
		if strings.HasPrefix(nm, "+0") {
			for k := 1; k < 10; k++ {
				fr := fmt.Sprintf("+%d%s", k, nm[2:])
				if _, ok := wadtex[fr]; ok {
					if _, ok2 := texLookup[fr]; !ok2 {
						texIndex(fr)
					}
				}
			}
		}
	}
	hdr := 4 + 4*len(texNames)
	w(2, int32(len(texNames)))
	pos := hdr
	for _, nm := range texNames {
		w(2, int32(pos))
		pos += len(wadtex[nm])
	}
	for _, nm := range texNames {
		L[2] = append(L[2], wadtex[nm]...)
	}
	for _, v := range verts {
		w(3, float32(v[0]), float32(v[1]), float32(v[2]))
	}
	fl := func(x float64) int16 { return s16(math.Floor(x)) }
	cl := func(x float64) int16 { return s16(math.Ceil(x)) }
	for _, n := range dnodes {
		w(5, int32(n.Plane), int16(n.Children[0]), int16(n.Children[1]), fl(n.Mins[0]), fl(n.Mins[1]), fl(n.Mins[2]),
			cl(n.Maxs[0]), cl(n.Maxs[1]), cl(n.Maxs[2]), uint16(n.FirstFace), uint16(n.NumFaces))
	}
	for _, t := range texinfos {
		w(6, float32(t.S[0]), float32(t.S[1]), float32(t.S[2]), float32(t.S[3]),
			float32(t.T[0]), float32(t.T[1]), float32(t.T[2]), float32(t.T[3]), int32(t.Miptex), int32(t.Flag))
	}
	for _, f := range dfaces {
		st := [4]byte{0, 255, 255, 255}
		if f.Sky {
			st = [4]byte{255, 255, 255, 255}
		}
		w(7, int16(f.Plane), int16(f.Side), int32(f.FirstEdge), int16(f.NumEdges), int16(f.Texinfo), st, int32(f.LightOfs))
	}
	L[8] = lightdata
	for _, c := range dclip {
		w(9, int32(c.Plane), int16(c.Children[0]), int16(c.Children[1]))
	}
	for _, lf := range dleafs {
		w(10, int32(lf.Contents), int32(-1), fl(lf.Mins[0]), fl(lf.Mins[1]), fl(lf.Mins[2]),
			cl(lf.Maxs[0]), cl(lf.Maxs[1]), cl(lf.Maxs[2]), uint16(lf.First), uint16(lf.Num), [4]byte{})
	}
	for _, m := range marksurf {
		w(11, uint16(m))
	}
	for _, e := range edges {
		w(12, uint16(e[0]), uint16(e[1]))
	}
	for _, s := range surfedges {
		w(13, int32(s))
	}
	for _, m := range models {
		w(14, float32(m.Mins[0]), float32(m.Mins[1]), float32(m.Mins[2]), float32(m.Maxs[0]), float32(m.Maxs[1]), float32(m.Maxs[2]),
			float32(0), float32(0), float32(0), int32(m.Heads[0]), int32(m.Heads[1]), int32(m.Heads[2]), int32(m.Heads[3]),
			int32(m.Visleafs), int32(m.FirstFace), int32(m.NumFaces))
	}
	out := make([]byte, 4+15*8)
	var info [30]int32
	for i, data := range L {
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
		info[2*i], info[2*i+1] = int32(len(out)), int32(len(data))
		out = append(out, data...)
	}
	le.PutUint32(out[0:4], 30)
	for i, v := range info {
		le.PutUint32(out[4+4*i:8+4*i], uint32(v))
	}
	return out
}
