// Package layout is Prairie Loop's street plan: the playable extent, the houses and garages. The map and its
// walkthrough video both use it.
package layout

// The playable extent (west, east, south and north edges) and the height of the walkable roofs.
const (
	XW   = -3072.0
	XE   = 1920.0
	YS   = -1472.0
	YN   = 1472.0
	Roof = 144.0
)

// HouseXs are the x spans of the four houses in each row.
var HouseXs = [][2]float64{{-1408, -896}, {-640, -128}, {128, 640}, {896, 1408}}

// House is one house: its centre, rotation (0: front faces south, 2: north, 1: west) and siding.
type House struct {
	Name   string
	CX, CY float64
	Rot    int
	Siding string
	Blue   bool // the blue house, whose basement holds the computer room and the stairs down to the hall
}

var sidings = []string{"SIDING_CREAM", "SIDING_RED", "SIDING_BLUE", "SIDING_GRAY", "SIDING_GREEN", "SIDING_YELLOW",
	"SIDING_WHITE", "SIDING_BROWN", "SIDING_SAGE"}

// Houses are the north row, the south row and the east house.
var Houses = func() []House {
	const blue = 2
	var hs []House
	names := []string{"N0", "N1", "N2", "N3", "S0", "S1", "S2", "S3"}
	for i, x := range HouseXs {
		hs = append(hs, House{names[i], (x[0] + x[1]) / 2, 1056, 0, sidings[i], i == blue})
	}
	for i, x := range HouseXs {
		hs = append(hs, House{names[4+i], (x[0] + x[1]) / 2, -1056, 2, sidings[4+i], false})
	}
	return append(hs, House{"E", 1504, 0, 1, "SIDING_SAGE", false})
}()

// Garages are the two garages' x0, y0, x1, y1.
var Garages = [][4]float64{{1280, 448, 1600, 704}, {1280, -704, 1600, -448}}

// L2W turns house-local (x, y) into world coordinates.
func L2W(h House, x, y float64) (float64, float64) {
	switch h.Rot {
	case 0:
		return h.CX + x, h.CY + y
	case 2:
		return h.CX - x, h.CY - y
	}
	return h.CX + y, h.CY - x // rot 1: local front (-y) faces world -x
}

// Footprint is a house's x0, y0, x1, y1.
func Footprint(h House) [4]float64 {
	ax, ay := L2W(h, -256, -224)
	bx, by := L2W(h, 256, 224)
	return [4]float64{min(ax, bx), min(ay, by), max(ax, bx), max(ay, by)}
}
