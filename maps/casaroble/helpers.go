package main

// Shared helpers for casaroble's zones (every zone_*.go file is a zone; other files are helpers). Keep them small
// and stable.

import (
	"fmt"
	"strings"

	"halflifemaps/tools/mapkit"
)

const (
	floorZ = 16.0  // house, deck: floor top. Ground (yards, driveway, garage) is z = 0.
	ceilZ  = 176.0 // house interior ceiling
	doorZ  = 128.0 // top of door openings in the house (floor 16 + 112)
)

// faces overrides a box's texture per side: "t"(op), "b"(ottom), "n"(orth, +y), "s"(outh, -y), "e"(ast, +x),
// "w"(est, -x).
type faces map[string]string

// box is an axis-aligned brush in Valve 220 format.
func box(x0, y0, z0, x1, y1, z1 float64, tex string, fs faces) string {
	return scaledBox(x0, y0, z0, x1, y1, z1, tex, 1, fs)
}

// scaledBox is box with texture scale sc (> 1 means fewer light samples, for big plain surfaces).
func scaledBox(x0, y0, z0, x1, y1, z1 float64, tex string, sc float64, fs faces) string {
	p := func(x, y, z float64) string { return mapkit.G(x) + " " + mapkit.G(y) + " " + mapkit.G(z) }
	face := func(side, p0, p1, p2, u, v string) string {
		t := tex
		if o, ok := fs[side]; ok {
			t = o
		}
		return fmt.Sprintf("( %s ) ( %s ) ( %s ) %s [ %s 0 ] [ %s 0 ] 0 %s %s", p0, p1, p2, t, u, v, mapkit.G(sc), mapkit.G(sc))
	}
	return "{\n" + strings.Join([]string{
		face("t", p(x0, y0, z1), p(x0, y1, z1), p(x1, y0, z1), "1 0 0", "0 -1 0"),
		face("b", p(x0, y0, z0), p(x1, y0, z0), p(x0, y1, z0), "1 0 0", "0 -1 0"),
		face("n", p(x0, y1, z0), p(x1, y1, z0), p(x0, y1, z1), "1 0 0", "0 0 -1"),
		face("s", p(x0, y0, z0), p(x0, y0, z1), p(x1, y0, z0), "1 0 0", "0 0 -1"),
		face("e", p(x1, y0, z0), p(x1, y0, z1), p(x1, y1, z0), "0 1 0", "0 0 -1"),
		face("w", p(x0, y0, z0), p(x0, y1, z0), p(x0, y0, z1), "0 1 0", "0 0 -1"),
	}, "\n") + "\n}\n"
}

// gap is an opening (a..b along the wall, gz0..gz1 high).
type gap struct{ a, b, gz0, gz1 float64 }

// wall is a wall from z0 to z1 with openings. It runs along its longer side; each gap is measured along that
// side. A gap whose bottom is above z0 is a window: with glass set, it gets a breakable pane.
func wall(ctx *mapkit.Ctx, x0, y0, x1, y1, z0, z1 float64, tex string, gaps []gap, glass string, fs faces) {
	alongX := (x1 - x0) >= (y1 - y0)
	lo, hi := y0, y1
	if alongX {
		lo, hi = x0, x1
	}
	piece := func(a, b, za, zb float64) {
		if b <= a || zb <= za {
			return
		}
		if alongX {
			ctx.Brush(box(a, y0, za, b, y1, zb, tex, fs))
		} else {
			ctx.Brush(box(x0, a, za, x1, b, zb, tex, fs))
		}
	}
	sorted := append([]gap(nil), gaps...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && less(sorted[j], sorted[j-1]); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	cur := lo
	for _, g := range sorted {
		piece(cur, g.a, z0, z1)
		piece(g.a, g.b, z0, g.gz0)
		piece(g.a, g.b, g.gz1, z1)
		if glass != "" && g.gz0 > z0 {
			pane := box(g.a, (y0+y1)/2-2, g.gz0, g.b, (y0+y1)/2+2, g.gz1, glass, nil)
			if !alongX {
				pane = box((x0+x1)/2-2, g.a, g.gz0, (x0+x1)/2+2, g.b, g.gz1, glass, nil)
			}
			ctx.BrushEntity("func_breakable", []string{pane}, "material", "0", "health", "15", "rendermode", "2", "renderamt", "110")
		}
		cur = g.b
	}
	piece(cur, hi, z0, z1)
}

func less(p, q gap) bool {
	for _, d := range [][2]float64{{p.a, q.a}, {p.b, q.b}, {p.gz0, q.gz0}, {p.gz1, q.gz1}} {
		if d[0] != d[1] {
			return d[0] < d[1]
		}
	}
	return false
}

// canopy is an oak canopy: overlapping leafy blocks high above the ground, as one func_illusionary (players never
// touch it). clamp (x0, y0, x1, y1) keeps it inside the zone.
func canopy(ctx *mapkit.Ctx, cx, cy float64, tex string, clamp [4]float64) {
	var parts []string
	for _, c := range [][5]float64{{0, 0, 176, 224, 304}, {-56, 40, 120, 304, 352}, {64, -48, 112, 296, 344},
		{-96, -88, 88, 208, 256}, {104, 96, 80, 216, 264}} {
		x0, y0 := max(cx+c[0]-c[2], clamp[0]), max(cy+c[1]-c[2], clamp[1])
		x1, y1 := min(cx+c[0]+c[2], clamp[2]), min(cy+c[1]+c[2], clamp[3])
		if x1 > x0 && y1 > y0 {
			parts = append(parts, box(x0, y0, c[3], x1, y1, c[4], tex, nil))
		}
	}
	ctx.BrushEntity("func_illusionary", parts)
}

// detail adds brushes as one func_detail (small brushes don't cut VIS).
func detail(ctx *mapkit.Ctx, brushes []string) {
	ctx.BrushEntity("func_detail", brushes, "zhlt_detaillevel", "1")
}

// spawn is a deathmatch spawn at (x, y) standing on floor z, facing yaw degrees.
func spawn(ctx *mapkit.Ctx, x, y, z float64, yaw int) {
	ctx.Point("info_player_deathmatch", x, y, z, "angles", fmt.Sprintf("0 %d 0", yaw))
}

func light(ctx *mapkit.Ctx, x, y, z float64, v string) { ctx.Point("light", x, y, z, "_light", v) }
