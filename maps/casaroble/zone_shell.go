package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

// Zone "shell": the sky box around the whole lot and the afternoon sun. No Bounds: it wraps every other zone.
func init() {
	mapkit.AddZone(mapkit.Zone{
		Textures: []mapkit.Texture{{Name: "sky", Draw: textures.Base("sky")}}, // lowercase, or the game crashes
		Build:    buildShell,
	})
}

func buildShell(ctx *mapkit.Ctx) {
	const x0, y0, z0, x1, y1, z1 = -480.0, -400.0, -16.0, 480.0, 1856.0, 624.0 // the playable box every zone sits in
	const t = 16.0
	for _, b := range [][6]float64{{x0 - t, y0 - t, z1, x1 + t, y1 + t, z1 + t}, {x0 - t, y0 - t, z0 - t, x1 + t, y1 + t, z0},
		{x0 - t, y0 - t, z0, x0, y1 + t, z1}, {x1, y0 - t, z0, x1 + t, y1 + t, z1},
		{x0, y0 - t, z0, x1, y0, z1}, {x0, y1, z0, x1, y1 + t, z1}} {
		ctx.Brush(box(b[0], b[1], b[2], b[3], b[4], b[5], "sky", nil))
	}
	// warm late-afternoon sun from the south-west, soft blue fill from the sky
	ctx.Point("light_environment", 0, -300, 560, "pitch", "-42", "angles", "0 60 0",
		"_light", "255 228 190 105", "_diffuse_light", "150 175 220 70")
}
