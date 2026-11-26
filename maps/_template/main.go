// Command _template is the starter map: copy this folder to start a new map (the folder name becomes the map
// name). It assembles every zone in this folder (the zone_*.go files); you normally never edit this file: add
// or change zones instead.
//
//	go run ./maps/_template    (or: make build MAP=_template)
package main

import (
	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

func main() {
	mapkit.Main(mapkit.ZoneMap(textures.Emissive, "message", "Template map"))
}
