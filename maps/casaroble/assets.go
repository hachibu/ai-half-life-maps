package main

// casaroble's own game assets: a warm late-afternoon skybox and the fireplace's fire sprite.

import (
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"

	"halflifemaps/tools/mapkit"
)

// sun is roughly where light_environment's sun sits (south-west, high).
var sun = mapkit.V3{-0.45, -0.55, 0.7}.Unit()

func afternoon(d mapkit.V3) [3]float64 {
	d = d.Unit()
	up := min(max(d[2], -1), 1)
	zenith, horizon, ground := [3]float64{70, 120, 196}, [3]float64{236, 214, 182}, [3]float64{120, 112, 98}
	t := math.Pow(min(max(up, 0), 1), 0.55)
	var c [3]float64
	for i := range 3 {
		c[i] = horizon[i] + (zenith[i]-horizon[i])*t
		if up < 0 {
			c[i] = ground[i] + (horizon[i]-ground[i])*min(max(1+up*4, 0), 1)
		}
	}
	s := min(max(d.Dot(sun), 0), 1)
	glow := math.Pow(s, 48)*1.2 + math.Pow(s, 6)*0.25
	sunCol := [3]float64{255, 226, 170}
	for i := range 3 {
		c[i] += sunCol[i] * glow
	}
	// a few soft cumulus bands near the horizon
	az := math.Atan2(d[1], d[0])
	k := (up - 0.18) / 0.08
	cloud := min(max(math.Sin(az*5+1.3)*math.Sin(az*11)-0.2, 0), 1) * math.Exp(-k*k)
	white := [3]float64{250, 244, 236}
	for i := range 3 {
		c[i] += (white[i] - c[i]) * cloud * 0.6
	}
	return c
}

func writeAssets(valve, name string) error {
	for _, sub := range []string{filepath.Join("gfx", "env"), "sprites"} {
		if err := os.MkdirAll(filepath.Join(valve, sub), 0o750); err != nil {
			return err
		}
	}
	const n = 256
	for _, f := range mapkit.SkyFaces {
		px := make([]float64, 0, n*n*3)
		for j := range n {
			for i := range n {
				c := afternoon(mapkit.SkyDir(f, (float64(i)+0.5)/n*2-1, 1-(float64(j)+0.5)/n*2))
				px = append(px, c[0], c[1], c[2])
			}
		}
		if err := mapkit.WriteTGA(filepath.Join(valve, "gfx", "env", name+f+".tga"), n, n, mapkit.ToBytes(px)); err != nil {
			return err
		}
	}
	frames := mapkit.FireFrames(rand.New(rand.NewPCG(7, 0)), 64, 128, 12) //nolint:gosec // a flame, not security
	b := make([][]byte, len(frames))
	for i, f := range frames {
		b[i] = mapkit.ToBytes(f)
	}
	return mapkit.WriteSPR(filepath.Join(valve, "sprites", name+"_fire.spr"), 64, 128, b, mapkit.FirePalette(), 128)
}
