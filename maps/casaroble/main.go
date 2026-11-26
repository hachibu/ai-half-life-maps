// Command casaroble generates Casa Roble: a 1920s one-storey Spanish bungalow on a hillside street, built from the
// zones in this folder (zone_*.go), plus its afternoon skybox and the fireplace's fire sprite. Add or change zones
// instead of editing this file.
//
//	go run ./maps/casaroble    (or: make build MAP=casaroble)
package main

import (
	"path/filepath"

	"halflifemaps/tools/mapkit"
	"halflifemaps/tools/mapkit/textures"
)

func main() {
	p := mapkit.ZoneMap(textures.Emissive, "message", "Casa Roble", "skyname", "casaroble")
	assemble := p.Generate
	p.Generate = func(out string) error {
		if err := assemble(out); err != nil {
			return err
		}
		return writeAssets(filepath.Join(out, "valve"), p.Name)
	}
	mapkit.Main(p)
}
