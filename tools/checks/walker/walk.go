// Package walker moves a player-sized box along routes in the compiled world, exactly like the old Python walker:
// the player steps up at most 18 units, drops any distance, and doors/breakables (entities) are ignored.
package walker

import (
	"fmt"
	"math"

	"halflifemaps/tools/bsp/bspfile"
)

// Step is the engine's step height.
const Step = 18

// Walker walks routes through a loaded map.
type Walker struct{ M *bspfile.BSP }

// Free reports whether a standing player's centre fits at (x, y, z).
func (w Walker) Free(x, y, z float64) bool { return w.M.PlayerFree([3]float64{x, y, z}) }

// norm2 mirrors numpy.linalg.norm on a 2-vector: sqrt(fma(v1, v1, v0*v0)).
func norm2(v0, v1 float64) float64 { return math.Sqrt(math.FMA(v1, v1, float64(v0*v0))) }

// Walk follows pts (x, y) starting on floorZ with 4-unit steps. It returns whether the route is passable,
// a reason when it isn't, and the final floor height.
func (w Walker) Walk(floorZ float64, pts [][2]float64) (bool, string, float64) {
	z := floorZ + 36
	if !w.Free(pts[0][0], pts[0][1], z) {
		return false, fmt.Sprintf("start blocked at (%.0f, %.0f)", pts[0][0], pts[0][1]), floorZ
	}
	for s := 0; s+1 < len(pts); s++ {
		ax, ay, bx, by := pts[s][0], pts[s][1], pts[s+1][0], pts[s+1][1]
		dx, dy := bx-ax, by-ay
		n := max(1, int(norm2(dx, dy)/4))
		for i := 1; i <= n; i++ {
			px := ax + float64(dx*float64(i))/float64(n)
			py := ay + float64(dy*float64(i))/float64(n)
			up := 0
			for ; up <= Step; up++ {
				if w.Free(px, py, z+float64(up)) {
					break
				}
			}
			if up > Step {
				return false, fmt.Sprintf("blocked at (%.0f, %.0f), floor %.0f", px, py, z-36), z - 36
			}
			z += float64(up)
			for w.Free(px, py, z-1) && z > -4000 {
				z--
			}
		}
	}
	return true, "", z - 36
}
