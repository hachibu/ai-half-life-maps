package paint

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

// Font is a parsed TrueType (glyf) font: enough of it to lay out and rasterise simple horizontal text.
type Font struct {
	data            []byte
	tables          map[string][]byte
	upem            float64
	ascent, descent float64 // font units (hhea)
	nhm, nglyphs    int
	longLoca        bool
	cmap            map[rune]int
}

var errFont = errors.New("unsupported font")

// LoadFont reads a .ttf file.
func LoadFont(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseFont(data)
}

func u16(b []byte, i int) int { return int(binary.BigEndian.Uint16(b[i:])) }
func i16(b []byte, i int) int { return int(int16(binary.BigEndian.Uint16(b[i:]))) } //nolint:gosec // signed field
func u32(b []byte, i int) int { return int(binary.BigEndian.Uint32(b[i:])) }

// ParseFont parses TrueType font data.
func ParseFont(data []byte) (*Font, error) {
	if len(data) < 12 {
		return nil, errFont
	}
	f := &Font{data: data, tables: map[string][]byte{}}
	n := u16(data, 4)
	for i := range n {
		rec := 12 + 16*i
		tag := string(data[rec : rec+4])
		off, ln := u32(data, rec+8), u32(data, rec+12)
		if off+ln > len(data) {
			return nil, fmt.Errorf("%w: table %s out of range", errFont, tag)
		}
		f.tables[tag] = data[off : off+ln]
	}
	for _, t := range []string{"head", "hhea", "hmtx", "maxp", "loca", "glyf", "cmap"} {
		if f.tables[t] == nil {
			return nil, fmt.Errorf("%w: no %s table", errFont, t)
		}
	}
	head, hhea := f.tables["head"], f.tables["hhea"]
	f.upem = float64(u16(head, 18))
	f.longLoca = i16(head, 50) == 1
	f.ascent, f.descent = float64(i16(hhea, 4)), float64(i16(hhea, 6))
	f.nhm = u16(hhea, 34)
	f.nglyphs = u16(f.tables["maxp"], 4)
	return f, f.parseCmap()
}

func (f *Font) parseCmap() error {
	cm := f.tables["cmap"]
	best := -1
	for i := range u16(cm, 2) {
		rec := 4 + 8*i
		pid, eid, off := u16(cm, rec), u16(cm, rec+2), u32(cm, rec+4)
		if u16(cm, off) == 4 && (pid == 3 && eid == 1 || pid == 0) {
			best = off
		}
	}
	if best < 0 {
		return fmt.Errorf("%w: no format 4 cmap", errFont)
	}
	t := cm[best:]
	segX2 := u16(t, 6)
	ends, starts, deltas, offs := 14, 16+segX2, 16+2*segX2, 16+3*segX2
	f.cmap = map[rune]int{}
	for s := 0; s < segX2; s += 2 {
		end, start, delta, ro := u16(t, ends+s), u16(t, starts+s), u16(t, deltas+s), u16(t, offs+s)
		for c := start; c <= end && c != 0xFFFF; c++ {
			g := 0
			if ro == 0 {
				g = (c + delta) & 0xFFFF
			} else {
				at := offs + s + ro + 2*(c-start)
				if at+2 <= len(t) {
					if g = u16(t, at); g != 0 {
						g = (g + delta) & 0xFFFF
					}
				}
			}
			f.cmap[rune(c)] = g //nolint:gosec // c is a 16-bit code point
		}
	}
	return nil
}

func (f *Font) advance(g int) float64 {
	hm := f.tables["hmtx"]
	if g >= f.nhm {
		g = f.nhm - 1
	}
	return float64(u16(hm, 4*g))
}

type pt struct {
	x, y float64
	on   bool
}

func (f *Font) glyphData(g int) []byte {
	loca, glyf := f.tables["loca"], f.tables["glyf"]
	if g < 0 || g >= f.nglyphs {
		return nil
	}
	var a, b int
	if f.longLoca {
		a, b = u32(loca, 4*g), u32(loca, 4*g+4)
	} else {
		a, b = 2*u16(loca, 2*g), 2*u16(loca, 2*g+2)
	}
	if a >= b || b > len(glyf) {
		return nil
	}
	return glyf[a:b]
}

// contours returns the glyph's outline in font units (y up).
func (f *Font) contours(g int, depth int) [][]pt {
	d := f.glyphData(g)
	if d == nil || depth > 8 {
		return nil
	}
	nc := i16(d, 0)
	if nc >= 0 {
		return simpleGlyph(d, nc)
	}
	return f.compositeGlyph(d, depth)
}

func simpleGlyph(d []byte, nc int) [][]pt {
	ends := make([]int, nc)
	for i := range nc {
		ends[i] = u16(d, 10+2*i)
	}
	if nc == 0 {
		return nil
	}
	np := ends[nc-1] + 1
	p := 10 + 2*nc
	p += 2 + u16(d, p)
	flags := make([]byte, 0, np)
	for len(flags) < np {
		fl := d[p]
		p++
		flags = append(flags, fl)
		if fl&8 != 0 {
			r := int(d[p])
			p++
			for range r {
				flags = append(flags, fl)
			}
		}
	}
	xs, ys := make([]float64, np), make([]float64, np)
	v := 0
	for i, fl := range flags[:np] {
		switch {
		case fl&2 != 0:
			dv := int(d[p])
			p++
			if fl&16 == 0 {
				dv = -dv
			}
			v += dv
		case fl&16 == 0:
			v += i16(d, p)
			p += 2
		}
		xs[i] = float64(v)
	}
	v = 0
	for i, fl := range flags[:np] {
		switch {
		case fl&4 != 0:
			dv := int(d[p])
			p++
			if fl&32 == 0 {
				dv = -dv
			}
			v += dv
		case fl&32 == 0:
			v += i16(d, p)
			p += 2
		}
		ys[i] = float64(v)
	}
	out := make([][]pt, nc)
	start := 0
	for c, e := range ends {
		for i := start; i <= e; i++ {
			out[c] = append(out[c], pt{xs[i], ys[i], flags[i]&1 != 0})
		}
		start = e + 1
	}
	return out
}

func f2dot14(b []byte, i int) float64 { return float64(int16(binary.BigEndian.Uint16(b[i:]))) / 16384 } //nolint:gosec // signed fixed point

func (f *Font) compositeGlyph(d []byte, depth int) [][]pt {
	var out [][]pt
	p := 10
	for {
		flags, gi := u16(d, p), u16(d, p+2)
		p += 4
		var dx, dy float64
		if flags&1 != 0 {
			dx, dy = float64(i16(d, p)), float64(i16(d, p+2))
			p += 4
		} else {
			dx, dy = float64(int8(d[p])), float64(int8(d[p+1])) //nolint:gosec // signed byte offsets
			p += 2
		}
		a, b, c, e := 1.0, 0.0, 0.0, 1.0
		switch {
		case flags&8 != 0:
			a = f2dot14(d, p)
			e = a
			p += 2
		case flags&0x40 != 0:
			a, e = f2dot14(d, p), f2dot14(d, p+2)
			p += 4
		case flags&0x80 != 0:
			a, b, c, e = f2dot14(d, p), f2dot14(d, p+2), f2dot14(d, p+4), f2dot14(d, p+6)
			p += 8
		}
		for _, ct := range f.contours(gi, depth+1) {
			n := make([]pt, len(ct))
			for i, q := range ct {
				n[i] = pt{a*q.x + c*q.y + dx, b*q.x + e*q.y + dy, q.on}
			}
			out = append(out, n)
		}
		if flags&0x20 == 0 {
			return out
		}
	}
}

// Face is a font at a pixel size (PIL's ImageFont.truetype(path, size): size pixels per em).
type Face struct {
	F    *Font
	Size float64
}

// Face returns the font at size pixels per em.
func (f *Font) Face(size float64) *Face { return &Face{f, size} }

func (fc *Face) scale() float64 { return fc.Size / fc.F.upem }

// Ascent is the line's ascent in pixels, rounded up (FreeType's size metrics).
func (fc *Face) Ascent() float64 { return math.Ceil(fc.F.ascent * fc.scale()) }

// Descent is the line's descent in pixels (negative), rounded down.
func (fc *Face) Descent() float64 { return math.Floor(fc.F.descent * fc.scale()) }

func (fc *Face) glyph(r rune) int { return fc.F.cmap[r] }

// glyphAdvance is a glyph's advance in whole pixels (hinted advances are rounded).
func (fc *Face) glyphAdvance(g int) float64 { return math.Round(fc.F.advance(g) * fc.scale()) }

// Length is the text's advance width in pixels (PIL's textlength / getlength).
func (fc *Face) Length(s string) float64 {
	w := 0.0
	for _, r := range s {
		w += fc.glyphAdvance(fc.glyph(r))
	}
	return w
}
