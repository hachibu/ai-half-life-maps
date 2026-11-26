package main

import "halflifemaps/tools/mapkit"

// routes are the walks that must stay passable (make test MAP=prairieloop walks them).
func routes() []mapkit.Route {
	n := func(cx, x, y float64) mapkit.V2 { return mapkit.V2{cx + x, 1056 + y} }  // north-row house: local -> world
	s := func(cx, x, y float64) mapkit.V2 { return mapkit.V2{cx - x, -1056 - y} } // south-row house: local -> world
	type conv func(cx, x, y float64) mapkit.V2
	houseDown := func(c conv, cx float64) []mapkit.V2 {
		return []mapkit.V2{c(cx, -20, -170), c(cx, -20, 40), c(cx, 120, 60), c(cx, 222, 168)}
	}
	basementOut := func(c conv, cx float64) []mapkit.V2 {
		return []mapkit.V2{c(cx, -100, 168), c(cx, -100, -100), c(cx, 0, -150), c(cx, 0, -260)}
	}
	cat := func(parts ...[]mapkit.V2) []mapkit.V2 {
		var out []mapkit.V2
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}
	const cxN, cxS = -384.0, -384.0
	const cb = 384.0 // the blue house
	r := func(name string, z float64, pts ...[]mapkit.V2) mapkit.Route {
		return mapkit.Route{Name: name, FloorZ: z, Points: cat(pts...)}
	}
	p := func(v ...float64) []mapkit.V2 {
		out := make([]mapkit.V2, 0, len(v)/2)
		for i := 0; i+1 < len(v); i += 2 {
			out = append(out, mapkit.V2{v[i], v[i+1]})
		}
		return out
	}
	return []mapkit.Route{
		// in through one house, under the street, up through another
		r("street -> N1 house -> stairs down", 0, p(-300, 760), []mapkit.V2{n(cxN, 0, -240)}, houseDown(n, cxN)),
		r("N1 basement -> ring tunnel -> S1 basement", -176, basementOut(n, cxN), p(cxN, 576, -1024, 576, -1024, -576, cxS, -576),
			[]mapkit.V2{s(cxS, 0, -260), s(cxS, 0, -150), s(cxS, -100, -100), s(cxS, -100, 168)}),
		r("S1 basement -> stairs up -> street", -176, []mapkit.V2{s(cxS, -80, 168), s(cxS, 222, 168), s(cxS, 120, 60), s(cxS, -20, 40),
			s(cxS, -20, -170), s(cxS, 0, -240)}, p(-300, -760)),
		r("corner house N0 -> ring tunnel", -176, basementOut(n, -1152), p(-1152, 576, -1024, 576)),
		r("corner house S3 -> ring tunnel", -176, basementOut(s, 1152), p(1152, -576, 1024, -576)),
		// back over the roofs
		r("snowbank -> S0 roof", 0, p(-1800, -1176, -1700, -1176, -1420, -1176, -1300, -1100)),
		r("roof loop: S row -> garages -> east -> N row", 144, p(-1300, -1056, -1152, -1056, 1152, -1056, 1344, -1056, 1344, -900,
			1344, -760, 1344, -600, 1440, -576, 1440, -350, 1440, 0, 1440, 350,
			1440, 576, 1344, 576, 1344, 760, 1344, 900, 1344, 1056, -1152, 1056, -1300, 1056)),
		r("N0 roof -> exterior stairs -> ground", 144, p(-1350, 1180, -1440, 1210, -1440, 1100, -1440, 980, -1440, 900)),
		r("east house exterior stairs up", 0, p(1720, -288, 1690, -288, 1490, -288, 1490, -220)),
		// the blue house reveal and the hall
		r("street -> blue house -> computer room", 0, p(300, 740), []mapkit.V2{n(cb, 0, -240)}, houseDown(n, cb),
			[]mapkit.V2{n(cb, -100, 168), n(cb, -100, 0)}),
		r("computer room -> reveal stairs -> balcony", -176, []mapkit.V2{n(cb, -100, 0), n(cb, 128, -150)},
			p(512, 840, 512, 700, 512, 500, 512, 330)),
		r("balcony -> stairs -> hall floor", -320, p(512, 330, 420, 336, 250, 336, 200, 300, 0, 200)),
		r("hall -> BattleAxe stairs -> ring tunnel", -464, p(0, 20, -700, 20, -700, 345, -792, 345, -792, 60, -792, 0, -900, 0, -1024, 0)),
		r("hall -> Crescent stairs -> ring tunnel", -464, p(0, 20, 700, -20, 700, -345, 792, -345, 792, -60, 792, 0, 900, 0, 1024, 0)),
		r("ring tunnel -> east house basement", -176, p(1024, 0, 1150, 0, 1270, 0, 1340, 0)),
		// interactive spots (doors/breakables are entities: the space behind them must be open)
		r("hall -> server room", -464, p(-560, -200, -560, -360, -560, -420, -560, -500)),
		r("road -> north garage -> breaker 1", 0, p(1200, 576, 1270, 576, 1330, 576, 1540, 580)),
		r("south tunnel -> breaker 2", -176, p(-300, -576, 0, -576, 0, -610)),
		r("west street -> breaker 3", 0, p(-2500, 60, -2900, 60, -2915, 200)),
		r("back yard -> steam vent", 0, p(384, -1400, 384, -1316)),
		r("tunnel -> health charger", -176, p(-1024, -200, -1040, -396)),
		r("tunnel -> armor charger", -176, p(1024, -200, 1040, -396)),
		r("hall -> stage -> teleport pad", -464, p(250, -200, 250, -268, 255, -330)),
		r("hall -> centre stage", -464, p(0, -200, 0, -268, 0, -320)),
	}
}
