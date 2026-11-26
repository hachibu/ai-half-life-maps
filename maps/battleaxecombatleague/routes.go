package main

import (
	"fmt"

	"halflifemaps/tools/mapkit"
)

// routes are the walks that must stay passable (make test MAP=battleaxecombatleague walks them).
func routes() []mapkit.Route {
	p := func(v ...float64) []mapkit.V2 {
		out := make([]mapkit.V2, 0, len(v)/2)
		for i := 0; i+1 < len(v); i += 2 {
			out = append(out, mapkit.V2{v[i], v[i+1]})
		}
		return out
	}
	rs := []mapkit.Route{
		{Name: "red walkout: locker room -> cage", FloorZ: 24, Points: p(0, -1870, 0, -1716, 0, -1300, 0, -400, 0, -300, 0, 0)},
		{Name: "blue walkout: locker room -> cage", FloorZ: 0, Points: p(0, 1700, 0, 1476, 0, 900, 0, 400, 0, 300, 0, 0)},
		{Name: "floor -> lower aisle -> concourse", FloorZ: 0, Points: []mapkit.V2{onoct(820, 0, 0.5), onoct(1900, 0, 0.5)}},
	}
	for _, seg := range []int{1, 3, 5, 7} {
		rs = append(rs, mapkit.Route{Name: fmt.Sprintf("concourse stairs -> balcony (corner %d)", seg), FloorZ: 368,
			Points: []mapkit.V2{onoct(1900, seg, 0.05), onoct(2008, seg, 0.06), onoct(2008, seg, 0.34), onoct(2040, seg, 0.34)}})
	}
	for _, seg := range []int{0, 1, 2, 3, 4, 5, 7} {
		rs = append(rs, mapkit.Route{Name: fmt.Sprintf("balcony -> upper deck top row (side %d)", seg), FloorZ: 608,
			Points: []mapkit.V2{onoct(2030, seg, 0.5), onoct(2620, seg, 0.5)}})
	}
	return rs
}

// views are the preview shots (name, eye, target); make views MAP=battleaxecombatleague renders them.
func views() []mapkit.View {
	v := func(name string, ex, ey, ez, tx, ty, tz float64) mapkit.View {
		return mapkit.View{Name: name, Eye: mapkit.V3{ex, ey, ez}, Target: mapkit.V3{tx, ty, tz}}
	}
	return []mapkit.View{v("cage", 420, -330, 120, 0, 0, 60), v("stadium", 0, -820, 140, 0, 900, 420),
		v("upperdeck", 1500, 1500, 820, -200, -200, 100), v("walkout", 0, -1150, 80, 0, -1716, 200),
		v("redroom", 0, -1800, 100, 0, -2000, 90), v("bluetunnel", 0, 800, 60, 0, 1700, 60),
		v("blueroom", 0, 1520, 70, 0, 1812, 90), v("stairs", 1300, 1700, 420, 1500, 1300, 500),
		v("jumbotron", 500, -500, 200, 0, 0, 760),
		v("aisleSteps", 720, -60, 44, 900, 0, 30), v("balconyStairs", 1890, 830, 440, 1632, 1208, 560)} // 8-unit stairs
}
