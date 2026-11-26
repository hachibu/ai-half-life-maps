package paint

import (
	"image"
	"image/color"
	"math"
)

// Canvas is an 8-bit RGBA image. Shapes are drawn aliased (every covered pixel takes the colour), like PIL's
// ImageDraw; text is anti-aliased. Opaque canvases stand in for PIL's RGB images, transparent ones for RGBA.
type Canvas struct {
	W, H int
	Pix  []uint8 // RGBA, row-major
}

// NewCanvas is an opaque canvas of one colour.
func NewCanvas(w, h int, bg Col) *Canvas {
	c := &Canvas{w, h, make([]uint8, 4*w*h)}
	for i := range w * h {
		c.Pix[4*i], c.Pix[4*i+1], c.Pix[4*i+2], c.Pix[4*i+3] = u8(bg[0]), u8(bg[1]), u8(bg[2]), 255
	}
	return c
}

// NewTransparent is a fully transparent canvas (PIL's Image.new("RGBA", size, (0, 0, 0, 0))).
func NewTransparent(w, h int) *Canvas { return &Canvas{w, h, make([]uint8, 4*w*h)} }

func u8(v float64) uint8 { return uint8(min(max(v, 0), 255)) }

// Clone copies the canvas.
func (c *Canvas) Clone() *Canvas {
	return &Canvas{c.W, c.H, append([]uint8(nil), c.Pix...)}
}

// Opaque sets every alpha to 255 (PIL's convert("RGB"), which drops alpha).
func (c *Canvas) Opaque() *Canvas {
	o := c.Clone()
	for i := 3; i < len(o.Pix); i += 4 {
		o.Pix[i] = 255
	}
	return o
}

// RGB is the colour at (x, y).
func (c *Canvas) RGB(x, y int) [3]uint8 {
	i := 4 * (y*c.W + x)
	return [3]uint8{c.Pix[i], c.Pix[i+1], c.Pix[i+2]}
}

// Set paints one pixel (ignored outside the canvas).
func (c *Canvas) Set(x, y int, col Col) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := 4 * (y*c.W + x)
	c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = u8(col[0]), u8(col[1]), u8(col[2]), 255
}

// blend mixes col into (x, y) with coverage m in 0..1, alpha included (PIL's paste of a colour through a mask).
func (c *Canvas) blend(x, y int, col Col, m float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || m <= 0 {
		return
	}
	m = min(m, 1)
	i := 4 * (y*c.W + x)
	for k := range 3 {
		c.Pix[i+k] = uint8(math.Round(float64(c.Pix[i+k])*(1-m) + col[k]*m))
	}
	c.Pix[i+3] = uint8(math.Round(float64(c.Pix[i+3])*(1-m) + 255*m))
}

// Image converts to a standard library image (for PNG previews).
func (c *Canvas) Image() *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	copy(im.Pix, c.Pix)
	return im
}

// At is the pixel at (x, y) as a standard colour.
func (c *Canvas) At(x, y int) color.RGBA {
	i := 4 * (y*c.W + x)
	return color.RGBA{c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3]}
}

// Rect fills [x0, x1] x [y0, y1] (inclusive corners, as PIL's rectangle).
func (c *Canvas) Rect(x0, y0, x1, y1 float64, fill Col) {
	ix0, iy0, ix1, iy1 := int(x0), int(y0), int(x1), int(y1)
	for y := max(iy0, 0); y <= min(iy1, c.H-1); y++ {
		for x := max(ix0, 0); x <= min(ix1, c.W-1); x++ {
			c.Set(x, y, fill)
		}
	}
}

// RectOutline draws a border width pixels thick inside [x0, x1] x [y0, y1].
func (c *Canvas) RectOutline(x0, y0, x1, y1 float64, col Col, width int) {
	ix0, iy0, ix1, iy1 := int(x0), int(y0), int(x1), int(y1)
	for i := range width {
		c.Rect(float64(ix0+i), float64(iy0+i), float64(ix1-i), float64(iy0+i), col)
		c.Rect(float64(ix0+i), float64(iy1-i), float64(ix1-i), float64(iy1-i), col)
		c.Rect(float64(ix0+i), float64(iy0+i), float64(ix0+i), float64(iy1-i), col)
		c.Rect(float64(ix1-i), float64(iy0+i), float64(ix1-i), float64(iy1-i), col)
	}
}

// RectFillOutline fills a rectangle and draws a border inside it.
func (c *Canvas) RectFillOutline(x0, y0, x1, y1 float64, fill, outline Col, width int) {
	c.Rect(x0, y0, x1, y1, fill)
	c.RectOutline(x0, y0, x1, y1, outline, width)
}

// Line draws segments through pts (x, y pairs) width pixels wide.
func (c *Canvas) Line(pts []float64, col Col, width int) {
	for i := 0; i+3 < len(pts); i += 2 {
		c.segment(pts[i], pts[i+1], pts[i+2], pts[i+3], col, width)
	}
}

func (c *Canvas) segment(fx0, fy0, fx1, fy1 float64, col Col, width int) {
	x0, y0, x1, y1 := int(fx0), int(fy0), int(fx1), int(fy1)
	if width <= 1 {
		c.bresenham(x0, y0, x1, y1, col)
		return
	}
	dx, dy := float64(x1-x0), float64(y1-y0)
	if dx == 0 && dy == 0 {
		c.Set(x0, y0, col)
		return
	}
	big := math.Hypot(dx, dy)
	small := float64(width-1) / 2
	rmax := roundUp(small) / big
	rmin := roundDown(small) / big
	c.polyFill([]float64{
		float64(x0) - rmax*dy, float64(y0) + rmax*dx,
		float64(x0) + rmin*dy, float64(y0) - rmin*dx,
		float64(x1) + rmin*dy, float64(y1) - rmin*dx,
		float64(x1) - rmax*dy, float64(y1) + rmax*dx,
	}, col)
}

func roundUp(f float64) float64 {
	if f >= 0 {
		return math.Floor(f + 0.5)
	}
	return -math.Floor(math.Abs(f) + 0.5)
}

func roundDown(f float64) float64 {
	if f >= 0 {
		return math.Ceil(f - 0.5)
	}
	return -math.Ceil(math.Abs(f) - 0.5)
}

func (c *Canvas) bresenham(x0, y0, x1, y1 int, col Col) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	e := dx + dy
	for {
		c.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Polygon fills the polygon through pts (x, y pairs), edges included.
func (c *Canvas) Polygon(pts []float64, fill Col) {
	ip := make([]float64, len(pts))
	for i, v := range pts {
		ip[i] = float64(int(v))
	}
	c.polyFill(ip, fill)
}

// PolygonOutline draws the polygon's edges.
func (c *Canvas) PolygonOutline(pts []float64, col Col, width int) {
	n := len(pts) / 2
	for i := range n {
		j := (i + 1) % n
		c.segment(pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1], col, width)
	}
}

func (c *Canvas) polyFill(pts []float64, fill Col) {
	n := len(pts) / 2
	if n < 2 {
		return
	}
	ymin, ymax := math.Inf(1), math.Inf(-1)
	for i := range n {
		ymin, ymax = min(ymin, pts[2*i+1]), max(ymax, pts[2*i+1])
	}
	for y := int(math.Ceil(ymin)); y <= int(math.Floor(ymax)); y++ {
		fy := float64(y)
		var xs []float64
		for i := range n {
			j := (i + 1) % n
			ax, ay, bx, by := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
			if ay == by {
				continue
			}
			if (fy >= ay && fy < by) || (fy >= by && fy < ay) || (fy == ymax && (fy == ay || fy == by) && max(ay, by) == ymax) {
				xs = append(xs, ax+(fy-ay)*(bx-ax)/(by-ay))
			}
		}
		sortFloats(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			for x := int(math.Round(xs[k])); x <= int(math.Round(xs[k+1])); x++ {
				c.Set(x, y, fill)
			}
		}
	}
	for i := range n {
		j := (i + 1) % n
		c.bresenham(int(math.Round(pts[2*i])), int(math.Round(pts[2*i+1])), int(math.Round(pts[2*j])), int(math.Round(pts[2*j+1])), fill)
	}
}

func sortFloats(v []float64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

// ellipse geometry for the bounding box [x0, x1] x [y0, y1] (inclusive pixels, as PIL).
type ellipse struct{ cx, cy, rx, ry float64 }

func newEllipse(x0, y0, x1, y1 float64) ellipse {
	ix0, iy0, ix1, iy1 := float64(int(x0)), float64(int(y0)), float64(int(x1)), float64(int(y1))
	return ellipse{(ix0 + ix1 + 1) / 2, (iy0 + iy1 + 1) / 2, (ix1 - ix0 + 1) / 2, (iy1 - iy0 + 1) / 2}
}

func (e ellipse) in(x, y int, shrink float64) bool {
	rx, ry := e.rx-shrink, e.ry-shrink
	if rx <= 0 || ry <= 0 {
		return false
	}
	dx, dy := (float64(x)+0.5-e.cx)/rx, (float64(y)+0.5-e.cy)/ry
	return dx*dx+dy*dy <= 1
}

// angle of pixel (x, y) around the centre, degrees clockwise from 3 o'clock, in [0, 360).
func (e ellipse) angle(x, y int) float64 {
	a := math.Atan2((float64(y)+0.5-e.cy)/e.ry, (float64(x)+0.5-e.cx)/e.rx) * 180 / math.Pi
	if a < 0 {
		a += 360
	}
	return a
}

func (c *Canvas) eachIn(e ellipse, f func(x, y int)) {
	for y := max(int(e.cy-e.ry)-1, 0); y <= min(int(e.cy+e.ry)+1, c.H-1); y++ {
		for x := max(int(e.cx-e.rx)-1, 0); x <= min(int(e.cx+e.rx)+1, c.W-1); x++ {
			if e.in(x, y, 0) {
				f(x, y)
			}
		}
	}
}

// Ellipse fills the ellipse inside the box.
func (c *Canvas) Ellipse(x0, y0, x1, y1 float64, fill Col) {
	c.eachIn(newEllipse(x0, y0, x1, y1), func(x, y int) { c.Set(x, y, fill) })
}

// EllipseOutline draws the ellipse's outline width pixels thick.
func (c *Canvas) EllipseOutline(x0, y0, x1, y1 float64, col Col, width int) {
	e := newEllipse(x0, y0, x1, y1)
	c.eachIn(e, func(x, y int) {
		if !e.in(x, y, float64(width)) {
			c.Set(x, y, col)
		}
	})
}

func inArc(a, start, end float64) bool {
	start, end = math.Mod(start, 360), math.Mod(end, 360)
	if start < 0 {
		start += 360
	}
	if end <= 0 {
		end += 360
	}
	if start <= end {
		return a >= start && a <= end
	}
	return a >= start || a <= end
}

// Arc draws the part of the outline between start and end degrees (clockwise from 3 o'clock).
func (c *Canvas) Arc(x0, y0, x1, y1, start, end float64, col Col, width int) {
	e := newEllipse(x0, y0, x1, y1)
	c.eachIn(e, func(x, y int) {
		if !e.in(x, y, float64(width)) && inArc(e.angle(x, y), start, end) {
			c.Set(x, y, col)
		}
	})
}

// Chord fills the region between the arc from start to end degrees and the straight line joining its ends.
func (c *Canvas) Chord(x0, y0, x1, y1, start, end float64, fill Col) {
	e := newEllipse(x0, y0, x1, y1)
	pt := func(deg float64) (float64, float64) {
		r := deg * math.Pi / 180
		return e.cx + e.rx*math.Cos(r), e.cy + e.ry*math.Sin(r)
	}
	ax, ay := pt(start)
	bx, by := pt(end)
	mx, my := pt((start + end) / 2)
	side := func(px, py float64) float64 { return (bx-ax)*(py-ay) - (by-ay)*(px-ax) }
	want := side(mx, my)
	c.eachIn(e, func(x, y int) {
		if s := side(float64(x)+0.5, float64(y)+0.5); s*want >= 0 {
			c.Set(x, y, fill)
		}
	})
}

// RoundedRect fills a rectangle with corners of radius r.
func (c *Canvas) RoundedRect(x0, y0, x1, y1, r float64, fill Col) {
	ix0, iy0, ix1, iy1 := float64(int(x0)), float64(int(y0)), float64(int(x1)), float64(int(y1))
	for y := max(int(iy0), 0); y <= min(int(iy1), c.H-1); y++ {
		for x := max(int(ix0), 0); x <= min(int(ix1), c.W-1); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			cx := min(max(px, ix0+r), ix1+1-r)
			cy := min(max(py, iy0+r), iy1+1-r)
			if (px-cx)*(px-cx)+(py-cy)*(py-cy) <= r*r {
				c.Set(x, y, fill)
			}
		}
	}
}

// AlphaComposite draws src over c with its top-left at (x, y) (PIL's alpha_composite).
func (c *Canvas) AlphaComposite(src *Canvas, x, y int) {
	for sy := range src.H {
		for sx := range src.W {
			dx, dy := x+sx, y+sy
			if dx < 0 || dy < 0 || dx >= c.W || dy >= c.H {
				continue
			}
			s := 4 * (sy*src.W + sx)
			d := 4 * (dy*c.W + dx)
			sa := float64(src.Pix[s+3]) / 255
			da := float64(c.Pix[d+3]) / 255
			oa := sa + da*(1-sa)
			if oa == 0 {
				c.Pix[d], c.Pix[d+1], c.Pix[d+2], c.Pix[d+3] = 0, 0, 0, 0
				continue
			}
			for k := range 3 {
				v := (float64(src.Pix[s+k])*sa + float64(c.Pix[d+k])*da*(1-sa)) / oa
				c.Pix[d+k] = uint8(math.Round(min(v, 255)))
			}
			c.Pix[d+3] = uint8(math.Round(oa * 255))
		}
	}
}

// Rotate90 turns the canvas a quarter turn counter-clockwise (PIL's rotate(90) on a square image).
func (c *Canvas) Rotate90() *Canvas {
	o := NewTransparent(c.H, c.W)
	for y := range o.H {
		for x := range o.W {
			copy(o.Pix[4*(y*o.W+x):4*(y*o.W+x)+4], c.Pix[4*(x*c.W+(c.W-1-y)):4*(x*c.W+(c.W-1-y))+4])
		}
	}
	return o
}

// FlipV flips the canvas top to bottom.
func (c *Canvas) FlipV() *Canvas {
	o := NewTransparent(c.W, c.H)
	for y := range c.H {
		copy(o.Pix[4*y*c.W:4*(y+1)*c.W], c.Pix[4*(c.H-1-y)*c.W:4*(c.H-y)*c.W])
	}
	return o
}
