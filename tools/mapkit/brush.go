package mapkit

import (
	"fmt"
	"math"
	"strings"
)

// Tex is a face's texture: a name plus optional scale and offset. Unset scales use the map's Scale for the
// name (default 1); unset offsets are 0.
type Tex struct {
	Name                       string
	SU, SV, OU, OV             float64
	hasSU, hasSV, hasOU, hasOV bool
}

// T is a plain texture by name.
func T(name string) Tex { return Tex{Name: name} }

// WithScale sets both scales.
func (t Tex) WithScale(su, sv float64) Tex {
	t.SU, t.SV, t.hasSU, t.hasSV = su, sv, true, true
	return t
}

// WithSU sets the horizontal scale.
func (t Tex) WithSU(v float64) Tex { t.SU, t.hasSU = v, true; return t }

// WithSV sets the vertical scale.
func (t Tex) WithSV(v float64) Tex { t.SV, t.hasSV = v, true; return t }

// WithOU sets the horizontal offset.
func (t Tex) WithOU(v float64) Tex { t.OU, t.hasOU = v, true; return t }

// WithOV sets the vertical offset.
func (t Tex) WithOV(v float64) Tex { t.OV, t.hasOV = v, true; return t }

// Face is one brush plane: points p with N·p = D are on it, N points out of the brush.
type Face struct {
	N   V3
	D   float64
	Tex Tex
}

// Sides overrides a box's texture per face (empty names keep the box's texture).
type Sides struct{ Top, Bottom, N, S, E, W Tex }

// Maker writes brushes for one map: Scale is the default texture scale per texture name, Sizes the pixel size of
// each texture (Fit needs it).
type Maker struct {
	Scale map[string]float64
	Sizes map[string][2]int
}

var zAxis = V3{0, 0, 1}

// Axes are the texture axes (u, v) for a face with normal n: world-aligned on floors and ceilings, horizontal u
// and downward-ish v on walls.
func Axes(n V3) (u, v V3) {
	if math.Abs(n[2]) > 0.7 {
		return V3{1, 0, 0}, V3{0, -1, 0}
	}
	u = zAxis.Cross(n).Unit()
	return u, u.Cross(n)
}

func f6(p V3) string { return F6(p[0]) + " " + F6(p[1]) + " " + F6(p[2]) }

// FaceLine is one Valve 220 plane line.
func (m *Maker) FaceLine(n V3, d float64, t Tex) string {
	n = n.Unit()
	var e1 V3
	if math.Abs(n[2]) < 0.9 {
		e1 = n.Cross(V3{0, 0, 1})
	} else {
		e1 = n.Cross(V3{1, 0, 0})
	}
	e1 = e1.Unit()
	e2 := n.Cross(e1)
	p1 := n.Mul(d)
	p0, p2 := p1.Add(e1.Mul(256)), p1.Add(e2.Mul(256))
	u, v := Axes(n)
	sc, ok := m.Scale[t.Name]
	if !ok {
		sc = 1
	}
	su, sv := sc, sc
	if t.hasSU {
		su = t.SU
	}
	if t.hasSV {
		sv = t.SV
	}
	return fmt.Sprintf("( %s ) ( %s ) ( %s ) %s [ %s %s ] [ %s %s ] 0 %s %s", f6(p0), f6(p1), f6(p2), t.Name,
		f6(u), F4(t.OU), f6(v), F4(t.OV), G(su), G(sv))
}

// Brush is a brush block from its planes.
func (m *Maker) Brush(faces []Face) string {
	lines := make([]string, len(faces))
	for i, f := range faces {
		lines[i] = m.FaceLine(f.N, f.D, f.Tex)
	}
	return "{\n" + strings.Join(lines, "\n") + "\n}\n"
}

func or(t, def Tex) Tex {
	if t.Name == "" {
		return def
	}
	return t
}

// Box is an axis-aligned brush; corners may come in any order.
func (m *Maker) Box(x0, y0, z0, x1, y1, z1 float64, tex string, sides ...Sides) string {
	return m.BoxT(x0, y0, z0, x1, y1, z1, T(tex), sides...)
}

// BoxT is Box with a full texture spec for every face.
func (m *Maker) BoxT(x0, y0, z0, x1, y1, z1 float64, tex Tex, sides ...Sides) string {
	x0, x1 = min(x0, x1), max(x0, x1)
	y0, y1 = min(y0, y1), max(y0, y1)
	z0, z1 = min(z0, z1), max(z0, z1)
	if x1-x0 <= 0.01 || y1-y0 <= 0.01 || z1-z0 <= 0.01 {
		panic(fmt.Sprintf("degenerate box (%v %v %v)-(%v %v %v)", x0, y0, z0, x1, y1, z1))
	}
	var s Sides
	if len(sides) > 0 {
		s = sides[0]
	}
	return m.Brush([]Face{{V3{0, 0, 1}, z1, or(s.Top, tex)}, {V3{0, 0, -1}, neg(z0), or(s.Bottom, tex)},
		{V3{0, 1, 0}, y1, or(s.N, tex)}, {V3{0, -1, 0}, neg(y0), or(s.S, tex)},
		{V3{1, 0, 0}, x1, or(s.E, tex)}, {V3{-1, 0, 0}, neg(x0), or(s.W, tex)}})
}

// neg is -v with zero kept positive (map coordinates are whole numbers, whose negation has no sign).
func neg(v float64) float64 {
	if v == 0 {
		return 0
	}
	return -v
}

// Beam is a square bar of thickness th from p0 to p1.
func (m *Maker) Beam(p0, p1 V3, th float64, tex string) string {
	a := p1.Sub(p0).Unit()
	hlp := zAxis
	if math.Abs(a[2]) >= 0.9 {
		hlp = V3{1, 0, 0}
	}
	side := a.Cross(hlp).Unit()
	up := side.Cross(a)
	t := T(tex)
	faces := []Face{{a, a.Dot(p1), t}, {a.Neg(), -a.Dot(p0), t}}
	for _, v := range []V3{side, side.Neg(), up, up.Neg()} {
		faces = append(faces, Face{v, v.Dot(p0) + th/2, t})
	}
	return m.Brush(faces)
}

// Hull is the convex hull of pts as a brush; faces whose normal points up more than 0.3 get top (if set).
func (m *Maker) Hull(pts []V3, tex, top string) string {
	var faces []Face
	for _, p := range HullPlanes(pts) {
		t := T(tex)
		if top != "" && p.N[2] > 0.3 {
			t = T(top)
		}
		faces = append(faces, Face{p.N, p.D, t})
	}
	return m.Brush(faces)
}

// Fit is a texture spec that stretches name's image exactly over u in [umin, umax] and z in [z0, z1] on a
// face with normal n.
func (m *Maker) Fit(name string, umin, umax, z0, z1 float64) Tex {
	sz, ok := m.Sizes[name]
	if !ok {
		panic("Fit: no size for texture " + name)
	}
	su := (umax - umin) / float64(sz[0])
	sv := (z1 - z0) / float64(sz[1])
	return T(name).WithScale(su, sv).WithOU(-umin / su).WithOV(z1 / sv)
}

// FacingNormal is the outward normal of a box face: "n", "s", "e" or "w".
func FacingNormal(facing string) V3 {
	switch facing {
	case "n":
		return V3{0, 1, 0}
	case "s":
		return V3{0, -1, 0}
	case "e":
		return V3{1, 0, 0}
	case "w":
		return V3{-1, 0, 0}
	}
	panic("bad facing " + facing)
}

// URange is the span of a box's corners along the u axis of the face with normal n.
func URange(n V3, x0, y0, x1, y1 float64) (lo, hi float64) {
	u, _ := Axes(n)
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, x := range []float64{x0, x1} {
		for _, y := range []float64{y0, y1} {
			v := V3{x, y, 0}.Dot(u)
			lo, hi = min(lo, v), max(hi, v)
		}
	}
	return lo, hi
}

// FitBox is the texture spec that fits name exactly on the facing side of a box.
func (m *Maker) FitBox(name, facing string, x0, y0, x1, y1, z0, z1 float64) Tex {
	lo, hi := URange(FacingNormal(facing), min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1))
	return m.Fit(name, lo, hi, z0, z1)
}

// SetSide returns s with the face named by facing ("top", "bottom", "n", "s", "e", "w") set to t.
func SetSide(s Sides, facing string, t Tex) Sides {
	switch facing {
	case "top":
		s.Top = t
	case "bottom":
		s.Bottom = t
	case "n":
		s.N = t
	case "s":
		s.S = t
	case "e":
		s.E = t
	case "w":
		s.W = t
	default:
		panic("bad side " + facing)
	}
	return s
}

// Sign is a thin box of back texture with name fitted exactly to its facing side.
func (m *Maker) Sign(name string, x0, y0, z0, x1, y1, z1 float64, facing, back string) string {
	x0, x1 = min(x0, x1), max(x0, x1)
	y0, y1 = min(y0, y1), max(y0, y1)
	return m.Box(x0, y0, z0, x1, y1, z1, back, SetSide(Sides{}, facing, m.FitBox(name, facing, x0, y0, x1, y1, z0, z1)))
}
