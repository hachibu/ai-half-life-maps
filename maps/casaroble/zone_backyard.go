package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

// Zone "backyard": the private deck off the family room and primary bedroom, ringed by oaks, and the lawn behind
// it. Deck top z = 16 (flush with the house floor), one 8-unit step down to the lawn at z = 0.
func init() {
	mapkit.AddZone(mapkit.Zone{
		Bounds: &[6]float64{-480, 1216, -16, 272, 1856, 624},
		Textures: []mapkit.Texture{
			{Name: "YARD_LAWN", Draw: lawn}, {Name: yardDeck, Draw: textures.Planks(col{105, 72, 45}, col{168, 124, 82}, false, 128, 16)},
			{Name: "YARD_BARK", Draw: bark}, {Name: "YARD_LEAVES", Draw: leaves}, {Name: yardFence, Draw: textures.Base("WOODFENCE")},
			{Name: "YARD_GRAVEL", Draw: gravel}, {Name: "YARD_TABLE", Draw: tables},
		},
		Routes: []mapkit.Route{
			{Name: "backyard: family slider across the deck and down to the lawn", FloorZ: 16,
				Points: []mapkit.V2{{128, 1180}, {128, 1240}, {-100, 1300}, {-100, 1460}, {-100, 1700}}},
			{Name: "backyard: primary slider to the deck", FloorZ: 16, Points: []mapkit.V2{{-160, 1240}, {-260, 1240}, {-300, 1380}}},
			{Name: "backyard: side yard to the deck and round the oaks", FloorZ: 0,
				Points: []mapkit.V2{{-424, 1230}, {-424, 1500}, {-260, 1490}, {-60, 1490}, {-60, 1770}, {160, 1770}, {240, 1500}}},
		},
		Views: []mapkit.View{
			{Name: "deck_under_the_oaks", Eye: mapkit.V3{220, 1780, 140}, Target: mapkit.V3{-120, 1300, 40}},
			{Name: "back_of_house", Eye: mapkit.V3{-40, 1800, 90}, Target: mapkit.V3{0, 1216, 110}},
		},
		Build: buildBackyard,
	})
}

const (
	yardDeck  = "YARD_DECK"
	yardFence = "YARD_FENCE"
)

func buildBackyard(ctx *mapkit.Ctx) {
	ctx.Brush(scaledBox(-480, 1216, -16, 272, 1856, 0, "YARD_LAWN", 2, nil))
	// deck, its 8-unit step along the lawn side, and a railing on the west end
	ctx.Brush(box(-400, 1216, 0, 128, 1440, 16, yardDeck, faces{"n": yardDeck}))
	d := []string{box(-400, 1440, 0, 128, 1464, 8, yardDeck, nil), box(-400, 1432, 16, -232, 1440, 52, yardDeck, nil),
		box(-224, 1264, 16, -136, 1352, 46, "YARD_TABLE", nil)} // outdoor table (cover)
	// oaks: solid trunks, leafy canopies far above head height
	for _, o := range [][2]float64{{200, 1330}, {-330, 1600}, {60, 1690}, {-190, 1530}} {
		d = append(d, box(o[0]-20, o[1]-20, 0, o[0]+20, o[1]+20, 256, "YARD_BARK", nil))
		canopy(ctx, o[0], o[1], "YARD_LEAVES", [4]float64{-480, 1216, 272, 1856})
	}
	// fences: west side and back
	ctx.Brush(box(-464, 1216, 0, -448, 1840, 96, yardFence, nil))
	ctx.Brush(box(-464, 1824, 0, 272, 1840, 96, yardFence, nil))
	detail(ctx, d)
	light(ctx, -100, 1330, 140, "255 220 170 120")
	spawn(ctx, -280, 1700, 37, 90)
	spawn(ctx, 140, 1770, 37, 180)
	spawn(ctx, -300, 1300, 53, 270)
	ctx.Point("weapon_crossbow", 40, 1400, 26) // out on the open deck
	ctx.Point("ammo_crossbow", -400, 1780, 10)
	ctx.Point("weapon_satchel", 200, 1560, 10)
}
