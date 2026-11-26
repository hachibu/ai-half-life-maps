package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

// Zone "garage": the long driveway down the east side of the lot and the detached one-car garage at the back,
// half of it converted to a home office. Ground z = 0; garage ceiling z = 144.
func init() {
	mapkit.AddZone(mapkit.Zone{
		Bounds: &[6]float64{272, 256, -16, 480, 1856, 624},
		Textures: []mapkit.Texture{
			{Name: "GARAGE_CONCRETE", Draw: textures.Base("CONCRETE")}, {Name: garageStucco, Draw: stuccoHouse},
			{Name: garagePlaster, Draw: plaster}, {Name: "GARAGE_ROOF", Draw: membrane}, {Name: "GARAGE_TILE", Draw: rooftile},
			{Name: garageFence, Draw: textures.Base("WOODFENCE")}, {Name: "GARAGE_LAWN", Draw: lawn},
			{Name: "GARAGE_DESK", Draw: tables}, {Name: "GARAGE_CRT", Draw: textures.Base("CRT")},
			{Name: "GARAGE_SHELF", Draw: textures.Base("METAL")}, {Name: garageBox, Draw: cardboard},
			{Name: "GARAGE_DOOR", Draw: whitepaint},
		},
		Routes: []mapkit.Route{
			{Name: "garage: up the driveway and into the garage", FloorZ: 0, Points: []mapkit.V2{{360, 280}, {360, 1440}, {360, 1560}, {360, 1660}}},
			{Name: "garage: into the office and out the side door to the backyard", FloorZ: 0,
				Points: []mapkit.V2{{360, 1620}, {368, 1716}, {360, 1630}, {300, 1632}, {230, 1632}}},
		},
		Views: []mapkit.View{
			{Name: "driveway_to_garage", Eye: mapkit.V3{360, 300, 90}, Target: mapkit.V3{360, 1600, 60}},
			{Name: "garage_office", Eye: mapkit.V3{330, 1500, 90}, Target: mapkit.V3{400, 1770, 40}},
		},
		Build: buildGarage,
	})
}

const (
	garageStucco  = "GARAGE_STUCCO"
	garagePlaster = "GARAGE_PLASTER"
	garageFence   = "GARAGE_FENCE"
	garageBox     = "GARAGE_BOX"
)

func buildGarage(ctx *mapkit.Ctx) {
	const gx0, gy0, gx1, gy1, gh = 272.0, 1472.0, 448.0, 1792.0, 144.0
	ctx.Brush(scaledBox(272, 256, -16, 448, 1856, 0, "GARAGE_CONCRETE", 2, nil))
	ctx.Brush(scaledBox(448, 256, -16, 480, 1856, 0, "GARAGE_LAWN", 2, nil))
	ctx.Brush(box(448, 256, 0, 464, 1840, 96, garageFence, nil))
	ctx.Brush(box(272, 1824, 0, 448, 1840, 96, garageFence, nil))
	s, p := garageStucco, garagePlaster
	wall(ctx, gx0, gy0, gx1, gy0+16, 0, gh, s, []gap{{304, 416, 0, 112}}, "", faces{"n": p}) // roll-up door opening
	wall(ctx, gx0, gy1-16, gx1, gy1, 0, gh, s, nil, "", faces{"s": p})
	wall(ctx, gx0, gy0+16, gx0+16, gy1-16, 0, gh, s, []gap{{1600, 1664, 0, 112}}, "", faces{"e": p}) // side door
	wall(ctx, gx1-16, gy0+16, gx1, gy1-16, 0, gh, s, nil, "", faces{"w": p})
	ctx.Brush(box(gx0, gy0, gh, gx1, gy1, gh+16, "GARAGE_ROOF", faces{"b": p, "s": s, "n": s, "e": s, "w": s}))
	d := []string{box(gx0-6, gy0-10, gh+8, gx1+6, gy0+6, gh+20, "GARAGE_TILE", nil),
		box(304, gy0-4, 104, 416, gy0+16, 112, "GARAGE_DOOR", nil), // the raised door's edge
		// office half: a partition with a doorway, desk with an old monitor
		box(288, 1680, 0, 336, 1696, 96, p, nil), box(400, 1680, 0, 432, 1696, 96, p, nil),
		box(352, 1736, 0, 432, 1776, 32, "GARAGE_DESK", nil), box(384, 1744, 32, 416, 1768, 56, "GARAGE_CRT", nil),
		// storage half: shelving and boxes
		box(408, 1504, 0, 432, 1656, 112, "GARAGE_SHELF", nil), box(296, 1500, 0, 344, 1540, 32, garageBox, nil),
		box(304, 1504, 32, 336, 1532, 56, garageBox, nil)}
	detail(ctx, d)
	light(ctx, 360, 1580, 128, "230 235 255 110")
	light(ctx, 360, 1730, 128, "255 236 200 90")
	spawn(ctx, 360, 900, 37, 90)
	ctx.Point("weapon_rpg", 310, 1730, 10) // dead-end office: risky to fetch
	ctx.Point("ammo_rpgclip", 330, 1580, 10)
	ctx.Point("ammo_buckshot", 360, 600, 10)
}
