// bspc: GoldSrc (BSP v30) compiler for the generated maps in this repo, parallelised with goroutines (CSG per
// brush, BSP subtrees, the four hulls, lighting per face). Its output is pinned by `make golden`; see README.md.
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
	for f := range strings.FieldsSeq(s) {
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
	defer func() { _ = fh.Close() }() // read-only: a failed close loses nothing
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
			for i := range 3 {
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

// ------------------------------------------------------------------ output state.
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
	var kb strings.Builder
	kb.WriteString(strings.ToUpper(t.Name))
	for _, x := range t.S {
		kb.WriteString("|" + f(x))
	}
	for _, x := range t.T {
		kb.WriteString("|" + f(x))
	}
	key := kb.String()
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
	for k := range 3 {
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
	for col := range 3 {
		var b [3]float64
		for i := range 3 {
			if piv[i] == col {
				b[i] = 1
			}
		}
		for i := range 3 {
			for j := range i {
				b[i] -= a[i][j] * b[j]
			}
		}
		for i := 2; i >= 0; i-- {
			for j := i + 1; j < 3; j++ {
				b[i] -= a[i][j] * b[j]
			}
			b[i] /= a[i][i]
		}
		for i := range 3 {
			inv[i][col] = b[i]
		}
	}
	return inv
}

// faceWinding sorts a face's points clockwise around their centre (seen from the front), registers the
// vertices, and drops consecutive duplicates. It returns the centre and the vertex indices.
func faceWinding(poly Poly, n Vec) (Vec, []int) {
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
	return c, dedup
}

// texFit is a face's texture projection after nudging, with its lightmap extents in 16-unit cells.
type texFit struct {
	t      *Tex
	ex     [2][2]int // [axis][min, max]
	sv, tv Vec
	so, to float64
}

// axisExtent projects the face's points onto one texture axis and returns the extent in 16-unit cells. ok is
// false when the engine's float32 arithmetic would round the extent differently, or a bound sits within
// 1e-3 of a cell edge.
func axisExtent(vs []Vec, v Vec, off float64) (lo, hi int, ok bool) {
	mn, mx := math.Inf(1), math.Inf(-1)
	mn32, mx32 := float32(math.Inf(1)), float32(math.Inf(-1))
	v32 := [3]float32{float32(v[0]), float32(v[1]), float32(v[2])}
	for _, p := range vs {
		x := p.DotM(v) + off // numpy vs @ vec (gemv)
		mn, mx = math.Min(mn, x), math.Max(mx, x)
		x32 := float32(float32(float32(float32(p[0])*v32[0])+float32(float32(p[1])*v32[1]))+float32(float32(p[2])*v32[2])) + float32(off)
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
	ok = !(a != a32 || b != b32 || (margin > 0 && margin < 1e-3))
	return a, b, ok
}

// fitTexture tries small texture-offset nudges until the lightmap extents come out the same in float64 and in
// the engine's float32 (counting a miss in extentMismatch when none does).
func fitTexture(vs []Vec, base *Tex) texFit {
	var f texFit
	ok := false
	for _, nudge := range []float64{0, 0.05, -0.05, 0.11, -0.11, 0.23, -0.23, 0.37} {
		f.t = &Tex{Name: base.Name, S: base.S, T: base.T}
		f.t.S[3] += nudge
		f.t.T[3] += nudge
		f.sv, f.tv = Vec{f.t.S[0], f.t.S[1], f.t.S[2]}, Vec{f.t.T[0], f.t.T[1], f.t.T[2]}
		f.so, f.to = f.t.S[3], f.t.T[3]
		ok = true
		for k, ax := range []struct {
			v   Vec
			off float64
		}{{f.sv, f.so}, {f.tv, f.to}} {
			a, b, axOK := axisExtent(vs, ax.v, ax.off)
			if !axOK {
				ok = false
			}
			f.ex[k] = [2]int{a, b}
		}
		if ok {
			break
		}
	}
	if !ok {
		extentMismatch++
	}
	return f
}

// lightSamples returns the world position of every lightmap sample (pulled 2% toward the face centre c).
func lightSamples(f texFit, n Vec, d float64, c Vec) []Vec {
	smn, smx, tmn, tmx := f.ex[0][0], f.ex[0][1], f.ex[1][0], f.ex[1][1]
	if (smx-smn)*16 > 256 || (tmx-tmn)*16 > 256 {
		panic("bad extents")
	}
	w, h := smx-smn+1, tmx-tmn+1
	sv, tv := f.sv, f.tv
	Minv := inv3([3][3]float64{{sv[0], sv[1], sv[2]}, {tv[0], tv[1], tv[2]}, {n[0], n[1], n[2]}})
	P := make([]Vec, 0, w*h)
	for j := range h {
		for i := range w {
			rhs := Vec{float64(i+smn)*16 - f.so, float64(j+tmn)*16 - f.to, d}
			p := Vec{rhs.Dot(Vec(Minv[0])), rhs.Dot(Vec(Minv[1])), rhs.Dot(Vec(Minv[2]))} // numpy rhs @ Minv.T
			p = p.Add(c.Sub(p).Mul(0.02))
			P = append(P, p.Add(lightOfs))
		}
	}
	return P
}

func emitFace(poly Poly, n Vec, d float64, tex *Tex) int {
	c, dedup := faceWinding(poly, n)
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
	f := fitTexture(vs, tex)
	ti := texinfoIndex(f.t)
	P := lightSamples(f, n, d, c)
	lofs := len(lightdata)
	lightdata = append(lightdata, make([]byte, len(P)*3)...)
	pending = append(pending, Pending{lofs, P, n, strings.ToUpper(f.t.Name)})
	dfaces = append(dfaces, DFace{pidx, side, first, len(dedup), ti, lofs, false})
	return len(dfaces) - 1
}

// ------------------------------------------------------------------ writing trees.
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

// ------------------------------------------------------------------ models.
var HULLS = [3][2]Vec{{{-16, -16, -36}, {16, 16, 36}}, {{-32, -32, -32}, {32, 32, 32}}, {{-16, -16, -18}, {16, 16, 18}}}

type Model struct {
	Mins, Maxs                    Vec
	Heads                         [4]int
	Visleafs, FirstFace, NumFaces int
}

var WMIN, WMAX Vec

func outsideWorld(p Vec) bool {
	for i := range 3 {
		if p[i] < WMIN[i]+EPS || p[i] > WMAX[i]-EPS {
			return true
		}
	}
	return false
}

// modelBounds returns a model's bounds and its solidity test (the world also counts outside the world box).
func modelBounds(brushes []Brush, isWorld bool, set *brushSet) (Vec, Vec, SolidFn) {
	if isWorld {
		return WMIN, WMAX, func(p Vec) bool { return outsideWorld(p) || set.inside(p) }
	}
	var pts []Vec
	for _, b := range brushes {
		pts = append(pts, windingPoints(brushWindings(b))...)
	}
	mn, mx := bounds(pts)
	return mn, mx, set.inside
}

// hasVisibleFaces: some brush side is textured with something other than AAATRIGGER.
func hasVisibleFaces(brushes []Brush) bool {
	hasFaces := false
	for _, b := range brushes {
		for _, p := range b {
			if strings.ToUpper(p.Tex.Name) != "AAATRIGGER" {
				hasFaces = true
			}
		}
	}
	return hasFaces
}

// clipHullTree builds clip hull h's tree: brushes expanded by the hull's player box.
func clipHullTree(brushes []Brush, h int, isWorld bool, mn, mx Vec) *Node {
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
	return buildTree(hv, boxRegion(hmn, hmx), hs)
}

// buildHullTrees builds the render tree and (unless noclip) the three clip hull trees concurrently. They are
// pure; the caller writes them afterwards in the original order.
func buildHullTrees(brushes []Brush, isWorld, noclip bool, mn, mx Vec, solid SolidFn) []*Node {
	bmn, bmx := mn, mx
	if !isWorld {
		bmn, bmx = mn.Sub(Vec{64, 64, 64}), mx.Add(Vec{64, 64, 64})
	}
	trees := make([]*Node, 4)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		vis := clipToBox(visibleFaces(brushes, solid), bmn, bmx)
		trees[0] = buildTree(vis, boxRegion(bmn, bmx), solid)
	}()
	if !noclip {
		for h := range 3 {
			wg.Add(1)
			go func(h int) {
				defer wg.Done()
				trees[h+1] = clipHullTree(brushes, h, isWorld, mn, mx)
			}(h)
		}
	}
	wg.Wait()
	return trees
}

// stripFaces drops every node's faces (a model with nothing visible, like a trigger).
func stripFaces(nd *Node) {
	if !nd.Leaf {
		nd.On = nil
		stripFaces(nd.Front)
		stripFaces(nd.Back)
	}
}

func compileModel(brushes []Brush, isWorld, noclip bool) Model {
	set := newBrushSet(brushes)
	mn, mx, solid := modelBounds(brushes, isWorld, set)
	hasFaces := hasVisibleFaces(brushes)
	trees := buildHullTrees(brushes, isWorld, noclip, mn, mx, solid)
	tree := trees[0]
	if !hasFaces {
		stripFaces(tree)
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

// ------------------------------------------------------------------ main.

// readEmissive reads the worldspawn's `_emissive` list ("TEXNAME:brightness,...").
func readEmissive(world *Entity) {
	for _, kv := range world.KV {
		if kv[0] == "_emissive" && kv[1] != "" {
			for item := range strings.SplitSeq(kv[1], ",") {
				p := strings.SplitN(item, ":", 2)
				v, _ := strconv.ParseFloat(p[1], 64)
				emissive[strings.ToUpper(p[0])] = v
			}
		}
	}
}

// loadWad indexes the WAD's textures by upper-case name.
func loadWad(path string) {
	wad, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	num := int(binary.LittleEndian.Uint32(wad[4:8]))
	ofs := int(binary.LittleEndian.Uint32(wad[8:12]))
	for i := range num {
		e := wad[ofs+32*i : ofs+32*(i+1)]
		pos := int(binary.LittleEndian.Uint32(e[0:4]))
		dsz := int(binary.LittleEndian.Uint32(e[4:8]))
		name := string(bytes.SplitN(e[16:32], []byte{0}, 2)[0])
		wadtex[strings.ToUpper(name)] = wad[pos : pos+dsz]
	}
}

// moveToOrigin handles a brush entity's ORIGIN brushes: the rest of its brushes (and their texture offsets)
// are moved so the origin brushes' centre becomes (0, 0, 0), and that centre becomes the entity's origin.
// It returns the brushes to compile and the updated key/values, or the inputs unchanged without ORIGIN brushes.
func moveToOrigin(brs []Brush, kv [][2]string) ([]Brush, [][2]string) {
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
	if len(org) == 0 {
		return brs, kv
	}
	var pts []Vec
	for _, b := range org {
		pts = append(pts, windingPoints(brushWindings(b))...)
	}
	mn, mx := bounds(pts)
	oc := npRoundVec(mn.Add(mx).Div(2), 3)
	var moved []Brush
	for _, b := range rest {
		nb := make(Brush, len(b))
		for i, p := range b {
			t := *p.Tex
			t.S[3] += Vec{t.S[0], t.S[1], t.S[2]}.Dot(oc)
			t.T[3] += Vec{t.T[0], t.T[1], t.T[2]}.Dot(oc)
			nb[i] = Plane{p.N, p.D - p.N.Dot(oc), &t}
		}
		moved = append(moved, nb)
	}
	var nkv [][2]string
	for _, p := range kv {
		if p[0] != "origin" {
			nkv = append(nkv, p)
		}
	}
	nkv = append(nkv, [2]string{"origin", pyG(oc[0]) + " " + pyG(oc[1]) + " " + pyG(oc[2])})
	lightOfs = oc
	return moved, nkv
}

// compileEntities compiles every brush entity into a model (after the world, models[0]) and returns the
// entity lump text, NUL-terminated.
func compileEntities(ents []*Entity, models *[]Model) []byte {
	world := ents[0]
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
			var brs []Brush
			brs, kv = moveToOrigin(e.Brushes, kv)
			noclip := cls == "func_illusionary" || cls == "func_rotating"
			*models = append(*models, compileModel(brs, false, noclip))
			lightOfs = Vec{}
			kv = append(kv, [2]string{"model", fmt.Sprintf("*%d", len(*models)-1)})
		}
		entText.WriteString("{\n")
		for _, p := range kv {
			entText.WriteString(`"` + p[0] + `" "` + p[1] + `"` + "\n")
		}
		entText.WriteString("}\n")
	}
	return append([]byte(entText.String()), 0)
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: bspc in.map in.wad out.bsp")
		os.Exit(2)
	}
	MAP, WAD, OUT := os.Args[1], os.Args[2], os.Args[3]
	t0 := time.Now()
	ents := parseMap(MAP)
	world := ents[0]
	readEmissive(world)
	var all []Vec
	for _, b := range world.Brushes {
		all = append(all, windingPoints(brushWindings(b))...)
	}
	WMIN, WMAX = bounds(all)

	dleafs = append(dleafs, &DLeaf{Contents: -2})
	loadWad(WAD)

	fmt.Println("world...")
	models := []Model{compileModel(world.Brushes, true, false)}
	dleafs = append(dleafs, &DLeaf{Contents: -1})
	emptyShared = len(dleafs) - 1
	entities := compileEntities(ents, &models)

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

// The BSP format stores indices and counts in fixed-width fields. A value that doesn't fit would wrap silently
// and write a corrupt map, so the compile stops instead (Python's struct.pack raised here too). Engine limits
// (AGENTS.md) keep real maps far below these.
func overflow(v int, typ string) string {
	return fmt.Sprintf("BSP field overflow: %d does not fit in %s", v, typ)
}

func i32(v int) int32 {
	if v < math.MinInt32 || v > math.MaxInt32 {
		panic(overflow(v, "int32"))
	}
	return int32(v)
}

func i16(v int) int16 {
	if v < math.MinInt16 || v > math.MaxInt16 {
		panic(overflow(v, "int16"))
	}
	return int16(v)
}

func u16(v int) uint16 {
	if v < 0 || v > math.MaxUint16 {
		panic(overflow(v, "uint16"))
	}
	return uint16(v)
}

func u32(v int) uint32 {
	if v < 0 || v > math.MaxUint32 {
		panic(overflow(v, "uint32"))
	}
	return uint32(v)
}

// lumpWriter collects the 15 BSP lumps as little-endian bytes.
type lumpWriter [15][]byte

// w appends fixed-width values to lump i.
func (lw *lumpWriter) w(i int, vals ...any) {
	var b bytes.Buffer
	for _, v := range vals {
		// Writing to a bytes.Buffer only fails for a type binary.Write can't encode; skipping it
		// would silently shift every later field in the lump, so stop instead.
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	lw[i] = append(lw[i], b.Bytes()...)
}

func fl(x float64) int16 { return s16(math.Floor(x)) }
func cl(x float64) int16 { return s16(math.Ceil(x)) }

// textures writes lump 2, after adding the missing frames of animated textures (+1name, +2name, ...).
func (lw *lumpWriter) textures() {
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
	lw.w(2, i32(len(texNames)))
	pos := hdr
	for _, nm := range texNames {
		lw.w(2, i32(pos))
		pos += len(wadtex[nm])
	}
	for _, nm := range texNames {
		lw[2] = append(lw[2], wadtex[nm]...)
	}
}

// tree writes the render tree, texinfo and faces: lumps 5 (nodes), 6 (texinfo) and 7 (faces).
func (lw *lumpWriter) tree() {
	for _, n := range dnodes {
		lw.w(5, i32(n.Plane), i16(n.Children[0]), i16(n.Children[1]), fl(n.Mins[0]), fl(n.Mins[1]), fl(n.Mins[2]),
			cl(n.Maxs[0]), cl(n.Maxs[1]), cl(n.Maxs[2]), u16(n.FirstFace), u16(n.NumFaces))
	}
	for _, t := range texinfos {
		lw.w(6, float32(t.S[0]), float32(t.S[1]), float32(t.S[2]), float32(t.S[3]),
			float32(t.T[0]), float32(t.T[1]), float32(t.T[2]), float32(t.T[3]), i32(t.Miptex), i32(t.Flag))
	}
	for _, f := range dfaces {
		st := [4]byte{0, 255, 255, 255}
		if f.Sky {
			st = [4]byte{255, 255, 255, 255}
		}
		lw.w(7, i16(f.Plane), i16(f.Side), i32(f.FirstEdge), i16(f.NumEdges), i16(f.Texinfo), st, i32(f.LightOfs))
	}
}

// leaves writes lumps 9 (clip nodes), 10 (leafs), 11 (marksurfaces), 12 (edges) and 13 (surfedges).
func (lw *lumpWriter) leaves() {
	for _, c := range dclip {
		lw.w(9, i32(c.Plane), i16(c.Children[0]), i16(c.Children[1]))
	}
	for _, lf := range dleafs {
		lw.w(10, i32(lf.Contents), int32(-1), fl(lf.Mins[0]), fl(lf.Mins[1]), fl(lf.Mins[2]),
			cl(lf.Maxs[0]), cl(lf.Maxs[1]), cl(lf.Maxs[2]), u16(lf.First), u16(lf.Num), [4]byte{})
	}
	for _, m := range marksurf {
		lw.w(11, u16(m))
	}
	for _, e := range edges {
		lw.w(12, u16(e[0]), u16(e[1]))
	}
	for _, s := range surfedges {
		lw.w(13, i32(s))
	}
}

// file lays the lumps out 4-byte aligned after the header and fills in the lump directory.
func (lw *lumpWriter) file() []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 4+15*8)) // header: version + 15 lump directory entries, filled in below
	var info [30]uint32
	for i, data := range lw {
		for buf.Len()%4 != 0 {
			buf.WriteByte(0)
		}
		info[2*i], info[2*i+1] = u32(buf.Len()), u32(len(data))
		buf.Write(data)
	}
	out := buf.Bytes()
	le := binary.LittleEndian
	le.PutUint32(out[0:4], 30)
	for i, v := range info {
		le.PutUint32(out[4+4*i:8+4*i], v)
	}
	return out
}

func pack(entities []byte, models []Model) []byte {
	var lw lumpWriter
	lw[0] = entities
	for _, p := range dplanes {
		lw.w(1, float32(p.N[0]), float32(p.N[1]), float32(p.N[2]), float32(p.D), i32(p.Typ))
	}
	lw.textures()
	for _, v := range verts {
		lw.w(3, float32(v[0]), float32(v[1]), float32(v[2]))
	}
	lw.tree()
	lw[8] = lightdata
	lw.leaves()
	for _, m := range models {
		lw.w(14, float32(m.Mins[0]), float32(m.Mins[1]), float32(m.Mins[2]), float32(m.Maxs[0]), float32(m.Maxs[1]), float32(m.Maxs[2]),
			float32(0), float32(0), float32(0), i32(m.Heads[0]), i32(m.Heads[1]), i32(m.Heads[2]), i32(m.Heads[3]),
			i32(m.Visleafs), i32(m.FirstFace), i32(m.NumFaces))
	}
	return lw.file()
}
