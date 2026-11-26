// Package renderer is the software preview renderer for compiled BSPs (Go port of the old Python renderer, same
// pixels): textures, baked lightmaps, skybox, additive brushes, animated sprites and moving brush entities
// (rotating fans, sliding doors via Dyn). It approximates the game (no texture filtering); it's for previews
// and videos.
//
// Floating point follows numpy's operation order exactly (see dot/dotM below), so images match the Python
// renderer pixel for pixel.
package renderer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"halflifemaps/tools/bspfile"
)

// dot = numpy 1-D a @ b and the rows of A @ M (gemm): fma(a2,b2, fma(a1,b1, a0*b0)).
func dot(a, b [3]float64) float64 {
	return math.FMA(a[2], b[2], math.FMA(a[1], b[1], float64(a[0]*b[0])))
}

// dotM = numpy rows @ vector (gemv, also for 3-D arrays): fma(a2,b2, fma(a0,b0, a1*b1)).
func dotM(a, b [3]float64) float64 {
	return math.FMA(a[2], b[2], math.FMA(a[0], b[0], float64(a[1]*b[1])))
}

// norm3 = numpy.linalg.norm on a 1-D 3-vector: sqrt(a @ a).
func norm3(a [3]float64) float64 { return math.Sqrt(dot(a, a)) }

// normRow = numpy.linalg.norm(x, axis=-1) on 3-vectors: sqrt((x0^2 + x1^2) + x2^2).
func normRow(a [3]float64) float64 {
	return math.Sqrt(float64(float64(a[0]*a[0])+float64(a[1]*a[1])) + float64(a[2]*a[2]))
}

func sub(a, b [3]float64) [3]float64 { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func cross(a, b [3]float64) [3]float64 {
	return [3]float64{float64(a[1]*b[2]) - float64(a[2]*b[1]), float64(a[2]*b[0]) - float64(a[0]*b[2]), float64(a[0]*b[1]) - float64(a[1]*b[0])}
}

// pyMod = Python's integer % (result has the sign of the divisor).
func pyMod(a, n int) int { return ((a % n) + n) % n }

// Transform moves a brush model: world = R * local + T (rows of R as in numpy).
type Transform struct {
	R [3][3]float64
	T [3]float64
}

// Identity is the identity rotation.
var Identity = [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

type sprite struct {
	origin [3]float64
	scale  float64
	frames []sprFrame
	amt    float64
	rate   float64
}

type sprFrame struct {
	ox, oy, w, h int
	pix          [][3]float64 // h*w colours
}

type lightmap struct {
	smin, tmin, w, h int
	rgb              [][3]float64 // h*w
}

// Renderer holds a loaded map and everything precomputed per face.
type Renderer struct {
	M      *bspfile.BSP
	sky    map[string]image3
	sprite []sprite
	add    map[int]float64
	// MInfo maps a brush model index to its entity's keys; MOrigin to the model's origin (entities with one).
	MInfo   map[int]map[string]string
	MOrigin map[int][3]float64
	morder  []int // models with an origin, in entity order (Python dict order)
	lverts  [][][3]float64
	cent0   [][3]float64
	rad     []float64
	n       [][3]float64
	d       []float64
	lm      []*lightmap
}

type image3 struct {
	w, h int
	px   [][3]float64
}

// New loads a BSP and the assets the game would load from assetRoot (gfx/env/<sky>XX.tga, sprites).
func New(bspPath, assetRoot string) (*Renderer, error) {
	m, err := bspfile.Load(bspPath)
	if err != nil {
		return nil, err
	}
	r := &Renderer{M: m, sky: map[string]image3{}, add: map[int]float64{}, MInfo: map[int]map[string]string{}, MOrigin: map[int][3]float64{}}
	skyname := ""
	if len(m.Entities) > 0 {
		skyname = m.Entities[0].Get("skyname")
	}
	for _, k := range []string{"rt", "lf", "bk", "ft", "up", "dn"} {
		im, err := readTGA(filepath.Join(assetRoot, "gfx", "env", skyname+k+".tga"))
		if err != nil {
			im = image3{4, 4, make([][3]float64, 16)}
		}
		r.sky[k] = im
	}
	sprCache := map[string][]sprFrame{}
	for _, e := range m.Entities {
		if e.Get("classname") == "env_sprite" && e.Get("model") != "" {
			path := filepath.Join(assetRoot, e.Get("model"))
			if _, err := os.Stat(path); err != nil {
				continue
			}
			if _, ok := sprCache[path]; !ok {
				fr, err := readSPR(path)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", path, err)
				}
				sprCache[path] = fr
			}
			o := vec3(e.Get("origin"))
			r.sprite = append(r.sprite, sprite{o, numOr(e, "scale", 1), sprCache[path], numOr(e, "renderamt", 255) / 255, numOr(e, "framerate", 10)})
		}
		if mi, ok := modelIndex(e.Get("model")); ok && e.Get("rendermode") == "5" && mi < len(m.Models) {
			md := m.Models[mi]
			for fi := int(md.FirstFace); fi < int(md.FirstFace+md.NumFaces); fi++ {
				r.add[fi] = numOr(e, "renderamt", 255) / 255.0
			}
		}
	}
	for _, e := range m.Entities {
		if mi, ok := modelIndex(e.Get("model")); ok {
			r.MInfo[mi] = e.Val
			if e.Has("origin") {
				o := vec3(e.Get("origin"))
				if _, seen := r.MOrigin[mi]; !seen {
					r.morder = append(r.morder, mi)
				}
				r.MOrigin[mi] = o
			}
		}
	}
	F := len(m.Faces)
	r.lverts, r.cent0, r.rad = make([][][3]float64, F), make([][3]float64, F), make([]float64, F)
	r.n, r.d, r.lm = make([][3]float64, F), make([]float64, F), make([]*lightmap, F)
	for i, fv := range m.FVerts {
		// numpy mean / norm on the float32 vertex array: float32 arithmetic, summed in order
		var c [3]float32
		for _, v := range fv {
			for k := range 3 {
				c[k] += v[k]
			}
		}
		for k := range 3 {
			c[k] /= float32(len(fv))
		}
		rmax := float32(0)
		lv := make([][3]float64, len(fv))
		for j, v := range fv {
			dx, dy, dz := v[0]-c[0], v[1]-c[1], v[2]-c[2]
			nn := float32(math.Sqrt(float64(float32(float32(dx*dx)+float32(dy*dy)) + float32(dz*dz))))
			rmax = max(rmax, nn)
			lv[j] = [3]float64{float64(v[0]), float64(v[1]), float64(v[2])}
		}
		r.lverts[i] = lv
		r.cent0[i] = [3]float64{float64(c[0]), float64(c[1]), float64(c[2])}
		r.rad[i] = float64(rmax)
	}
	for i, fc := range m.Faces {
		pl := m.Planes[fc.Plane]
		nv, dv := pl.N, pl.D
		if fc.Side != 0 {
			nv, dv = [3]float64{-nv[0], -nv[1], -nv[2]}, -dv
		}
		r.n[i], r.d[i] = nv, dv
		ex := m.FExt[i]
		if ex == nil {
			continue
		}
		lw, lh := ex.Size[0]/16+1, ex.Size[1]/16+1
		lm := &lightmap{smin: ex.Min[0], tmin: ex.Min[1], w: lw, h: lh, rgb: make([][3]float64, lw*lh)}
		o := int(fc.LightOfs)
		for k := 0; k < lw*lh && o+3*k+2 < len(m.Light); k++ {
			lm.rgb[k] = [3]float64{float64(m.Light[o+3*k]), float64(m.Light[o+3*k+1]), float64(m.Light[o+3*k+2])}
		}
		r.lm[i] = lm
	}
	return r, nil
}

func modelIndex(s string) (int, bool) {
	rest, ok := strings.CutPrefix(s, "*")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

func numOr(e bspfile.Entity, k string, def float64) float64 {
	if !e.Has(k) {
		return def
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(e.Get(k)), 64)
	if err != nil {
		return def
	}
	return v
}

// vec3 parses "x y z"; missing or bad numbers read as 0, like a default origin.
func vec3(s string) [3]float64 {
	f := strings.Fields(s)
	var v [3]float64
	for i := 0; i < len(v) && i < len(f); i++ {
		if x, err := strconv.ParseFloat(f[i], 64); err == nil {
			v[i] = x
		}
	}
	return v
}

// Solid reports whether point p is inside solid (render hull), e.g. a camera inside a wall.
func (r *Renderer) Solid(p [3]float64) bool { return r.M.Solid(p) }

// Frame renders one image (RGB, row-major, W*H*3 bytes). dyn moves brush models (by model index); models with
// an origin and no entry in dyn stay at their origin.
func (r *Renderer) Frame(eye, target [3]float64, t float64, width, height int, fov float64, dyn map[int]Transform) []byte {
	W, H := width, height
	m := r.M
	F := len(m.Faces)
	nw := slices.Clone(r.n)
	dw := slices.Clone(r.d)
	cent := slices.Clone(r.cent0)
	wverts := map[int][][3]float64{}
	ftr := map[int]Transform{}
	order := slices.Clone(r.morder)
	trans := map[int]Transform{}
	for _, mi := range r.morder {
		trans[mi] = Transform{Identity, r.MOrigin[mi]}
	}
	for mi, tr := range dyn {
		if _, ok := trans[mi]; !ok {
			order = append(order, mi)
		}
		trans[mi] = tr
	}
	for _, mi := range order {
		tr := trans[mi]
		if mi >= len(m.Models) {
			continue
		}
		md := m.Models[mi]
		for fi := int(md.FirstFace); fi < int(md.FirstFace+md.NumFaces) && fi < F; fi++ {
			vw := make([][3]float64, len(r.lverts[fi]))
			var c [3]float64
			for j, v := range r.lverts[fi] {
				// v @ R.T + T
				vw[j] = [3]float64{dot(v, tr.R[0]) + tr.T[0], dot(v, tr.R[1]) + tr.T[1], dot(v, tr.R[2]) + tr.T[2]}
				for k := range 3 {
					c[k] += vw[j][k]
				}
			}
			for k := range 3 {
				c[k] /= float64(len(vw))
			}
			wverts[fi], ftr[fi] = vw, tr
			nn := [3]float64{dotM(tr.R[0], r.n[fi]), dotM(tr.R[1], r.n[fi]), dotM(tr.R[2], r.n[fi])} // R @ n
			nw[fi], dw[fi], cent[fi] = nn, r.d[fi]+dot(nn, tr.T), c
		}
	}
	fwd := sub(target, eye)
	nf := norm3(fwd)
	fwd = [3]float64{fwd[0] / nf, fwd[1] / nf, fwd[2] / nf}
	right := cross(fwd, [3]float64{0, 0, 1})
	nr := norm3(right)
	right = [3]float64{right[0] / nr, right[1] / nr, right[2] / nr}
	up := cross(right, fwd)
	hw, hh := float64(W)/2, float64(H)/2
	f := hw / math.Tan(fov/2*(math.Pi/180))
	zbuf := make([]float64, W*H)
	for i := range zbuf {
		zbuf[i] = math.Inf(1)
	}
	img := make([][3]float64, W*H)

	// bulk culling: backfaces, behind the camera, outside the view cone
	halfdiag := math.Atan(hypot(hw, hh) / f)
	for fi := range F {
		rel := sub(cent[fi], eye)
		cz := dotM(rel, fwd)
		if !(dotM(nw[fi], eye)-dw[fi] > 0) || !(cz > -r.rad[fi]) {
			continue
		}
		dist := normRow(rel) + 1e-6
		ang := math.Acos(clip(cz/dist, -1, 1)) - math.Asin(clip(r.rad[fi]/dist, 0, 1))
		if !(ang < halfdiag || dist < r.rad[fi]+4) {
			continue
		}
		r.drawFace(fi, wverts, ftr, nw[fi], dw[fi], eye, fwd, right, up, f, W, H, zbuf, img)
	}
	r.drawSprites(t, eye, fwd, right, up, f, W, H, zbuf, img)
	out := make([]byte, W*H*3)
	for i, c := range img {
		for k := range 3 {
			out[3*i+k] = byte(clip(c[k], 0, 255))
		}
	}
	return out
}

func clip(x, lo, hi float64) float64 { return math.Min(math.Max(x, lo), hi) }

func (r *Renderer) drawFace(fi int, wverts map[int][][3]float64, ftr map[int]Transform, n [3]float64, d float64,
	eye, fwd, right, up [3]float64, f float64, width, height int, zbuf []float64, img [][3]float64) {
	W, H := width, height
	m := r.M
	vs, moved := wverts[fi]
	if !moved {
		vs = r.lverts[fi]
	}
	k := len(vs)
	if k == 0 {
		return
	}
	// clip the polygon to the near plane
	dd := make([]float64, k)
	anyNeg, allNeg := false, true
	for i, v := range vs {
		dd[i] = dotM(sub(v, eye), fwd) - 1.0
		if dd[i] < 0 {
			anyNeg = true
		} else {
			allNeg = false
		}
	}
	if allNeg {
		return
	}
	cv := vs
	if anyNeg {
		cv = nil
		for i := range k {
			a, b := vs[i], vs[(i+1)%k]
			da, db := dd[i], dd[(i+1)%k]
			if da >= 0 {
				cv = append(cv, a)
			}
			if (da >= 0) != (db >= 0) {
				tt := da / (da - db)
				cv = append(cv, [3]float64{a[0] + float64((b[0]-a[0])*tt), a[1] + float64((b[1]-a[1])*tt), a[2] + float64((b[2]-a[2])*tt)})
			}
		}
	}
	hw, hh := float64(W)/2, float64(H)/2
	sxMin, sxMax, syMin, syMax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, v := range cv {
		rv := sub(v, eye)
		cz := dotM(rv, fwd)
		sx := hw + float64(f*dotM(rv, right))/cz
		sy := hh - float64(f*dotM(rv, up))/cz
		sxMin, sxMax = math.Min(sxMin, sx), math.Max(sxMax, sx)
		syMin, syMax = math.Min(syMin, sy), math.Max(syMax, sy)
	}
	x0, x1 := int(math.Max(0, sxMin)), int(math.Min(float64(W-1), sxMax+1))
	y0, y1 := int(math.Max(0, syMin)), int(math.Min(float64(H-1), syMax+1))
	if x0 > x1 || y0 > y1 {
		return
	}
	ti := m.TexInfo[m.Faces[fi].TexInfo]
	tex := &m.Textures[ti.Miptex]
	isSky := tex.Name == "sky"
	cutout := strings.HasPrefix(tex.Name, "{") // GoldSrc: palette index 255 of a {texture is transparent
	amt, additive := r.add[fi]
	tr, hasTr := ftr[fi]
	lm := r.lm[fi]
	ne := dot(n, eye)
	sv := [3]float64{float64(ti.S[0]), float64(ti.S[1]), float64(ti.S[2])}
	tv := [3]float64{float64(ti.T[0]), float64(ti.T[1]), float64(ti.T[2])}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			ax, ay := float64(x)-hw, float64(y)-hh
			var D [3]float64
			for c := range 3 {
				D[c] = float64(float64(fwd[c]*f)+float64(right[c]*ax)) - float64(up[c]*ay)
			}
			tt := (d - ne) / dotM(D, n)
			if !(tt > 0) || math.IsInf(tt, 0) || math.IsNaN(tt) {
				continue
			}
			P := [3]float64{eye[0] + float64(D[0]*tt), eye[1] + float64(D[1]*tt), eye[2] + float64(D[2]*tt)}
			inside := true
			for i := range k {
				a, b := vs[i], vs[(i+1)%k]
				if !(dotM(cross(sub(b, a), sub(P, a)), n) <= 1e-3) {
					inside = false
					break
				}
			}
			idx := y*W + x
			if !inside || !(tt < zbuf[idx]) {
				continue
			}
			if isSky {
				zbuf[idx] = tt
				img[idx] = r.skyColour(D)
				continue
			}
			pl := P
			if hasTr { // texture / lightmap coordinates in the model's own space: (P - T) @ R
				q := sub(P, tr.T)
				pl = [3]float64{dot(q, [3]float64{tr.R[0][0], tr.R[1][0], tr.R[2][0]}),
					dot(q, [3]float64{tr.R[0][1], tr.R[1][1], tr.R[2][1]}),
					dot(q, [3]float64{tr.R[0][2], tr.R[1][2], tr.R[2][2]})}
			}
			s := dotM(pl, sv) + float64(ti.S[3])
			tq := dotM(pl, tv) + float64(ti.T[3])
			col, hole := shade(tex, lm, s, tq, cutout)
			if hole {
				continue // a hole in a see-through ({name) texture: whatever is behind shows instead
			}
			if additive {
				for c := range 3 {
					img[idx][c] += float64(col[c] * amt)
				}
				continue
			}
			zbuf[idx] = tt
			img[idx] = col
		}
	}
}

// skyColour mirrors the Python sky lookup for one view ray.
func (r *Renderer) skyColour(dir [3]float64) [3]float64 {
	nn := normRow(dir)
	x, y, z := dir[0]/nn, dir[1]/nn, dir[2]/nn
	ax := 0
	a := [3]float64{math.Abs(x), math.Abs(y), math.Abs(z)}
	if a[1] > a[ax] {
		ax = 1
	}
	if a[2] > a[ax] {
		ax = 2
	}
	var face string
	var ss, tt float64
	switch {
	case ax == 0 && x > 0:
		face, ss, tt = "rt", -y/x, z/x
	case ax == 0:
		face, ss, tt = "lf", y/-x, z/-x
	case ax == 1 && y > 0:
		face, ss, tt = "bk", x/y, z/y
	case ax == 1:
		face, ss, tt = "ft", -x/-y, z/-y
	case z > 0:
		face, ss, tt = "up", -y/z, -x/z
	default:
		face, ss, tt = "dn", -y/-z, x/-z
	}
	im := r.sky[face]
	N := float64(im.h)
	row := int(clip(math.Trunc((1-tt)/2*N), 0, float64(im.h-1)))
	col := int(clip(math.Trunc((ss+1)/2*N), 0, float64(im.h-1)))
	if math.IsNaN(tt) || math.IsNaN(ss) {
		row, col = 0, 0
	}
	if col >= im.w {
		col = im.w - 1
	}
	return im.px[row*im.w+col]
}

func (r *Renderer) drawSprites(t float64, eye, fwd, right, up [3]float64, f float64, width, height int, zbuf []float64, img [][3]float64) {
	W, H := width, height
	hw, hh := float64(W)/2, float64(H)/2
	for _, s := range r.sprite {
		fr := s.frames[pyMod(int(t*s.rate), len(s.frames))]
		rel := sub(s.origin, eye)
		czs := dot(rel, fwd)
		if czs < 16 {
			continue
		}
		sx := hw + float64(f*dot(rel, right))/czs
		sy := hh - float64(f*dot(rel, up))/czs
		k := float64(s.scale*f) / czs
		xa, xb := int(sx+float64(float64(fr.ox)*k)), int(sx+float64(float64(fr.ox+fr.w)*k))
		ya, yb := int(sy-float64(float64(fr.oy)*k)), int(sy+float64(float64(fr.h-fr.oy)*k))
		if xb <= 0 || ya >= H || xa >= W || yb <= 0 || xb-xa < 1 || yb-ya < 1 {
			continue
		}
		cx0, cy0, cx1, cy1 := max(0, xa), max(0, ya), min(W, xb), min(H, yb)
		for y := cy0; y < cy1; y++ {
			vv := min(max(int(float64(float64(y-ya)/float64(yb-ya))*float64(fr.h)), 0), fr.h-1)
			for x := cx0; x < cx1; x++ {
				idx := y*W + x
				if !(zbuf[idx]*f > czs) {
					continue
				}
				uu := min(max(int(float64(float64(x-xa)/float64(xb-xa))*float64(fr.w)), 0), fr.w-1)
				c := fr.pix[vv*fr.w+uu]
				for ch := range 3 {
					img[idx][ch] += float64(c[ch] * s.amt)
				}
			}
		}
	}
}

// hypot matches Python's correctly rounded math.hypot for the image half-sizes used here: they are small
// integers or halves, so x*x + y*y is exact and Sqrt rounds it correctly.
func hypot(x, y float64) float64 { return math.Sqrt(x*x + y*y) }

// Asset readers.

var (
	errShortTGA       = errors.New("short TGA")
	errUnsupportedTGA = errors.New("unsupported TGA")
	errShortSprite    = errors.New("short sprite")
)

func readTGA(path string) (image3, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return image3{}, err
	}
	if len(b) < 18 {
		return image3{}, errShortTGA
	}
	idLen, typ := int(b[0]), b[2]
	w, h := int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:]))
	bpp, desc := int(b[16]), b[17]
	if typ != 2 || (bpp != 24 && bpp != 32) {
		return image3{}, fmt.Errorf("%w (type %d, %d bpp)", errUnsupportedTGA, typ, bpp)
	}
	px := b[18+idLen:]
	step := bpp / 8
	if len(px) < w*h*step {
		return image3{}, fmt.Errorf("%w: pixel data", errShortTGA)
	}
	im := image3{w, h, make([][3]float64, w*h)}
	topDown := desc&0x20 != 0
	for y := range h {
		src := y
		if !topDown {
			src = h - 1 - y
		}
		for x := range w {
			o := (src*w + x) * step
			im.px[y*w+x] = [3]float64{float64(px[o+2]), float64(px[o+1]), float64(px[o])}
		}
	}
	return im, nil
}

func readSPR(path string) ([]sprFrame, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 42 {
		return nil, errShortSprite
	}
	le := binary.LittleEndian
	nf := int(le.Uint32(b[28:]))
	o := 40
	ncol := int(le.Uint16(b[o:]))
	o += 2
	if o+3*ncol > len(b) {
		return nil, fmt.Errorf("%w: palette", errShortSprite)
	}
	pal := b[o : o+3*ncol]
	o += 3 * ncol
	var frames []sprFrame
	for range nf {
		o += 4
		if o+16 > len(b) {
			return nil, fmt.Errorf("%w: frame header", errShortSprite)
		}
		ox := int(int32(le.Uint32(b[o:])))   //nolint:gosec // signed sprite offsets stored as two's complement
		oy := int(int32(le.Uint32(b[o+4:]))) //nolint:gosec // as above
		w, h := int(le.Uint32(b[o+8:])), int(le.Uint32(b[o+12:]))
		o += 16
		if o+w*h > len(b) {
			return nil, fmt.Errorf("%w: pixels", errShortSprite)
		}
		fr := sprFrame{ox, oy, w, h, make([][3]float64, w*h)}
		for k := range w * h {
			c := int(b[o+k])
			if 3*c+2 < len(pal) {
				fr.pix[k] = [3]float64{float64(pal[3*c]), float64(pal[3*c+1]), float64(pal[3*c+2])}
			}
		}
		o += w * h
		frames = append(frames, fr)
	}
	return frames, nil
}

// shade is a face's colour at texture coordinates (s, t): the texel lit by the lightmap. hole reports a
// transparent texel of a see-through ({name) texture, where whatever is behind should show instead.
func shade(tex *bspfile.Texture, lm *lightmap, s, t float64, cutout bool) (col [3]float64, hole bool) {
	si, tj := pyMod(int(s), tex.W), pyMod(int(t), tex.H)
	pi := tex.Pixels[tj*tex.W+si]
	if cutout && pi == 255 {
		return col, true
	}
	pal := tex.Palette[pi]
	if lm != nil {
		li := min(max(int(math.RoundToEven((s-float64(lm.smin))/16)), 0), lm.w-1)
		lj := min(max(int(math.RoundToEven((t-float64(lm.tmin))/16)), 0), lm.h-1)
		lc := lm.rgb[lj*lm.w+li]
		for c := range 3 {
			col[c] = float64(float64(pal[c])*lc[c]) / 128.0
		}
	}
	return col, false
}
