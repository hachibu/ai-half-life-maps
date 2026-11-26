package mapkit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"strings"

	"halflifemaps/tools/mapkit/paint"
)

// Texture is one named texture and the function that draws it (sizes must be multiples of 16).
type Texture struct {
	Name string
	Draw func() *paint.Canvas
}

// MipSampling picks how the smaller mip levels are made from the full-size palette indices.
type MipSampling int

const (
	// MipCentre samples the centre of each k x k block (PIL's NEAREST resize).
	MipCentre MipSampling = iota
	// MipCorner takes each block's top-left pixel (numpy's idx[::k, ::k]).
	MipCorner
)

type colorBox struct {
	cols  [][3]uint8
	count []int
	total int
}

// quantize maps an image to at most 255 colours by median cut (the 256th palette entry is the transparency blue).
// It returns one palette index per pixel and the palette.
func quantize(px [][3]uint8) ([]uint8, [][3]uint8) {
	hist := map[[3]uint8]int{}
	var order [][3]uint8
	for _, p := range px {
		if _, ok := hist[p]; !ok {
			order = append(order, p)
		}
		hist[p]++
	}
	boxes := []*colorBox{{}}
	for _, c := range order {
		boxes[0].cols = append(boxes[0].cols, c)
		boxes[0].count = append(boxes[0].count, hist[c])
		boxes[0].total += hist[c]
	}
	for len(boxes) < 255 {
		best := -1
		for i, b := range boxes {
			if len(b.cols) > 1 && (best < 0 || b.total > boxes[best].total) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		a, b := split(boxes[best])
		boxes[best] = a
		boxes = append(boxes, b)
	}
	index := map[[3]uint8]uint8{}
	pal := make([][3]uint8, len(boxes))
	for bi, b := range boxes {
		var sum [3]int
		for i, c := range b.cols {
			index[c] = uint8(bi)
			for k := range 3 {
				sum[k] += int(c[k]) * b.count[i]
			}
		}
		for k := range 3 {
			pal[bi][k] = uint8((sum[k] + b.total/2) / b.total) //nolint:gosec // an average of bytes
		}
	}
	idx := make([]uint8, len(px))
	for i, p := range px {
		idx[i] = index[p]
	}
	return idx, pal
}

// split cuts a box at the pixel-weighted median of its widest channel.
func split(b *colorBox) (lo, hi *colorBox) {
	ch, wid := 0, -1
	for k := range 3 {
		mn, mx := 255, 0
		for _, c := range b.cols {
			mn, mx = min(mn, int(c[k])), max(mx, int(c[k]))
		}
		if mx-mn > wid {
			ch, wid = k, mx-mn
		}
	}
	ids := make([]int, len(b.cols))
	for i := range ids {
		ids[i] = i
	}
	slices.SortStableFunc(ids, func(x, y int) int { return int(b.cols[x][ch]) - int(b.cols[y][ch]) })
	lo, hi = &colorBox{}, &colorBox{}
	half, run := b.total/2, 0
	for n, i := range ids {
		dst := hi
		if (run < half || n == 0) && n < len(ids)-1 {
			dst = lo
		}
		run += b.count[i]
		dst.cols = append(dst.cols, b.cols[i])
		dst.count = append(dst.count, b.count[i])
		dst.total += b.count[i]
	}
	return lo, hi
}

// ToMip encodes a texture as a WAD3 miptex lump: four mip levels of palette indices plus a 256-colour palette.
// Textures named "{..." are see-through: pure blue (0, 0, 255) becomes palette index 255.
func ToMip(name string, c *paint.Canvas, sampling MipSampling) ([]byte, error) {
	w, h := c.W, c.H
	if w%16 != 0 || h%16 != 0 {
		return nil, fmt.Errorf("%w: %s is %dx%d", errTextureSize, name, w, h)
	}
	px := make([][3]uint8, w*h)
	cut := make([]bool, w*h)
	for i := range px {
		px[i] = [3]uint8{c.Pix[4*i], c.Pix[4*i+1], c.Pix[4*i+2]}
		if strings.HasPrefix(name, "{") && px[i] == [3]uint8{0, 0, 255} {
			cut[i], px[i] = true, [3]uint8{}
		}
	}
	idx, pal := quantize(px)
	for i, m := range cut {
		if m {
			idx[i] = 255
		}
	}
	mips := [][]uint8{idx}
	for _, k := range []int{2, 4, 8} {
		mw, mh := w/k, h/k
		m := make([]uint8, mw*mh)
		for y := range mh {
			for x := range mw {
				sx, sy := x*k, y*k
				if sampling == MipCentre {
					sx, sy = x*k+k/2, y*k+k/2
				}
				m[y*mw+x] = idx[sy*w+sx]
			}
		}
		mips = append(mips, m)
	}
	var buf bytes.Buffer
	var nm [16]byte
	copy(nm[:], name)
	buf.Write(nm[:])
	le := binary.LittleEndian
	buf.Write(le32(nil, w))
	buf.Write(le32(nil, h))
	pos := 40
	for _, m := range mips {
		buf.Write(le32(nil, pos))
		pos += len(m)
	}
	for _, m := range mips {
		buf.Write(m)
	}
	buf.Write(le.AppendUint16(nil, 256))
	for i := range 255 {
		if i < len(pal) {
			buf.Write(pal[i][:])
		} else {
			buf.Write([]byte{0, 0, 0})
		}
	}
	buf.Write([]byte{0, 0, 255, 0, 0})
	return buf.Bytes(), nil
}

// le32 appends v as a little-endian uint32 (sizes and offsets in a WAD are far below 4 GB).
func le32(b []byte, v int) []byte {
	return binary.LittleEndian.AppendUint32(b, uint32(v)) //nolint:gosec // see above
}

// WriteWAD draws every texture and writes them to a WAD3 file; it returns each texture's pixel size.
func WriteWAD(path string, texs []Texture, sampling MipSampling) (map[string][2]int, error) {
	sizes := map[string][2]int{}
	var body bytes.Buffer
	var dir bytes.Buffer
	pos := 12
	for _, t := range texs {
		c := t.Draw()
		sizes[t.Name] = [2]int{c.W, c.H}
		data, err := ToMip(t.Name, c, sampling)
		if err != nil {
			return nil, err
		}
		var nm [16]byte
		copy(nm[:], t.Name)
		dir.Write(le32(nil, pos))
		dir.Write(le32(nil, len(data)))
		dir.Write(le32(nil, len(data)))
		dir.Write([]byte{0x43, 0, 0, 0})
		dir.Write(nm[:])
		body.Write(data)
		pos += len(data)
	}
	out := []byte("WAD3")
	out = le32(out, len(texs))
	out = le32(out, pos)
	out = append(append(out, body.Bytes()...), dir.Bytes()...)
	return sizes, os.WriteFile(path, out, 0o644)
}
