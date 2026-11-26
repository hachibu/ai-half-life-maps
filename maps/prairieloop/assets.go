package main

import (
	"math"
	"path/filepath"

	"halflifemaps/tools/mapkit"
	tx "halflifemaps/tools/mapkit/textures"
)

// ------------------------------------------------------------------ overcast snowstorm skybox

func stormSky(d mapkit.V3) [3]float64 {
	n := math.Sqrt(float64(float64(d[0]*d[0])+float64(d[1]*d[1])) + float64(d[2]*d[2]))
	d = d.Div(n)
	e := math.Asin(min(max(d[2], -1), 1))
	k := math.Pow(min(max(e/(math.Pi/2), 0), 1), 0.7)
	c := [3]float64{62*(1-k) + 24*k, 70*(1-k) + 29*k, 86*(1-k) + 40*k}
	cl := math.Sin(d.Dot(mapkit.V3{7, 3, 5})+1)*math.Sin(d.Dot(mapkit.V3{-4, 9, 2})+2) +
		0.5*math.Sin(d.Dot(mapkit.V3{13, -11, 6}))*math.Sin(d.Dot(mapkit.V3{5, 17, -9}))
	glow := math.Exp(-math.Max(e, 0) / 0.07) // town glow under the clouds
	for i, v := range []float64{10, 11, 14} {
		c[i] += v * cl
	}
	for i, v := range []float64{40, 30, 22} {
		c[i] += v * glow
	}
	if e < 0.012+0.006*math.Sin(5*math.Atan2(d[1], d[0])) {
		c = [3]float64{34, 37, 46}
	}
	return c
}

func writeSkybox(prefix string) error {
	const n = 256
	for _, f := range mapkit.SkyFaces {
		px := make([]float64, 0, n*n*3)
		for j := range n {
			for i := range n {
				s := (float64(i)+0.5)/n*2 - 1
				t := 1 - (float64(j)+0.5)/n*2
				c := stormSky(mapkit.SkyDir(f, s, t))
				px = append(px, c[0], c[1], c[2])
			}
		}
		if err := mapkit.WriteTGA(prefix+f+".tga", n, n, mapkit.ToBytes(px)); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------ snow and steam sprites

var snowPal = func() (p [256][3]uint8) {
	for i := range 256 {
		p[i] = [3]uint8{uint8(float64(i) * 0.92), uint8(float64(i) * 0.95), uint8(i)}
	}
	return p
}()

// edgeFade fades a sprite to black at its borders, so its square never shows.
func edgeFade(s int) []float64 {
	fx := make([]float64, s)
	for i := range s {
		u := -1 + 2*float64(i)/float64(s-1)
		fx[i] = math.Pow(min(max((1-math.Abs(u))/0.35, 0), 1), 1.5)
	}
	out := make([]float64, s*s)
	for y := range s {
		for x := range s {
			out[y*s+x] = fx[y] * fx[x]
		}
	}
	return out
}

// burstFrames is a gust of snow: one heavy whiteout burst per loop.
func burstFrames() [][]byte {
	const n, s = 64, 128
	r := seeded(11)
	hz := tx.Noise(s, s, 10)
	pts := make([][2]float64, 900)
	for i := range pts {
		pts[i] = [2]float64{r.Float64() * s, r.Float64() * s}
	}
	fade := edgeFade(s)
	frames := make([][]byte, n)
	for f := range n {
		t := float64(f) / n
		d := (t - 0.45) / 0.12
		env := math.Exp(-d * d)
		a := make([]float64, s*s)
		if env > 0.02 {
			for y := range s {
				yy := float64(y) / (s - 1)
				band := min(max(1.2-math.Abs(yy-0.5)*1.6, 0), 1)
				for x := range s {
					a[y*s+x] += env * 150 * hz.At(x, y) * band // whiteout haze
				}
			}
			for _, p := range pts[:int(900*env)] {
				xx := math.Mod(p[0]+float64(f*9), s)
				yv := math.Mod(p[1]+float64(f*4), s)
				for k := range 6 {
					i := mapkit.PyMod(int(yv), s)*s + mapkit.PyMod(int(xx-float64(k)), s)
					a[i] = max(a[i], 200*env)
				}
			}
		}
		for i := range a {
			a[i] *= fade[i]
		}
		frames[f] = mapkit.ToBytes(a)
	}
	return frames
}

// steamFrames is puffs of steam rising from the vent.
func steamFrames() [][]byte {
	const n, s = 32, 64
	r := seeded(5)
	type puff struct{ x0, ph, rad float64 }
	puffs := make([]puff, 14)
	for i := range puffs {
		puffs[i] = puff{r.Float64() * s, r.Float64(), 5 + r.Float64()*5}
	}
	fade := edgeFade(s)
	frames := make([][]byte, n)
	for f := range n {
		a := make([]float64, s*s)
		for _, p := range puffs {
			t := math.Mod(float64(f)/n+p.ph, 1)
			cy := s - t*s
			cx := p.x0 + 6*math.Sin(t*6+p.ph*10)
			rr := p.rad * (0.6 + t)
			for y := range s {
				for x := range s {
					dx, dy := float64(x)-cx, float64(y)-cy
					a[y*s+x] += 110 * (1 - t) * math.Exp(-((dx*dx + dy*dy) / (rr * rr)))
				}
			}
		}
		for i := range a {
			a[i] = min(a[i]*fade[i], 255)
		}
		frames[f] = mapkit.ToBytes(a)
	}
	return frames
}

// ------------------------------------------------------------------ looping sounds (with a cue point so GoldSrc loops them)

type biquad struct{ b0, b1, b2, a1, a2 float64 }

// butter2 is one second-order Butterworth section (RBJ cookbook), low- or high-pass, quality q.
func butter2(fc, sr, q float64, high bool) biquad {
	w := 2 * math.Pi * fc / sr
	alpha := math.Sin(w) / (2 * q)
	cw := math.Cos(w)
	a0 := 1 + alpha
	if high {
		return biquad{(1 + cw) / 2 / a0, -(1 + cw) / a0, (1 + cw) / 2 / a0, -2 * cw / a0, (1 - alpha) / a0}
	}
	return biquad{(1 - cw) / 2 / a0, (1 - cw) / a0, (1 - cw) / 2 / a0, -2 * cw / a0, (1 - alpha) / a0}
}

func (b biquad) run(x []float64) []float64 {
	y := make([]float64, len(x))
	var x1, x2, y1, y2 float64
	for i, v := range x {
		o := b.b0*v + b.b1*x1 + b.b2*x2 - b.a1*y1 - b.a2*y2
		x2, x1, y2, y1 = x1, v, y1, o
		y[i] = o
	}
	return y
}

func reverse(x []float64) []float64 {
	y := make([]float64, len(x))
	for i, v := range x {
		y[len(x)-1-i] = v
	}
	return y
}

// bandpass is a zero-phase 4th-order Butterworth band from lo to hi Hz (filtered forwards and backwards).
func bandpass(x []float64, sr, lo, hi float64) []float64 {
	secs := []biquad{butter2(lo, sr, 0.5412, true), butter2(lo, sr, 1.3066, true),
		butter2(hi, sr, 0.5412, false), butter2(hi, sr, 1.3066, false)}
	pad := min(len(x)-1, 3000)
	ext := make([]float64, 0, len(x)+2*pad) // odd extension at both ends, as scipy's filtfilt
	for i := pad; i > 0; i-- {
		ext = append(ext, 2*x[0]-x[i])
	}
	ext = append(ext, x...)
	for i := 1; i <= pad; i++ {
		ext = append(ext, 2*x[len(x)-1]-x[len(x)-1-i])
	}
	for _, s := range secs {
		ext = s.run(ext)
	}
	ext = reverse(ext)
	for _, s := range secs {
		ext = s.run(ext)
	}
	return reverse(ext)[pad : pad+len(x)]
}

// loopNoise is band-limited noise n samples long whose end crossfades into its start, so it loops seamlessly.
func loopNoise(n int, lo, hi float64, seed uint64) []float64 {
	sr := 22050.0
	r := seeded(seed)
	f := int(0.25 * sr)
	x := make([]float64, n+f)
	for i := range x {
		x[i] = r.NormFloat64()
	}
	y := bandpass(x, sr, lo, hi)
	for i := range f {
		w := float64(i) / float64(f-1)
		y[i] = y[i]*w + y[n+i]*(1-w)
	}
	return normalise(y[:n], 1)
}

func normalise(x []float64, peak float64) []float64 {
	m := 0.0
	for _, v := range x {
		m = max(m, math.Abs(v))
	}
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = v / (m + 1e-9) * peak
	}
	return out
}

func makeSounds(dir string) error {
	const sr = 22050
	n := 8 * sr
	body := loopNoise(n, 60, 700, 1)
	hiss := loopNoise(n, 1200, 5000, 2)
	whistle := loopNoise(n, 650, 820, 3)
	wind := make([]float64, n)
	for i := range wind {
		t := float64(i) / sr
		env := 0.35 + 0.65*math.Pow(0.5+0.5*math.Sin(2*math.Pi*t/8-1.2), 3) + 0.25*math.Pow(0.5+0.5*math.Sin(2*math.Pi*t/8*3+0.4), 4)
		wind[i] = body[i]*env*0.55 + hiss[i]*env*env*0.18 + whistle[i]*min(max(env-0.6, 0), 1)*0.5
	}
	if err := mapkit.WriteWAV(filepath.Join(dir, "wind_loop.wav"), normalise(wind, 0.8), sr); err != nil {
		return err
	}
	n = 4 * sr
	air := loopNoise(n, 300, 2500, 4)
	fans := make([]float64, n)
	for i := range fans {
		t := float64(i) / sr
		hum := 0.0
		for _, fa := range [][2]float64{{60, 0.35}, {120, 0.3}, {180, 0.1}, {240, 0.08}} {
			hum += fa[1] * math.Sin(2*math.Pi*fa[0]*t)
		}
		blades := 0.12*math.Sin(2*math.Pi*212*t) + 0.1*math.Sin(2*math.Pi*213.25*t) // two fans beating
		fans[i] = hum + blades + air[i]*0.35
	}
	if err := mapkit.WriteWAV(filepath.Join(dir, "fans_loop.wav"), normalise(fans, 0.6), sr); err != nil {
		return err
	}
	return makeAlarm(filepath.Join(dir, "alarm.wav"))
}

// makeAlarm is a two-tone siren with a whole number of cycles per tone.
func makeAlarm(path string) error {
	const sr = 22050
	n := 2 * sr
	x := make([]float64, n)
	ph := 0.0
	for i := range x {
		t := float64(i) / sr
		f := 660.0
		if math.Mod(t, 1) < 0.5 {
			f = 880
		}
		ph += 2 * math.Pi * f / sr
		sq := 0.0
		if s := math.Sin(ph); s > 0 {
			sq = 1
		} else if s < 0 {
			sq = -1
		}
		env := min(1, math.Mod(t, 0.5)/0.01, (0.5-math.Mod(t, 0.5))/0.01)
		x[i] = (0.6*sq*0.35 + 0.4*math.Sin(ph)) * env * 0.7
	}
	return mapkit.WriteWAV(path, x, sr)
}
