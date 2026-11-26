package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

// Zone "room": a lit room with four spawns, a crate you climb by 8-unit stairs and two weapons. Copy this file to
// add a zone.
//
// Things new maps get wrong (see AGENTS.md: Lessons and References):
//   - Stairs: use stairs() from helpers.go (8-unit risers); every walkable area needs a route in Routes.
//   - Strong weapons (RPG, gauss, egon, crossbow) go in exposed or dead-end spots, well away from spawns.
//   - Outdoors: one zone must register the lowercase sky texture, e.g. {Name: "sky", Draw: textures.Base("sky")}.
//   - Small brushes (furniture, trim, steps) go in a func_detail entity so full VIS stays fast.
func init() {
	mapkit.AddZone(mapkit.Zone{
		Bounds: &[6]float64{-528, -528, -16, 528, 528, 272}, // this zone's space; other zones must not overlap it
		Textures: []mapkit.Texture{ // zone-prefixed names
			{Name: "ROOM_FLOOR", Draw: textures.Base("WOODFLOOR")}, {Name: roomWall, Draw: textures.Base("WALLPAPER")},
			{Name: "ROOM_CEIL", Draw: textures.Base("CEILING")}, {Name: roomCrate, Draw: textures.Base("CONCRETE")},
		},
		Routes: []mapkit.Route{
			{Name: "room: corner to corner around the crate", FloorZ: 0, Points: []mapkit.V2{{-300, -300}, {-300, 300}, {300, 300}}},
			{Name: "room: up the stairs onto the crate", FloorZ: 0, Points: []mapkit.V2{{0, -200}, {0, -100}, {0, 0}}},
		},
		Views: []mapkit.View{{Name: "room", Eye: mapkit.V3{-400, -400, 120}, Target: mapkit.V3{100, 100, 40}}},
		Build: buildRoom,
	})
}

const (
	roomWall  = "ROOM_WALL"
	roomCrate = "ROOM_CRATE"
)

func buildRoom(ctx *mapkit.Ctx) {
	const r, h = 512, 256
	for _, b := range []struct {
		x0, y0, z0, x1, y1, z1 float64
		tex                    string
	}{
		{-r - 16, -r - 16, -16, r + 16, r + 16, 0, "ROOM_FLOOR"}, {-r - 16, -r - 16, h, r + 16, r + 16, h + 16, "ROOM_CEIL"},
		{r, -r, 0, r + 16, r, h, roomWall}, {-r - 16, -r, 0, -r, r, h, roomWall},
		{-r, r, 0, r, r + 16, h, roomWall}, {-r, -r - 16, 0, r, -r, h, roomWall},
		{-64, -64, 0, 64, 64, 48, roomCrate},
	} {
		ctx.Brush(box(b.x0, b.y0, b.z0, b.x1, b.y1, b.z1, b.tex))
	}
	// 8-unit stairs up the crate's south side, as func_detail (small brushes don't cut VIS)
	ctx.BrushEntity("func_detail", stairs(-32, -136, 32, -64, 0, 48, roomCrate, 8, "+y"), "zhlt_detaillevel", "1")
	for _, p := range [][2]float64{{-300, -300}, {300, -300}, {-300, 300}, {300, 300}} {
		ctx.Point("info_player_deathmatch", p[0], p[1], 37, "angles", "0 0 0") // floor + 37
	}
	ctx.Point("weapon_shotgun", 0, 0, 58) // on the crate, in the open
	ctx.Point("weapon_9mmAR", 0, 200, 10)
	ctx.Point("light", -200, 0, 200, "_light", "255 240 220 300")
	ctx.Point("light", 200, 0, 200, "_light", "255 240 220 300")
}
