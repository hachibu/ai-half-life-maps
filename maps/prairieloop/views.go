package main

import "halflifemaps/tools/mapkit"

// views are the preview shots (name, eye, target); make views MAP=prairieloop renders them.
func views() []mapkit.View {
	v := func(name string, ex, ey, ez, tx, ty, tz float64) mapkit.View {
		return mapkit.View{Name: name, Eye: mapkit.V3{ex, ey, ez}, Target: mapkit.V3{tx, ty, tz}}
	}
	return []mapkit.View{
		v("street", -1700, -40, 70, 0, 300, 80), v("bluehouse", 384, 560, 60, 384, 1000, 90),
		v("rink", -700, -520, 120, 0, 0, 20), v("roofs", -1250, 1150, 210, 400, 1000, 160),
		v("compRoom", 500, 1080, -130, 250, 900, -150), v("reveal", 512, 340, -270, -100, -250, -300),
		v("hall", -700, 300, -330, 300, -200, -380), v("handles", -480, 284, -400, -480, 144, -425),
		v("axeTunnel", -1024, 400, -120, -1024, -300, -120), v("moonTunnel", 700, 576, -120, 1100, 576, -120),
		v("serverroom", -560, -250, -400, -560, -560, -420), v("farend", -1400, 0, 80, -3000, 0, 120),
		// 8-unit stairs
		v("basementStairs", -534, 1224, -120, -184, 1224, -30), v("hallStairs", -690, 360, -390, -792, 40, -190),
		v("stageSteps", 0, -60, -415, 0, -280, -455), v("roofStairs", -1620, 820, 100, -1440, 1110, 50),
	}
}
