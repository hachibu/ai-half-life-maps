package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

// Zone "street": the quiet residential street, sidewalk and the front yard: low-water gravel garden with agave
// planters, a concrete walk up to a tiled entry porch, and the front of the driveway. Ground is z = 0; the road
// sits 6 units lower behind the curb.
func init() {
	mapkit.AddZone(mapkit.Zone{
		Bounds: &[6]float64{-480, -400, -16, 480, 256, 624},
		Textures: []mapkit.Texture{
			{Name: "STREET_ASPHALT", Draw: asphalt}, {Name: streetConcrete, Draw: textures.Base("CONCRETE")},
			{Name: streetGravel, Draw: gravel}, {Name: streetLawn, Draw: lawn},
			{Name: streetStucco, Draw: stucco(col{200, 182, 150}, col{232, 216, 188})}, {Name: "STREET_AGAVE", Draw: agave},
			{Name: "STREET_FENCE", Draw: textures.Base("WOODFENCE")}, {Name: "STREET_TILE", Draw: rooftile},
			{Name: streetTerra, Draw: terracotta}, {Name: streetCarbody, Draw: textures.Base("CARSILVER")},
			{Name: "STREET_CARGLASS", Draw: textures.Base("CARGLASS")}, {Name: "STREET_TIRE", Draw: textures.Base("TIRE")},
			{Name: streetMetal, Draw: textures.Base("METAL")}, {Name: "STREET_HEDGE", Draw: hedge},
		},
		Routes: []mapkit.Route{
			{Name: "street: along the sidewalk", FloorZ: 0, Points: []mapkit.V2{{-420, -50}, {420, -50}}},
			{Name: "street: across the road and back over the curb", FloorZ: -6, Points: []mapkit.V2{{-420, -160}, {420, -160}, {420, -50}}},
			{Name: "street: sidewalk up the front walk and onto the porch", FloorZ: 0, Points: []mapkit.V2{{-144, -50}, {-144, 150}, {-144, 236}}},
			{Name: "street: front yard to the side yard gate", FloorZ: 0, Points: []mapkit.V2{{-300, 160}, {-424, 200}, {-424, 250}}},
			{Name: "street: sidewalk up the driveway", FloorZ: 0, Points: []mapkit.V2{{360, -50}, {360, 250}}},
		},
		Views: []mapkit.View{
			{Name: "front_from_street", Eye: mapkit.V3{60, -330, 120}, Target: mapkit.V3{-80, 400, 110}},
			{Name: "front_porch", Eye: mapkit.V3{-60, 120, 70}, Target: mapkit.V3{-150, 256, 90}},
		},
		Build: buildStreet,
	})
}

const (
	streetConcrete = "STREET_CONCRETE"
	streetGravel   = "STREET_GRAVEL"
	streetLawn     = "STREET_LAWN"
	streetStucco   = "STREET_STUCCO"
	streetTerra    = "STREET_TERRA"
	streetCarbody  = "STREET_CARBODY"
	streetMetal    = "STREET_METAL"
)

func buildStreet(ctx *mapkit.Ctx) {
	var d []string
	add := func(b string) { d = append(d, b) }
	// road (z -6), sidewalk and curb, then the lot
	ctx.Brush(scaledBox(-480, -400, -16, 480, -112, -6, "STREET_ASPHALT", 2, nil))
	ctx.Brush(scaledBox(-480, -112, -16, 480, 0, 0, streetConcrete, 2, nil))
	for _, s := range []struct {
		x0, x1 float64
		t      string
	}{{-480, -448, streetLawn}, {-448, -176, streetGravel}, {-176, -112, streetConcrete},
		{-112, 272, streetGravel}, {272, 448, streetConcrete}, {448, 480, streetLawn}} {
		ctx.Brush(scaledBox(s.x0, 0, -16, s.x1, 256, 0, s.t, 2, nil))
	}
	// porch: two 8-unit steps up to the house floor (16), tiled, under a clay-tile roof on stucco posts
	add(box(-176, 176, 0, -112, 208, 8, streetTerra, nil))
	ctx.Brush(box(-208, 208, 0, -80, 256, 16, streetTerra, faces{"s": streetStucco, "e": streetStucco, "w": streetStucco}))
	for _, x0 := range []float64{-224, -80} {
		add(box(x0, 192, 0, x0+16, 208, 160, streetStucco, nil))
	}
	add(box(-232, 184, 160, -56, 256, 176, "STREET_TILE", faces{"b": streetStucco}))
	// low stucco planters with agave
	for _, p := range [][4]float64{{-400, 64, -240, 112}, {16, 64, 208, 112}} {
		add(box(p[0], p[1], 0, p[2], p[3], 24, streetStucco, faces{"t": streetGravel}))
		for i, x := 0, p[0]+16; x < p[2]-16; i, x = i+1, x+48 {
			add(box(x, p[1]+12, 24, x+24, p[3]-12, 40+8*float64(i%2), "STREET_AGAVE", nil))
		}
	}
	// side fences and a hedge between the lot and the neighbours
	for _, x0 := range []float64{-464, 448} {
		ctx.Brush(box(x0, 0, 0, x0+16, 256, 64, "STREET_FENCE", nil))
		if x0 < 0 {
			add(box(x0-16, 0, 0, x0, 256, 112, "STREET_HEDGE", nil))
		} else {
			add(box(x0+16, 0, 0, x0+32, 256, 112, "STREET_HEDGE", nil))
		}
	}
	// a parked car (cover on the street) and a street lamp
	const cx0, cy0 = -300.0, -300.0
	for _, w := range [][2]float64{{24, 0}, {176, 0}, {24, 88}, {176, 88}} {
		add(box(cx0+w[0], cy0+w[1]-8, -6, cx0+w[0]+40, cy0+w[1]+8, 20, "STREET_TIRE", nil))
	}
	add(box(cx0, cy0+4, 8, cx0+240, cy0+84, 44, streetCarbody, nil))
	add(box(cx0+56, cy0+12, 44, cx0+176, cy0+76, 72, "STREET_CARGLASS", faces{"t": streetCarbody}))
	add(box(392, -100, 0, 404, -88, 200, streetMetal, nil))
	add(box(360, -108, 200, 404, -80, 208, streetMetal, nil))
	detail(ctx, d)
	light(ctx, 376, -94, 190, "255 214 150 160")
	light(ctx, -144, 220, 140, "255 214 160 90")
	for _, s := range [][3]float64{{100, -50, 90}, {-320, 180, 0}, {300, -250, 180}} {
		z := 31.0
		if s[1] > -112 {
			z = 37
		}
		spawn(ctx, s[0], s[1], z, int(s[2]))
	}
	ctx.Point("weapon_handgrenade", 120, 180, 10)
	ctx.Point("ammo_9mmclip", -180, -200, 4)
}
