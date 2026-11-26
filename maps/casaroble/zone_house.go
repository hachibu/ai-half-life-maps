package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

// Zone "house": a one-storey 1920s Spanish bungalow. Stucco walls, flat roof behind a parapet with a clay-tile
// coping, copper downspouts. Inside: living room with a wood-burning fireplace, dining room, kitchen with a
// breakfast bar and a skylight, three bedrooms (the primary has its own bath and a sliding door to the deck), a
// hall bath and a family room that slides open onto the deck. House floor is z = 16, ceiling z = 176.
//
// Plan (x east, y north; the street is south):
//
//	y 272-568   living (x -384..-56)          | dining (x -40..256)
//	y 584-792   bedroom 2 | hall bath          | kitchen (x -40..256, y 584..888), skylight
//	y 808-888   hall (x -384..-40) ------------> kitchen
//	y 904-1200  bedroom 3 | primary + ensuite  | family room (x 48..256)
func init() {
	mapkit.AddZone(mapkit.Zone{
		Bounds: &[6]float64{-480, 256, -16, 272, 1216, 624},
		Textures: []mapkit.Texture{
			{Name: houseStucco, Draw: stuccoHouse}, {Name: housePlaster, Draw: plaster}, {Name: houseRooftile, Draw: rooftile},
			{Name: houseTerra, Draw: terracotta}, {Name: houseCopper, Draw: copper}, {Name: houseWood, Draw: textures.Base("WOODFLOOR")},
			{Name: "HOUSE_CEIL", Draw: textures.Base("CEILING")}, {Name: "HOUSE_ROOF", Draw: membrane},
			{Name: houseBrick, Draw: textures.Base("BRICK")}, {Name: "HOUSE_GLASS", Draw: textures.Base("GLASS")},
			{Name: houseCounter, Draw: textures.Base("COUNTER")}, {Name: "HOUSE_FRIDGE", Draw: textures.Base("FRIDGE")},
			{Name: "HOUSE_METAL", Draw: textures.Base("METAL")}, {Name: houseTable, Draw: tables},
			{Name: houseCouch, Draw: textures.Base("COUCH")}, {Name: houseRug, Draw: textures.Base("CARPET")},
			{Name: "HOUSE_TV", Draw: textures.Base("TVBLACK")}, {Name: houseWhite, Draw: whitepaint},
			{Name: "HOUSE_BED", Draw: bedspread(col{60, 92, 120}, col{110, 148, 175})},
			{Name: houseBed2, Draw: bedspread(col{150, 110, 60}, col{205, 165, 105})},
			{Name: "HOUSE_SIDEYARD", Draw: gravel}, {Name: "HOUSE_FENCE", Draw: textures.Base("WOODFENCE")},
		},
		Routes: []mapkit.Route{
			{Name: "house: porch through the front door, living and dining to the kitchen", FloorZ: floorZ,
				Points: []mapkit.V2{{-144, 236}, {-144, 300}, {-100, 420}, {0, 500}, {0, 640}, {0, 760}}},
			{Name: "house: kitchen down the hall to bedroom 2 and the hall bath", FloorZ: floorZ,
				Points: []mapkit.V2{{0, 848}, {-288, 848}, {-288, 760}, {-288, 848}, {-144, 848}, {-144, 720}}},
			{Name: "house: hall to bedroom 3", FloorZ: floorZ, Points: []mapkit.V2{{-200, 848}, {-288, 848}, {-288, 960}, {-300, 1040}}},
			{Name: "house: hall into the primary bedroom, its bath and out the slider", FloorZ: floorZ,
				Points: []mapkit.V2{{-48, 848}, {-48, 940}, {-84, 964}, {-168, 964}, {-84, 964}, {-84, 1100}, {-160, 1180}, {-160, 1240}}},
			{Name: "house: kitchen into the family room and out the slider", FloorZ: floorZ,
				Points: []mapkit.V2{{100, 780}, {176, 960}, {176, 1100}, {128, 1180}, {128, 1240}}},
			{Name: "house: side yard along the west wall", FloorZ: 0, Points: []mapkit.V2{{-424, 280}, {-424, 1200}}},
		},
		Views: []mapkit.View{
			{Name: "living_fireplace", Eye: mapkit.V3{-90, 530, 88}, Target: mapkit.V3{-384, 410, 56}},
			{Name: "kitchen_skylight", Eye: mapkit.V3{-20, 600, 80}, Target: mapkit.V3{150, 760, 150}},
			{Name: "family_room_to_deck", Eye: mapkit.V3{220, 920, 84}, Target: mapkit.V3{120, 1220, 60}},
			{Name: "primary_bedroom", Eye: mapkit.V3{-24, 930, 90}, Target: mapkit.V3{-170, 1190, 60}},
			{Name: "hall", Eye: mapkit.V3{-30, 848, 70}, Target: mapkit.V3{-384, 848, 60}},
		},
		Build: buildHouse,
	})
}

const (
	houseStucco   = "HOUSE_STUCCO"
	housePlaster  = "HOUSE_PLASTER"
	houseRooftile = "HOUSE_ROOFTILE"
	houseTerra    = "HOUSE_TERRA"
	houseCopper   = "HOUSE_COPPER"
	houseWood     = "HOUSE_WOOD"
	houseBrick    = "HOUSE_BRICK"
	houseCounter  = "HOUSE_COUNTER"
	houseTable    = "HOUSE_TABLE"
	houseCouch    = "HOUSE_COUCH"
	houseRug      = "HOUSE_RUG"
	houseWhite    = "HOUSE_WHITE"
	houseBed2     = "HOUSE_BED2"

	hx0, hy0, hx1, hy1 = -400.0, 256.0, 272.0, 1216.0 // outside of the exterior walls
	roofTop            = 192.0                        // roof slab top
	sill, head         = 64.0, 144.0                  // window sill and head
)

// stuccoSides is stucco on every vertical side.
var stuccoSides = faces{"s": houseStucco, "n": houseStucco, "e": houseStucco, "w": houseStucco}

func buildHouse(ctx *mapkit.Ctx) {
	var d []string // furniture and trim -> one func_detail
	houseShell(ctx, &d)
	houseRoof(ctx, &d)
	houseRooms(ctx, &d)
	detail(ctx, d)
	for _, l := range [][2]float64{{-216, 420}, {108, 400}, {108, 736}, {-304, 688}, {-132, 700}, {-200, 848},
		{-304, 1050}, {-30, 1120}, {-168, 964}, {152, 1050}} {
		light(ctx, l[0], l[1], 150, "255 228 190 120")
	}
	for _, s := range [][3]float64{{-300, 320, 0}, {200, 320, 180}, {0, 700, 90}, {-250, 760, 0}, {-280, 960, 0},
		{-170, 1120, 0}, {200, 950, 90}} {
		spawn(ctx, s[0], s[1], floorZ+37, int(s[2]))
	}
	const zi = floorZ + 10
	ctx.Point("weapon_shotgun", 110, 520, zi) // strong weapons sit away from spawns
	ctx.Point("weapon_9mmAR", 80, 720, zi)
	ctx.Point("ammo_9mmAR", 150, 1150, zi)
	ctx.Point("weapon_357", -40, 940, zi)
	ctx.Point("item_healthkit", -130, 720, zi)
	ctx.Point("item_healthkit", -190, 930, zi)
	ctx.Point("item_battery", -250, 620, zi)
	ctx.Point("ammo_buckshot", -330, 1000, zi)
}

// houseShell is the ground, floors, exterior walls with windows and sliders, and the interior walls.
func houseShell(ctx *mapkit.Ctx, d *[]string) {
	s, p := houseStucco, housePlaster
	// ground under the house zone (side yard strip and under the slab), and the west fence
	ctx.Brush(box(-464, 256, 0, -448, 1216, 96, "HOUSE_FENCE", nil))
	ctx.Brush(scaledBox(-480, 256, -16, -400, 1216, 0, "HOUSE_SIDEYARD", 2, nil))
	ctx.Brush(scaledBox(-400, 256, -16, 272, 1216, 0, "HOUSE_SIDEYARD", 4, nil))
	// floors (z 0..16): hardwood except tile in the kitchen and baths
	for _, f := range []struct {
		x0, y0, x1, y1 float64
		t              string
	}{{-400, 256, 272, 584, houseWood}, {-400, 584, -208, 808, houseWood}, {-208, 584, -40, 808, houseTerra},
		{-40, 584, 272, 904, houseTerra}, {-400, 808, -40, 904, houseWood}, {-400, 904, -208, 1216, houseWood},
		{-208, 904, -112, 1040, houseTerra}, {-208, 1040, -112, 1216, houseWood}, {-112, 904, 272, 1216, houseWood}} {
		ctx.Brush(box(f.x0, f.y0, 0, f.x1, f.y1, floorZ, f.t, stuccoSides))
	}
	// exterior walls (stucco outside, plaster inside), with breakable windows and the two sliders
	const g = "HOUSE_GLASS"
	wall(ctx, hx0, hy0, hx1, hy0+16, floorZ, roofTop, s, []gap{{-352, -224, sill, head}, {-176, -112, floorZ, doorZ}, {16, 208, sill, head}},
		g, faces{"n": p}) // front
	wall(ctx, hx0, hy1-16, hx1, hy1, floorZ, roofTop, s, []gap{{-352, -272, sill, head}, {-192, -64, floorZ, doorZ}, {96, 224, floorZ, doorZ}},
		g, faces{"s": p}) // back: sliders
	wall(ctx, hx0, hy0+16, hx0+16, hy1-16, floorZ, roofTop, s, []gap{{512, 560, sill, head}, {640, 736, sill, head}, {960, 1088, sill, head}},
		g, faces{"e": p}) // west
	wall(ctx, hx1-16, hy0+16, hx1, hy1-16, floorZ, roofTop, s, []gap{{320, 512, sill, head}, {640, 736, 72, 144}, {960, 1104, sill, head}},
		g, faces{"w": p}) // east
	ctx.Brush(box(hx0, hy0, 0, hx1, hy0+16, floorZ, s, nil)) // (slab edge is stucco)
	// sliders: the east half of each stays shut (fixed glass), the west half is open
	for _, x := range [][2]float64{{-128, -64}, {160, 224}} {
		ctx.BrushEntity("func_wall", []string{box(x[0], hy1-10, floorZ, x[1], hy1-6, doorZ, g, nil)}, "rendermode", "2", "renderamt", "100")
	}
	for _, x := range [][2]float64{{-192, -64}, {96, 224}} { // door frames
		*d = append(*d, box(x[0], hy1-18, doorZ, x[1], hy1-14, doorZ+8, houseWhite, nil))
	}
	// interior walls (plaster), doors are open 64-wide gaps
	dz := func(a, b float64) gap { return gap{a, b, floorZ, doorZ} }
	for _, w := range []struct {
		x0, y0, x1, y1 float64
		gaps           []gap
	}{
		{-384, 568, -40, 584, nil},                                   // living | bedroom 2, bath
		{-56, 272, -40, 568, []gap{{336, 528, floorZ, 152}}},         // living | dining: wide opening
		{-224, 584, -208, 792, nil},                                  // bedroom 2 | bath
		{-56, 584, -40, 792, nil},                                    // bath | kitchen
		{-384, 792, -40, 808, []gap{dz(-320, -256), dz(-176, -112)}}, // hall wall: bedroom 2, bath
		{-384, 888, 32, 904, []gap{dz(-320, -256), dz(-80, -16)}},    // hall wall: bedroom 3, primary
		{32, 888, 256, 904, []gap{{64, 224, floorZ, 152}}},           // kitchen | family: wide opening
		{-224, 904, -208, 1200, nil},                                 // bedroom 3 | primary
		{32, 904, 48, 1200, nil},                                     // primary | family
		{-128, 904, -112, 1040, []gap{dz(936, 992)}},                 // ensuite east wall + door
		{-208, 1024, -128, 1040, nil},                                // ensuite north wall
	} {
		wall(ctx, w.x0, w.y0, w.x1, w.y1, floorZ, ceilZ, p, w.gaps, "", nil)
	}
	*d = append(*d, box(-40, 568, 144, 256, 584, ceilZ, p, nil)) // soffit over the breakfast bar
}

// houseRoof is the roof slab with the kitchen skylight, the parapet and its tile coping, eyebrows, the entry
// gable, downspouts and the chimney.
func houseRoof(ctx *mapkit.Ctx, d *[]string) {
	s, p := houseStucco, housePlaster
	sk := [4]float64{96, 672, 192, 768}
	for _, r := range [][4]float64{{hx0, hy0, hx1, sk[1]}, {hx0, sk[3], hx1, hy1}, {hx0, sk[1], sk[0], sk[3]}, {sk[2], sk[1], hx1, sk[3]}} {
		ctx.Brush(box(r[0], r[1], ceilZ, r[2], r[3], roofTop, "HOUSE_CEIL", faces{"t": "HOUSE_ROOF", "s": s, "n": s, "e": s, "w": s}))
	}
	for _, r := range [][4]float64{{sk[0] - 8, sk[1] - 8, sk[2] + 8, sk[1]}, {sk[0] - 8, sk[3], sk[2] + 8, sk[3] + 8},
		{sk[0] - 8, sk[1], sk[0], sk[3]}, {sk[2], sk[1], sk[2] + 8, sk[3]}} {
		ctx.Brush(box(r[0], r[1], ceilZ, r[2], r[3], 232, p, faces{"t": s})) // light well
	}
	ctx.Brush(box(sk[0]-8, sk[1]-8, 232, sk[2]+8, sk[3]+8, 240, "sky", nil))
	for _, b := range [][6]float64{{hx0, hy0, roofTop, hx1, hy0 + 16, 224}, {hx0, hy1 - 16, roofTop, hx1, hy1, 224},
		{hx0, hy0 + 16, roofTop, hx0 + 16, hy1 - 16, 224}, {hx1 - 16, hy0 + 16, roofTop, hx1, hy1 - 16, 224}} {
		ctx.Brush(box(b[0], b[1], b[2], b[3], b[4], b[5], s, nil))
	}
	for _, b := range [][6]float64{{hx0 - 6, hy0 - 6, 224, hx1 + 6, hy0 + 18, 234}, {hx0 - 6, hy1 - 18, 224, hx1 + 6, hy1 + 6, 234},
		{hx0 - 6, hy0 + 18, 224, hx0 + 18, hy1 - 18, 234}, {hx1 - 18, hy0 + 18, 224, hx1 + 6, hy1 - 18, 234}} {
		*d = append(*d, box(b[0], b[1], b[2], b[3], b[4], b[5], houseRooftile, nil))
	}
	// tile eyebrows over the big front windows, a stepped gable over the entry, copper downspouts
	for _, x := range [][2]float64{{-368, -208}, {0, 224}} {
		*d = append(*d, box(x[0], hy0-20, 150, x[1], hy0, 160, houseRooftile, nil))
	}
	*d = append(*d, box(-208, hy0-4, roofTop, -80, hy0+16, 248, s, nil), box(-216, hy0-10, 248, -72, hy0+18, 256, houseRooftile, nil))
	for _, c := range [][2]float64{{hx0, hy0 - 8}, {hx1 - 8, hy0 - 8}, {hx0, hy1}, {hx1 - 8, hy1}} { // on the front/back faces
		*d = append(*d, box(c[0], c[1], 0, c[0]+8, c[1]+8, 228, houseCopper, nil))
	}
	// chimney
	ctx.Brush(box(hx0, 384, roofTop, hx0+48, 448, 296, s, nil))
	*d = append(*d, box(hx0-4, 380, 296, hx0+52, 452, 304, houseCopper, nil))
}

// houseRooms furnishes the rooms.
func houseRooms(ctx *mapkit.Ctx, d *[]string) {
	p := housePlaster
	add := func(bs ...string) { *d = append(*d, bs...) }
	// living room: fireplace (open firebox), couch, coffee table, rug
	ctx.Brush(box(-384, 336, floorZ, -352, 384, ceilZ, p, nil))
	ctx.Brush(box(-384, 448, floorZ, -352, 496, ceilZ, p, nil))
	ctx.Brush(box(-384, 384, 96, -352, 448, ceilZ, p, nil))
	add(box(-352, 352, floorZ, -320, 480, floorZ+4, houseBrick, nil), box(-388, 384, floorZ, -384, 448, 96, houseBrick, nil))
	ctx.Point("env_sprite", -372, 416, floorZ+2, "model", "sprites/casaroble_fire.spr", "rendermode", "5",
		"renderamt", "255", "framerate", "10", "scale", "0.45", "spawnflags", "1")
	light(ctx, -360, 416, 44, "255 140 50 140")
	add(box(-280, 496, floorZ, -136, 544, 44, houseCouch, nil), box(-280, 544, floorZ, -136, 560, 80, houseCouch, nil),
		box(-248, 400, floorZ, -168, 448, 36, houseTable, nil), box(-304, 360, floorZ, -112, 480, floorZ+1, houseRug, nil))
	// dining: table and a sideboard
	add(box(48, 352, floorZ, 176, 464, 48, houseTable, nil), box(-36, 288, floorZ, 0, 400, 56, houseTable, nil))
	// kitchen: counters on the east wall with the range, fridge, breakfast bar facing the dining room
	add(box(208, 600, floorZ, 256, 680, 56, houseCounter, nil), box(208, 680, floorZ, 256, 728, 58, "HOUSE_METAL", nil),
		box(208, 728, floorZ, 256, 832, 56, houseCounter, nil), box(200, 832, floorZ, 256, 888, 144, "HOUSE_FRIDGE", nil),
		box(64, 568, floorZ, 256, 600, 64, houseCounter, nil))
	// bedroom 2, bedroom 3, primary (bed head against the east wall)
	add(box(-376, 592, floorZ, -280, 720, 44, houseBed2, nil), box(-376, 1072, floorZ, -280, 1192, 44, houseBed2, nil),
		box(-56, 976, floorZ, 24, 1088, 44, "HOUSE_BED", nil), box(8, 976, 44, 24, 1088, 80, houseTable, nil))
	// hall bath: tub and vanity; ensuite: vanity; hall: stacked washer/dryer
	add(box(-200, 592, floorZ, -72, 640, 48, houseWhite, nil), box(-200, 690, floorZ, -176, 760, 56, houseWhite, nil),
		box(-200, 1000, floorZ, -144, 1024, 56, houseWhite, nil), box(-384, 816, floorZ, -344, 880, 144, houseWhite, nil))
	// family room: sofa facing a TV on the east wall, rug
	add(box(80, 960, floorZ, 128, 1104, 44, houseCouch, nil), box(64, 960, floorZ, 80, 1104, 80, houseCouch, nil),
		box(248, 992, 64, 256, 1072, 112, "HOUSE_TV", nil), box(144, 960, floorZ, 224, 1104, floorZ+1, houseRug, nil))
}
