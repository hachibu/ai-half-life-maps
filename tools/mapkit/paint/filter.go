package paint

import (
	"math"
	"math/rand/v2"
)

func gaussKernel(sigma float64) []float64 {
	r := int(math.Ceil(3 * sigma))
	k := make([]float64, 2*r+1)
	sum := 0.0
	for i := range k {
		d := float64(i - r)
		k[i] = math.Exp(-d * d / (2 * sigma * sigma))
		sum += k[i]
	}
	for i := range k {
		k[i] /= sum
	}
	return k
}

// blurPlane blurs one w x h plane of values in place (separable Gaussian, edges extended).
func blurPlane(v []float64, w, h int, sigma float64) {
	if sigma <= 0 {
		return
	}
	k := gaussKernel(sigma)
	r := len(k) / 2
	tmp := make([]float64, len(v))
	for y := range h {
		for x := range w {
			s := 0.0
			for i, kv := range k {
				s += kv * v[y*w+min(max(x+i-r, 0), w-1)]
			}
			tmp[y*w+x] = s
		}
	}
	for y := range h {
		for x := range w {
			s := 0.0
			for i, kv := range k {
				s += kv * tmp[min(max(y+i-r, 0), h-1)*w+x]
			}
			v[y*w+x] = s
		}
	}
}

// Blur is PIL's GaussianBlur(radius) on an 8-bit canvas (radius is the standard deviation).
func (c *Canvas) Blur(radius float64) *Canvas {
	o := c.Clone()
	plane := make([]float64, c.W*c.H)
	for ch := range 4 {
		for i := range plane {
			plane[i] = float64(c.Pix[4*i+ch])
		}
		blurPlane(plane, c.W, c.H, radius)
		for i, v := range plane {
			o.Pix[4*i+ch] = uint8(math.Round(min(max(v, 0), 255)))
		}
	}
	return o
}

// Noise is the textures' base noise: uniform random bytes, blurred, stretched to fill 0..1, times amp.
func Noise(r *rand.Rand, w, h int, blur, amp float64) *Field {
	plane := make([]float64, w*h)
	for i := range plane {
		plane[i] = float64(uint8(r.Float64() * 255))
	}
	blurPlane(plane, w, h, blur)
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, v := range plane {
		plane[i] = math.Round(min(max(v, 0), 255)) / 255
		lo, hi = min(lo, plane[i]), max(hi, plane[i])
	}
	f := NewField(w, h)
	for i, v := range plane {
		f.V[i] = (v - lo) / (hi - lo + 1e-9) * amp
	}
	return f
}

// ResizeNearest scales to w x h picking the nearest source pixel (PIL's NEAREST: centre sampling).
func (c *Canvas) ResizeNearest(w, h int) *Canvas {
	o := NewTransparent(w, h)
	sx, sy := float64(c.W)/float64(w), float64(c.H)/float64(h)
	for y := range h {
		yy := min(int((float64(y)+0.5)*sy), c.H-1)
		for x := range w {
			xx := min(int((float64(x)+0.5)*sx), c.W-1)
			copy(o.Pix[4*(y*w+x):4*(y*w+x)+4], c.Pix[4*(yy*c.W+xx):4*(yy*c.W+xx)+4])
		}
	}
	return o
}

// ResizeField is ResizeNearest's partner for the noise fields some textures stretch (PIL's default BICUBIC
// resize of an 8-bit image).
func ResizeField(f *Field, w, h int) *Field {
	c := NewTransparent(f.W, f.H)
	for i, v := range f.V {
		b := uint8(v * 255)
		c.Pix[4*i], c.Pix[4*i+1], c.Pix[4*i+2], c.Pix[4*i+3] = b, b, b, 255
	}
	r := c.resample(w, h, bicubic, 2)
	o := NewField(w, h)
	for i := range o.V {
		o.V[i] = float64(r.Pix[4*i]) / 255
	}
	return o
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	x *= math.Pi
	return math.Sin(x) / x
}

func lanczos(x float64) float64 {
	if x > -3 && x < 3 {
		return sinc(x) * sinc(x/3)
	}
	return 0
}

func bicubic(x float64) float64 {
	const a = -0.5
	x = math.Abs(x)
	switch {
	case x < 1:
		return ((a+2)*x-(a+3))*x*x + 1
	case x < 2:
		return (((x-5)*x+8)*x - 4) * a
	}
	return 0
}

// ResizeLanczos scales to w x h with PIL's LANCZOS filter (alpha premultiplied while filtering, as PIL does).
func (c *Canvas) ResizeLanczos(w, h int) *Canvas { return c.resample(w, h, lanczos, 3) }

func (c *Canvas) resample(w, h int, filter func(float64) float64, support float64) *Canvas {
	src := premul(c)
	tmp := pass(src, c.W, c.H, w, true, filter, support)
	out := pass(tmp, w, c.H, h, false, filter, support)
	return unpremul(out, w, h)
}

func premul(c *Canvas) []float64 {
	p := make([]float64, len(c.Pix))
	for i := 0; i < len(c.Pix); i += 4 {
		a := float64(c.Pix[i+3]) / 255
		p[i], p[i+1], p[i+2], p[i+3] = float64(c.Pix[i])*a, float64(c.Pix[i+1])*a, float64(c.Pix[i+2])*a, float64(c.Pix[i+3])
	}
	return p
}

func unpremul(p []float64, w, h int) *Canvas {
	o := NewTransparent(w, h)
	for i := 0; i < len(p); i += 4 {
		a := min(max(p[i+3], 0), 255)
		o.Pix[i+3] = uint8(math.Round(a))
		for k := range 3 {
			v := 0.0
			if a > 0 {
				v = p[i+k] * 255 / a
			}
			o.Pix[i+k] = uint8(math.Round(min(max(v, 0), 255)))
		}
	}
	return o
}

// pass resamples one axis: horizontal (w -> n columns) or vertical (h -> n rows).
func pass(p []float64, w, h, n int, horiz bool, filter func(float64) float64, support float64) []float64 {
	in := h
	if horiz {
		in = w
	}
	scale := float64(in) / float64(n)
	fs := max(scale, 1)
	sup := support * fs
	ow, oh := w, n
	if horiz {
		ow, oh = n, h
	}
	out := make([]float64, 4*ow*oh)
	for o := range n {
		center := (float64(o) + 0.5) * scale
		lo := max(int(center-sup+0.5), 0)
		hi := min(int(center+sup+0.5), in)
		ws := make([]float64, hi-lo)
		total := 0.0
		for j := range ws {
			ws[j] = filter((float64(j+lo) - center + 0.5) / fs)
			total += ws[j]
		}
		for j := range ws {
			ws[j] /= total
		}
		lines := h
		if !horiz {
			lines = w
		}
		for l := range lines {
			var acc [4]float64
			for j, wt := range ws {
				var si int
				if horiz {
					si = 4 * (l*w + lo + j)
				} else {
					si = 4 * ((lo+j)*w + l)
				}
				for k := range 4 {
					acc[k] += wt * p[si+k]
				}
			}
			var di int
			if horiz {
				di = 4 * (l*ow + o)
			} else {
				di = 4 * (o*ow + l)
			}
			for k := range 4 {
				out[di+k] = min(max(acc[k], 0), 255*255)
			}
		}
	}
	return out
}

// Shear is PIL's AFFINE transform (1, k, -k*h, 0, 1, 0) with BICUBIC sampling: each output row samples the
// input shifted by k*(y - h), which slants text. Outside the source is transparent.
func (c *Canvas) Shear(k float64) *Canvas {
	src := premul(c)
	out := make([]float64, len(src))
	for y := range c.H {
		for x := range c.W {
			xin := float64(x) + 0.5 + k*(float64(y)+0.5) - k*float64(c.H)
			yin := float64(y) + 0.5
			if xin < 0 || xin >= float64(c.W) {
				continue
			}
			fx := xin - 0.5
			x0 := int(math.Floor(fx))
			t := fx - float64(x0)
			var acc [4]float64
			for i := -1; i <= 2; i++ {
				wt := bicubic(float64(i) - t)
				xx := min(max(x0+i, 0), c.W-1)
				for ch := range 4 {
					acc[ch] += wt * src[4*(int(yin)*c.W+xx)+ch]
				}
			}
			for ch := range 4 {
				out[4*(y*c.W+x)+ch] = min(max(acc[ch], 0), 255*255)
			}
		}
	}
	return unpremul(out, c.W, c.H)
}
