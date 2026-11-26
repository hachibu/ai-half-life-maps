// Package bspfile reads a compiled GoldSrc BSP (v30): the structures every Go check and the
// renderer need, plus the engine-limit and consistency checks the old Python reader (tools/bsp/reader.py)
// performed, with the same messages.
package bspfile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
)

// Limit is one engine limit: label (as printed), counter key, and the exclusive maximum.
type Limit struct {
	Label, Key string
	Max        int
}

// Counter keys (also the labels printed by the structure check where they read the same).
const (
	KeyLux     = "lux"
	KeyClip    = "clip"
	KeyNodes   = "nodes"
	KeyLeafs   = "leafs"
	KeyFaces   = "faces"
	KeyMarks   = "marks"
	KeyPlanes  = "planes"
	KeyTexInfo = "texinfo"
	KeyModels  = "models"
	KeyVerts   = "verts"
)

// Limits the structure check enforces. Light samples: the GoldSrc lightmap atlas holds ~1,048,576
// samples and packing wastes some, so we keep a safety margin.
var Limits = []Limit{
	{"light samples", KeyLux, 800000},
	{"clipnodes", KeyClip, 32767},
	{KeyNodes, KeyNodes, 32767},
	{KeyLeafs, KeyLeafs, 8192},
	{KeyFaces, KeyFaces, 65535},
	{"marksurfaces", KeyMarks, 65535},
	{KeyPlanes, KeyPlanes, 32767},
	{KeyTexInfo, KeyTexInfo, 8192},
	{KeyModels, KeyModels, 400},
	{"vertexes", KeyVerts, 65535},
}

// Plane is a BSP plane: normal, distance and axial type (0-2 axial, 3-5 nearest axis).
type Plane struct {
	N   [3]float64 // float32 values from the file, widened exactly
	D   float64
	Typ int32
}

// Node is an interior node of the render hull (hull 0).
type Node struct {
	Plane               int32
	Children            [2]int16
	Mins, Maxs          [3]int16
	FirstFace, NumFaces uint16
}

// TexInfo maps a face to its texture: s and t axes (xyz + offset), miptex index and flags (1 = sky).
type TexInfo struct {
	S, T   [4]float32
	Miptex int32
	Flags  int32
}

// Face is a rendered polygon: plane, edges, texinfo and lightmap offset.
type Face struct {
	Plane, Side       int16
	FirstEdge         int32
	NumEdges, TexInfo int16
	Styles            [4]byte
	LightOfs          int32
}

// ClipNode is a node of a collision hull (hulls 1-3); negative children are contents (-1 empty, -2 solid).
type ClipNode struct {
	Plane    int32
	Children [2]int16
}

// Leaf is a render-hull leaf: contents, bounds and its marksurfaces.
type Leaf struct {
	Contents, VisOfs   int32
	Mins, Maxs         [3]int16
	FirstMark, NumMark uint16
	Ambient            [4]byte
}

// Model is a brush model (0 = the world): bounds, origin, hull heads and faces.
type Model struct {
	Mins, Maxs, Origin            [3]float32
	Heads                         [4]int32
	VisLeafs, FirstFace, NumFaces int32
}

// Texture is an embedded mip texture (mip 0 pixels and palette).
type Texture struct {
	Name    string
	W, H    int
	Pixels  []byte // W*H palette indices (mip 0)
	Palette [256][3]byte
}

// Extent is a face's lightmap extent: start and size in texels, per texture axis (as the engine computes).
type Extent struct{ Min, Size [2]int }

// BSP is a loaded map with the derived data the checks use (face vertices, lightmap extents, counts, errors).
type BSP struct {
	Planes   []Plane
	Verts    [][3]float32
	Nodes    []Node
	TexInfo  []TexInfo
	Faces    []Face
	Clip     []ClipNode
	Leafs    []Leaf
	Edges    [][2]uint16
	Surf     []int32
	Marks    []uint16
	Models   []Model
	Light    []byte
	EntText  string
	Entities []Entity
	Textures []Texture
	FVerts   [][][3]float32 // per face: vertices in winding order
	FExt     []*Extent      // per face: lightmap extent (nil for sky)
	Lux      int
	Counts   map[string]int
	Errors   []string
}

// Entity keeps keys in file order (Python's dict order) and the last value for duplicate keys.
type Entity struct {
	Keys []string
	Val  map[string]string
}

// Get returns the value of key k ("" when missing).
func (e Entity) Get(k string) string { return e.Val[k] }

// Has reports whether key k is present.
func (e Entity) Has(k string) bool { _, ok := e.Val[k]; return ok }

var (
	blockRe = regexp.MustCompile(`\{([^{}]*)\}`)
	kvRe    = regexp.MustCompile(`"([^"]*)" "([^"]*)"`)
)

// ParseEntities mirrors reader.py: every {...} block, every "k" "v" pair, later duplicates win.
func ParseEntities(text string) []Entity {
	var out []Entity
	for _, b := range blockRe.FindAllStringSubmatch(text, -1) {
		e := Entity{Val: map[string]string{}}
		for _, kv := range kvRe.FindAllStringSubmatch(b[1], -1) {
			if _, ok := e.Val[kv[1]]; !ok {
				e.Keys = append(e.Keys, kv[1])
			}
			e.Val[kv[1]] = kv[2]
		}
		out = append(out, e)
	}
	return out
}

// s32 and s16 read little-endian signed fields straight from bytes (two's complement, no overflow).
func s32(b []byte) int32 { return int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24 }
func s16(b []byte) int16 { return int16(b[0]) | int16(b[1])<<8 }

// Load reads and checks a BSP file.
func Load(path string) (*BSP, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Errors Parse returns (wrapped with the lump number where one applies).
var (
	ErrNotBSP30   = errors.New("not a BSP v30 file")
	ErrLumpRange  = errors.New("out of range of the file")
	ErrLumpLength = errors.New("length is not a multiple of its record size")
)

// lumps reads the lump directory of a BSP v30 file.
type lumps []byte

func (d lumps) lump(i int) ([]byte, error) {
	o := int(s32(d[4+8*i:]))
	n := int(s32(d[8+8*i:]))
	if o < 0 || n < 0 || o+n > len(d) {
		return nil, fmt.Errorf("lump %d: %w", i, ErrLumpRange)
	}
	return d[o : o+n], nil
}

// recs returns lump i and its number of fixed-size records.
func (d lumps) recs(i, size int) ([]byte, int, error) {
	b, err := d.lump(i)
	if err != nil {
		return nil, 0, err
	}
	if len(b)%size != 0 {
		return nil, 0, fmt.Errorf("lump %d: %w (%d bytes, records of %d)", i, ErrLumpLength, len(b), size)
	}
	return b, len(b) / size, nil
}

func f32(b []byte) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b)) }

func u16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }

// recordLumps lists the fixed-size record lumps in the order Parse reads them (which decides the error a
// broken file reports first), each with the function that decodes one record into m.
var recordLumps = []struct {
	lump, size int
	read       func(m *BSP, r []byte)
}{
	{1, 20, readPlane},
	{3, 12, readVert},
	{5, 24, readNode},
	{6, 40, readTexInfo},
	{7, 20, readFace},
	{9, 8, readClip},
	{10, 28, readLeaf},
	{12, 4, func(m *BSP, r []byte) { m.Edges = append(m.Edges, [2]uint16{u16(r), u16(r[2:])}) }},
	{13, 4, func(m *BSP, r []byte) { m.Surf = append(m.Surf, s32(r)) }},
	{11, 2, func(m *BSP, r []byte) { m.Marks = append(m.Marks, u16(r)) }},
	{14, 64, readModel},
}

func readPlane(m *BSP, r []byte) {
	m.Planes = append(m.Planes, Plane{[3]float64{float64(f32(r)), float64(f32(r[4:])), float64(f32(r[8:]))}, float64(f32(r[12:])), s32(r[16:])})
}

func readVert(m *BSP, r []byte) {
	m.Verts = append(m.Verts, [3]float32{f32(r), f32(r[4:]), f32(r[8:])})
}

func readNode(m *BSP, r []byte) {
	m.Nodes = append(m.Nodes, Node{s32(r), [2]int16{s16(r[4:]), s16(r[6:])},
		[3]int16{s16(r[8:]), s16(r[10:]), s16(r[12:])}, [3]int16{s16(r[14:]), s16(r[16:]), s16(r[18:])},
		u16(r[20:]), u16(r[22:])})
}

func readTexInfo(m *BSP, r []byte) {
	var t TexInfo
	for j := range 4 {
		t.S[j] = f32(r[4*j:])
		t.T[j] = f32(r[16+4*j:])
	}
	t.Miptex, t.Flags = s32(r[32:]), s32(r[36:])
	m.TexInfo = append(m.TexInfo, t)
}

func readFace(m *BSP, r []byte) {
	m.Faces = append(m.Faces, Face{s16(r), s16(r[2:]), s32(r[4:]), s16(r[8:]), s16(r[10:]),
		[4]byte{r[12], r[13], r[14], r[15]}, s32(r[16:])})
}

func readClip(m *BSP, r []byte) {
	m.Clip = append(m.Clip, ClipNode{s32(r), [2]int16{s16(r[4:]), s16(r[6:])}})
}

func readLeaf(m *BSP, r []byte) {
	m.Leafs = append(m.Leafs, Leaf{s32(r), s32(r[4:]), [3]int16{s16(r[8:]), s16(r[10:]), s16(r[12:])},
		[3]int16{s16(r[14:]), s16(r[16:]), s16(r[18:])}, u16(r[20:]), u16(r[22:]),
		[4]byte{r[24], r[25], r[26], r[27]}})
}

func readModel(m *BSP, r []byte) {
	var md Model
	for j := range 3 {
		md.Mins[j], md.Maxs[j], md.Origin[j] = f32(r[4*j:]), f32(r[12+4*j:]), f32(r[24+4*j:])
	}
	for j := range 4 {
		md.Heads[j] = s32(r[36+4*j:])
	}
	md.VisLeafs, md.FirstFace, md.NumFaces = s32(r[52:]), s32(r[56:]), s32(r[60:])
	m.Models = append(m.Models, md)
}

// Parse decodes a BSP v30 file held in memory and runs the consistency checks.
func Parse(data []byte) (*BSP, error) {
	if len(data) < 124 || s32(data) != 30 {
		return nil, ErrNotBSP30
	}
	d := lumps(data)
	m := &BSP{}
	for _, rl := range recordLumps {
		b, n, err := d.recs(rl.lump, rl.size)
		if err != nil {
			return nil, err
		}
		for k := range n {
			rl.read(m, b[rl.size*k:])
		}
	}
	var err error
	if m.Light, err = d.lump(8); err != nil {
		return nil, err
	}
	ents, err := d.lump(0)
	if err != nil {
		return nil, err
	}
	m.EntText = strings.ToValidUTF8(string(ents), "�")
	m.Entities = ParseEntities(m.EntText)
	texLump, err := d.lump(2)
	if err != nil {
		return nil, err
	}
	m.check(texLump)
	return m, nil
}

func (m *BSP) chk(ok bool, msg string) {
	if !ok {
		m.Errors = append(m.Errors, msg)
	}
}

// check mirrors reader.py's consistency checks, texture parsing, lightmap extents and limit counts. The
// steps run in this order, which is the order errors are reported in.
func (m *BSP) check(tb []byte) {
	m.checkTree()
	m.checkGeometryRefs()
	m.readTextures(tb)
	m.checkAnimatedTextures()
	for _, f := range m.Faces {
		m.addFaceExtent(f)
	}
	m.checkLimits()
}

// checkTree: node and clip-node planes and children point at things that exist.
func (m *BSP) checkTree() {
	nn, nl, nc := len(m.Nodes), len(m.Leafs), len(m.Clip)
	for _, n := range m.Nodes {
		m.chk(n.Plane >= 0 && int(n.Plane) < len(m.Planes), "node plane")
		for _, c := range n.Children {
			if c >= 0 {
				m.chk(int(c) < nn, "node child")
			} else {
				m.chk(-1-int(c) < nl, "node child")
			}
		}
		m.chk(int(n.FirstFace)+int(n.NumFaces) <= len(m.Faces), "node faces")
	}
	for _, c := range m.Clip {
		m.chk(c.Plane >= 0 && int(c.Plane) < len(m.Planes), "clip plane")
		for _, ch := range c.Children {
			if ch >= 0 {
				m.chk(int(ch) < nc, "clip child")
			} else {
				m.chk(ch == -1 || ch == -2, "clip child")
			}
		}
	}
}

// checkGeometryRefs: faces, surfedges, edges, marksurfaces and leaves reference valid indices, and axial
// planes face the positive direction.
func (m *BSP) checkGeometryRefs() {
	for _, f := range m.Faces {
		m.chk(f.Plane >= 0 && int(f.Plane) < len(m.Planes) && f.TexInfo >= 0 && int(f.TexInfo) < len(m.TexInfo), "face refs")
		m.chk(int(f.FirstEdge)+int(f.NumEdges) <= len(m.Surf) && f.NumEdges >= 3, "face edges")
	}
	for _, s := range m.Surf {
		a := s
		if a < 0 {
			a = -a
		}
		m.chk(int(a) < len(m.Edges), "surfedge")
	}
	for _, e := range m.Edges {
		m.chk(int(e[0]) < len(m.Verts) && int(e[1]) < len(m.Verts), "edge")
	}
	for _, mk := range m.Marks {
		m.chk(int(mk) < len(m.Faces), "mark")
	}
	for _, lf := range m.Leafs {
		m.chk(int(lf.FirstMark)+int(lf.NumMark) <= len(m.Marks), "leaf marks")
	}
	for _, p := range m.Planes {
		if p.Typ < 3 && p.Typ >= 0 {
			m.chk(math.Abs(p.N[p.Typ]-1) < 1e-6, "axial plane must be positive")
		}
	}
}

// readTextures decodes the embedded textures (lump 2): name, size, pixels and palette.
func (m *BSP) readTextures(tb []byte) {
	if len(tb) < 4 {
		return
	}
	count := int(s32(tb))
	for i := 0; i < count && 4+4*i+4 <= len(tb); i++ {
		o := int(s32(tb[4+4*i:]))
		if o < 0 || o+40 > len(tb) {
			m.chk(false, "texture palette")
			continue
		}
		m.Textures = append(m.Textures, m.readTexture(tb, o))
	}
}

func (m *BSP) readTexture(tb []byte, o int) Texture {
	le := binary.LittleEndian
	name := string(tb[o : o+16])
	if z := strings.IndexByte(name, 0); z >= 0 {
		name = name[:z]
	}
	w, h := int(le.Uint32(tb[o+16:])), int(le.Uint32(tb[o+20:]))
	o0 := int(le.Uint32(tb[o+24:]))
	t := Texture{Name: name, W: w, H: h}
	if o+o0+w*h <= len(tb) {
		t.Pixels = tb[o+o0 : o+o0+w*h]
	}
	palo := o + 40 + w*h*85/64
	if palo+2+768 <= len(tb) {
		m.chk(le.Uint16(tb[palo:]) == 256, "texture palette")
		for c := range 256 {
			copy(t.Palette[c][:], tb[palo+2+3*c:palo+5+3*c])
		}
	} else {
		m.chk(false, "texture palette")
	}
	return t
}

// checkAnimatedTextures: an animated texture (+0name) needs its +1name frame.
func (m *BSP) checkAnimatedTextures() {
	names := map[string]bool{}
	for _, t := range m.Textures {
		names[t.Name] = true
	}
	for nm := range names {
		if strings.HasPrefix(nm, "+0") {
			m.chk(names["+1"+nm[2:]], fmt.Sprintf("animated texture %s is missing frame +1", nm))
		}
	}
}

// faceVerts returns a face's vertices in winding order, skipping references that don't resolve.
func (m *BSP) faceVerts(f Face) [][3]float32 {
	var vs [][3]float32
	for k := range int(f.NumEdges) {
		idx := int(f.FirstEdge) + k
		if idx >= len(m.Surf) {
			break
		}
		s := m.Surf[idx]
		var v int
		if s >= 0 && int(s) < len(m.Edges) {
			v = int(m.Edges[s][0])
		} else if s < 0 && int(-s) < len(m.Edges) {
			v = int(m.Edges[-s][1])
		}
		if v < len(m.Verts) {
			vs = append(vs, m.Verts[v])
		}
	}
	return vs
}

// addFaceExtent records a face's vertices and lightmap extent exactly as the engine computes them (nil for
// sky and broken faces), and checks the extent and lightmap size.
func (m *BSP) addFaceExtent(f Face) {
	vs := m.faceVerts(f)
	m.FVerts = append(m.FVerts, vs)
	if int(f.TexInfo) >= len(m.TexInfo) || len(vs) == 0 {
		m.FExt = append(m.FExt, nil)
		return
	}
	ti := m.TexInfo[f.TexInfo]
	if ti.Flags&1 != 0 { // TEX_SPECIAL (sky)
		if int(ti.Miptex) < len(m.Textures) {
			got := m.Textures[ti.Miptex].Name
			m.chk(got == "sky", fmt.Sprintf("sky texture must be named lowercase 'sky' (got %s)", got))
		}
		m.FExt = append(m.FExt, nil)
		return
	}
	var ex Extent
	for j := range 2 {
		ax := ti.S
		if j == 1 {
			ax = ti.T
		}
		mn, mx := float32(math.Inf(1)), float32(math.Inf(-1))
		for _, p := range vs {
			v := float32(float32(float32(p[0]*ax[0])+float32(p[1]*ax[1]))+float32(p[2]*ax[2])) + ax[3]
			mn, mx = min(mn, v), max(mx, v)
		}
		lo, hi := int(math.Floor(float64(mn/16))), int(math.Ceil(float64(mx/16)))
		ex.Min[j], ex.Size[j] = lo*16, (hi-lo)*16
		m.chk((hi-lo)*16 <= 256, "bad surface extents")
	}
	w, h := ex.Size[0]/16+1, ex.Size[1]/16+1
	m.chk(w*h <= 324, "lightmap larger than 324 samples")
	m.chk(int(f.LightOfs)+w*h*3 <= len(m.Light), "lightmap out of range")
	m.Lux += w * h
	m.FExt = append(m.FExt, &ex)
}

// checkLimits counts what the engine limits and checks each against its maximum.
func (m *BSP) checkLimits() {
	m.Counts = map[string]int{KeyLux: m.Lux, KeyClip: len(m.Clip), KeyNodes: len(m.Nodes), KeyLeafs: len(m.Leafs),
		KeyFaces: len(m.Faces), KeyMarks: len(m.Marks), KeyPlanes: len(m.Planes), KeyTexInfo: len(m.TexInfo),
		KeyModels: len(m.Models), KeyVerts: len(m.Verts)}
	for _, l := range Limits {
		m.chk(m.Counts[l.Key] < l.Max, "over engine limit: "+l.Label)
	}
}

// dot mirrors reader.py's np.dot(plane[:3], p): an FMA chain (OpenBLAS ddot).
func dot(n [3]float64, p [3]float64) float64 {
	return math.FMA(n[2], p[2], math.FMA(n[1], p[1], float64(n[0]*p[0])))
}

func (m *BSP) side(pl Plane, p [3]float64) float64 {
	if pl.Typ >= 0 && pl.Typ < 3 {
		return p[pl.Typ] - pl.D
	}
	return dot(pl.N, p) - pl.D
}

// Hull0 returns the leaf contents at p in the render hull (head = a model's Heads[0]).
func (m *BSP) Hull0(head int, p [3]float64) int {
	n := head
	for n >= 0 {
		nd := m.Nodes[n]
		if m.side(m.Planes[nd.Plane], p) >= 0 {
			n = int(nd.Children[0])
		} else {
			n = int(nd.Children[1])
		}
	}
	return int(m.Leafs[-1-n].Contents)
}

// HullC returns the contents (-1 empty, -2 solid) at p in a clip hull (head = Heads[1..3]).
func (m *BSP) HullC(head int, p [3]float64) int {
	n := head
	for n >= 0 {
		c := m.Clip[n]
		if m.side(m.Planes[c.Plane], p) >= 0 {
			n = int(c.Children[0])
		} else {
			n = int(c.Children[1])
		}
	}
	return n
}

// WorldHeads are the world model's hull heads.
func (m *BSP) WorldHeads() [4]int {
	h := m.Models[0].Heads
	return [4]int{int(h[0]), int(h[1]), int(h[2]), int(h[3])}
}

// Solid reports whether point p is inside solid in the render hull.
func (m *BSP) Solid(p [3]float64) bool { return m.Hull0(m.WorldHeads()[0], p) == -2 }

// PlayerFree reports whether a standing player's centre fits at p (clip hull 1).
func (m *BSP) PlayerFree(p [3]float64) bool { return m.HullC(m.WorldHeads()[1], p) == -1 }
