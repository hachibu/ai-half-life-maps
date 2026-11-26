package main

// Shared helpers for this map's zones (every zone_*.go file is a zone; other files are helpers). Keep them small
// and stable.

import (
	"fmt"
	"math"
	"strings"

	"halflifemaps/tools/mapkit"
)

// box is an axis-aligned brush in Valve 220 format.
func box(x0, y0, z0, x1, y1, z1 float64, tex string) string {
	p := func(x, y, z float64) string { return mapkit.G(x) + " " + mapkit.G(y) + " " + mapkit.G(z) }
	face := func(p0, p1, p2, u, v string) string {
		return fmt.Sprintf("( %s ) ( %s ) ( %s ) %s [ %s 0 ] [ %s 0 ] 0 1 1", p0, p1, p2, tex, u, v)
	}
	return "{\n" + strings.Join([]string{
		face(p(x0, y0, z1), p(x0, y1, z1), p(x1, y0, z1), "1 0 0", "0 -1 0"),
		face(p(x0, y0, z0), p(x1, y0, z0), p(x0, y1, z0), "1 0 0", "0 -1 0"),
		face(p(x0, y1, z0), p(x1, y1, z0), p(x0, y1, z1), "1 0 0", "0 0 -1"),
		face(p(x0, y0, z0), p(x0, y0, z1), p(x1, y0, z0), "1 0 0", "0 0 -1"),
		face(p(x1, y0, z0), p(x1, y0, z1), p(x1, y1, z0), "0 1 0", "0 0 -1"),
		face(p(x0, y0, z0), p(x0, y1, z0), p(x0, y0, z1), "0 1 0", "0 0 -1"),
	}, "\n") + "\n}\n"
}

// stairs is a straight staircase filling (x0, y0)-(x1, y1), climbing from floor z0 to z1 in the direction axis
// ("+x", "-x", "+y" or "-y"). Risers are rise units (8 looks best in GoldSrc; never more than 16, the walker
// and players step at most 18). The last step's top is z1, so it meets the floor above flush.
func stairs(x0, y0, x1, y1, z0, z1 float64, tex string, rise float64, axis string) []string {
	n := max(1, int(math.Ceil((z1-z0)/rise)))
	lo, hi := y0, y1
	if axis[1] == 'x' {
		lo, hi = x0, x1
	}
	out := make([]string, 0, n)
	for i := range n {
		a, b := lo+(hi-lo)*float64(i)/float64(n), lo+(hi-lo)*float64(i+1)/float64(n)
		if axis[0] == '-' {
			a, b = hi-(b-lo), hi-(a-lo)
		}
		top := min(z1, z0+rise*float64(i+1))
		if axis[1] == 'x' {
			out = append(out, box(a, y0, z0, b, y1, top, tex))
		} else {
			out = append(out, box(x0, a, z0, x1, b, top, tex))
		}
	}
	return out
}
