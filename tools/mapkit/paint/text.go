package paint

import "math"

type seg struct{ x0, y0, x1, y1 float64 }

// outline is the text's outline as line segments in pixel space (y down), with the baseline's left end at
// (x, baseline).
func (fc *Face) outline(s string, x, baseline float64) []seg {
	sc := fc.scale()
	var segs []seg
	pen := x
	for _, r := range s {
		g := fc.glyph(r)
		for _, ct := range fc.F.contours(g, 0) {
			segs = append(segs, flatten(ct, func(p pt) (float64, float64) { return pen + p.x*sc, baseline - p.y*sc })...)
		}
		pen += fc.glyphAdvance(g)
	}
	return segs
}

// flatten turns one quadratic TrueType contour into line segments.
func flatten(ct []pt, tr func(pt) (float64, float64)) []seg {
	n := len(ct)
	if n == 0 {
		return nil
	}
	// start at an on-curve point (or the midpoint of two off-curve ones)
	start := -1
	for i, p := range ct {
		if p.on {
			start = i
			break
		}
	}
	var first pt
	var seq []pt // the points after first, ending back at first
	if start < 0 {
		first = pt{(ct[0].x + ct[n-1].x) / 2, (ct[0].y + ct[n-1].y) / 2, true}
		seq = append(append(seq, ct...), first)
	} else {
		first = ct[start]
		seq = append(append(append(seq, ct[start+1:]...), ct[:start]...), first)
	}
	var segs []seg
	cx, cy := tr(first)
	line := func(x, y float64) {
		segs = append(segs, seg{cx, cy, x, y})
		cx, cy = x, y
	}
	quad := func(qx, qy, x, y float64) {
		steps := int(math.Ceil(math.Sqrt(math.Hypot(qx-(cx+x)/2, qy-(cy+y)/2) * 4)))
		steps = min(max(steps, 1), 16)
		x0, y0 := cx, cy
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			a, b, c := (1-t)*(1-t), 2*(1-t)*t, t*t
			line(a*x0+b*qx+c*x, a*y0+b*qy+c*y)
		}
	}
	var ctrl *pt
	for _, p := range seq {
		switch {
		case p.on && ctrl == nil:
			line(tr(p))
		case p.on:
			qx, qy := tr(*ctrl)
			x, y := tr(p)
			quad(qx, qy, x, y)
			ctrl = nil
		case ctrl == nil:
			pp := p
			ctrl = &pp
		default:
			mid := pt{(ctrl.x + p.x) / 2, (ctrl.y + p.y) / 2, true}
			qx, qy := tr(*ctrl)
			x, y := tr(mid)
			quad(qx, qy, x, y)
			pp := p
			ctrl = &pp
		}
	}
	return segs
}

// coverage rasterises segments into a w x h coverage mask (signed-area accumulation, non-zero fill).
func coverage(segs []seg, w, h int) []float64 {
	bw := w + 3
	acc := make([]float64, bw*h+bw)
	clampX := func(x float64) float64 { return min(max(x+1, 0), float64(bw)-2.001) }
	for _, s := range segs {
		drawLine(acc, bw, h, clampX(s.x0), s.y0, clampX(s.x1), s.y1)
	}
	m := make([]float64, w*h)
	for y := range h {
		a := 0.0
		for x := range bw {
			a += acc[y*bw+x]
			if xx := x - 1; xx >= 0 && xx < w {
				m[y*w+xx] = min(math.Abs(a), 1)
			}
		}
	}
	return m
}

func drawLine(a []float64, w, h int, x0, y0, x1, y1 float64) {
	if math.Abs(y0-y1) < 1e-9 {
		return
	}
	dir := 1.0
	if y0 > y1 {
		dir = -1
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	dxdy := (x1 - x0) / (y1 - y0)
	x := x0
	if y0 < 0 {
		x -= y0 * dxdy
	}
	for y := max(int(y0), 0); y < min(h, int(math.Ceil(y1))); y++ {
		ls := y * w
		dy := min(float64(y+1), y1) - max(float64(y), y0)
		xn := x + dxdy*dy
		d := dy * dir
		xa, xb := min(x, xn), max(x, xn)
		xaf := math.Floor(xa)
		xai := int(xaf)
		xbc := math.Ceil(xb)
		xbi := int(xbc)
		if xbi <= xai+1 {
			xmf := 0.5*(x+xn) - xaf
			a[ls+xai] += d - d*xmf
			a[ls+xai+1] += d * xmf
		} else {
			s := 1 / (xb - xa)
			x0f := xa - xaf
			a0 := 0.5 * s * (1 - x0f) * (1 - x0f)
			x1f := xb - xbc + 1
			am := 0.5 * s * x1f * x1f
			a[ls+xai] += d * a0
			if xbi == xai+2 {
				a[ls+xai+1] += d * (1 - a0 - am)
			} else {
				a1 := s * (1.5 - x0f)
				a[ls+xai+1] += d * (a1 - a0)
				for xi := xai + 2; xi < xbi-1; xi++ {
					a[ls+xi] += d * s
				}
				a2 := a1 + float64(xbi-xai-3)*s
				a[ls+xbi-1] += d * (1 - a2 - am)
			}
			a[ls+xbi] += d * am
		}
		x = xn
	}
}

// origin is where the baseline starts for text anchored at (x, y) with a PIL anchor ("la", "mm", "lm", "rm", ...).
func (fc *Face) origin(s string, x, y float64, anchor string) (float64, float64) {
	if anchor == "" {
		anchor = "la"
	}
	switch anchor[0] {
	case 'm':
		x -= fc.Length(s) / 2
	case 'r':
		x -= fc.Length(s)
	}
	switch anchor[1] {
	case 'a':
		y += fc.Ascent()
	case 'm':
		y += (fc.Ascent() + fc.Descent()) / 2
	case 'd':
		y += fc.Descent()
	}
	return x, y
}

// Text draws s anchored at (x, y) in colour col, blended by coverage (PIL's ImageDraw.text).
func (c *Canvas) Text(x, y float64, s string, col Col, fc *Face, anchor string) {
	ox, base := fc.origin(s, x, y, anchor)
	m := coverage(fc.outline(s, ox, base), c.W, c.H)
	for i, v := range m {
		if v > 0 {
			c.blend(i%c.W, i/c.W, col, v)
		}
	}
}

// BBox is the ink box of s drawn at the origin with the default anchor (PIL's getbbox), in whole pixels.
func (fc *Face) BBox(s string) (x0, y0, x1, y1 float64) {
	ox, base := fc.origin(s, 0, 0, "la")
	x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, sg := range fc.outline(s, ox, base) {
		x0, x1 = min(x0, sg.x0, sg.x1), max(x1, sg.x0, sg.x1)
		y0, y1 = min(y0, sg.y0, sg.y1), max(y1, sg.y0, sg.y1)
	}
	if math.IsInf(x0, 0) {
		return 0, 0, 0, 0
	}
	return math.Floor(x0), math.Floor(y0), math.Ceil(x1), math.Ceil(y1)
}
