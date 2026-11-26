package mapkit

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"os"
)

// SkyDir is the view direction through skybox face ("rt", "lf", "bk", "ft", "up", "dn") at image coordinates
// s, t in [-1, 1] (t = +1 is the top row), as GoldSrc / Quake 2 lay the faces out.
func SkyDir(face string, s, t float64) V3 {
	switch face {
	case "rt":
		return V3{1, -s, t}
	case "lf":
		return V3{-1, s, t}
	case "bk":
		return V3{s, 1, t}
	case "ft":
		return V3{-s, -1, t}
	case "up":
		return V3{-t, -s, 1}
	case "dn":
		return V3{t, -s, -1}
	}
	panic("bad sky face " + face)
}

// SkyFaces are the six skybox faces in the order the files are written.
var SkyFaces = []string{"rt", "bk", "lf", "ft", "up", "dn"}

// WriteTGA writes an uncompressed 24-bit TGA; rgb is w*h*3 bytes, top row first.
func WriteTGA(path string, w, h int, rgb []byte) error {
	out := make([]byte, 18, 18+len(rgb))
	out[2] = 2
	binary.LittleEndian.PutUint16(out[12:], uint16(w)) //nolint:gosec // skybox faces are 256 px
	binary.LittleEndian.PutUint16(out[14:], uint16(h)) //nolint:gosec // likewise
	out[16] = 24
	for y := h - 1; y >= 0; y-- { // bottom-up, BGR
		for x := range w {
			i := 3 * (y*w + x)
			out = append(out, rgb[i+2], rgb[i+1], rgb[i])
		}
	}
	return os.WriteFile(path, out, 0o644)
}

// WriteSPR writes an additive sprite (.spr version 2): frames are w*h palette indices; originY is the frame
// origin's height (h puts the sprite's base at its entity, h/2 centres it).
func WriteSPR(path string, w, h int, frames [][]byte, pal [256][3]uint8, originY int) error {
	le := binary.LittleEndian
	out := []byte("IDSP")
	for _, v := range []int32{2, 0, 1} { // version, type (parallel-upright... 0), texture format (additive)
		out = le.AppendUint32(out, uint32(v)) //nolint:gosec // small constants
	}
	out = le.AppendUint32(out, math.Float32bits(float32(math.Hypot(float64(w), float64(h))/2)))
	for _, v := range []int{w, h, len(frames)} {
		out = le.AppendUint32(out, uint32(v)) //nolint:gosec // sprite sizes are small
	}
	out = le.AppendUint32(out, math.Float32bits(0))
	out = le.AppendUint32(out, 1)
	out = le.AppendUint16(out, 256)
	for _, c := range pal {
		out = append(out, c[0], c[1], c[2])
	}
	for _, f := range frames {
		out = le.AppendUint32(out, 0)
		for _, v := range []int{-w / 2, originY, w, h} {
			out = le.AppendUint32(out, uint32(int32(v))) //nolint:gosec // signed origin, stored as int32
		}
		out = append(out, f...)
	}
	return os.WriteFile(path, out, 0o644)
}

// ToBytes clips float values to 0..255 and truncates them to bytes.
func ToBytes(v []float64) []byte {
	b := make([]byte, len(v))
	for i, x := range v {
		b[i] = byte(min(max(x, 0), 255))
	}
	return b
}

// FireFrames is an animated flame (n frames of w x h heat values 0..255), from a little fire simulation.
func FireFrames(r *rand.Rand, w, h, n int) [][]float64 {
	heat := make([]float64, (h+2)*w)
	xs := make([]float64, w)
	for i := range xs {
		xs[i] = -1 + 2*float64(i)/float64(w-1)
	}
	at := func(y, x int) float64 { return heat[PyMod(y, h+2)*w+PyMod(x, w)] }
	var frames [][]float64
	for step := range n*3 + 40 {
		for y := h; y < h+2; y++ {
			for x := range w {
				v := 0.0
				if math.Abs(xs[x]) < 0.75 {
					v = math.Sqrt(r.Float64()) * 255
				}
				heat[y*w+x] = v
			}
		}
		next := make([]float64, len(heat))
		for y := range h + 2 {
			for x := range w {
				avg := (at(y+1, x) + at(y+1, x-1) + at(y+1, x+1) + at(y+2, x)) / 4
				next[y*w+x] = max(avg-r.Float64()*3.2, 0)
			}
		}
		heat = next
		if step >= 40 && (step-40)%3 == 0 {
			f := make([]float64, w*h)
			for y := range h {
				yy := float64(y) / float64(h-1)
				for x := range w {
					env := min(max(1-math.Abs(xs[x])/(0.25+0.75*yy), 0), 1)
					f[y*w+x] = min(max(heat[y*w+x]*env*1.3, 0), 255)
				}
			}
			frames = append(frames, f)
		}
	}
	return frames[:n]
}

// FirePalette runs black -> red -> orange -> yellow-white.
func FirePalette() [256][3]uint8 {
	var pal [256][3]uint8
	for i := range 256 {
		t := float64(i) / 255
		r := min(255, int(t*3*255))
		g := min(255, int(max(0, t*3-1)*255))
		b := min(255, int(max(0, t*3-2)*255))
		pal[i] = [3]uint8{uint8(r), uint8(float64(g) * 0.85), uint8(float64(b) * 0.6)} //nolint:gosec // 0..255
	}
	pal[0] = [3]uint8{}
	return pal
}

// WriteWAV writes 16-bit mono PCM with a cue point, so GoldSrc loops it (samples are clipped to -1..1).
func WriteWAV(path string, x []float64, sr int) error {
	le := binary.LittleEndian
	pcm := make([]byte, 0, 2*len(x))
	for _, v := range x {
		pcm = le.AppendUint16(pcm, uint16(int16(min(max(v, -1), 1)*32000))) //nolint:gosec // in range by the clip
	}
	var fmtc []byte
	fmtc = le.AppendUint16(fmtc, 1)
	fmtc = le.AppendUint16(fmtc, 1)
	fmtc = le.AppendUint32(fmtc, uint32(sr))   //nolint:gosec // sample rate
	fmtc = le.AppendUint32(fmtc, uint32(sr*2)) //nolint:gosec // byte rate
	fmtc = le.AppendUint16(fmtc, 2)
	fmtc = le.AppendUint16(fmtc, 16)
	cue := le.AppendUint32(nil, 1)
	cue = le.AppendUint32(cue, 1)
	cue = le.AppendUint32(cue, 0)
	cue = append(cue, "data"...)
	cue = le.AppendUint32(cue, 0)
	cue = le.AppendUint32(cue, 0)
	cue = le.AppendUint32(cue, 0)
	body := []byte("WAVEfmt ")
	body = le.AppendUint32(body, 16)
	body = append(body, fmtc...)
	body = append(body, "cue "...)
	body = le.AppendUint32(body, uint32(len(cue))) //nolint:gosec // small
	body = append(body, cue...)
	body = append(body, "data"...)
	body = le.AppendUint32(body, uint32(len(pcm))) //nolint:gosec // a few MB at most
	body = append(body, pcm...)
	out := append([]byte("RIFF"), le.AppendUint32(nil, uint32(len(body)))...) //nolint:gosec // likewise
	return os.WriteFile(path, append(out, body...), 0o644)
}
