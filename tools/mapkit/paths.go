// Package mapkit is what the map programs (maps/<name>/) are built with: brush and entity text for Valve 220
// .map files, the func_detail split and chunking SDHLT needs, texture lights, WAD writing, zone assembly for
// zone-based maps, and the routes and views each map hands to the Go checks and renderer.
package mapkit

import (
	"errors"
	"os"
	"path/filepath"
)

var (
	errTextureSize = errors.New("texture size must be a multiple of 16")
	errNoRepo      = errors.New("not inside the repository (no go.mod above the working directory)")
	errUsage       = errors.New("unknown command")
)

// Repo is the repository root: the nearest directory at or above the working directory holding go.mod.
func Repo() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		up := filepath.Dir(d)
		if up == d {
			return "", errNoRepo
		}
		d = up
	}
}

// BuildDir is build/<map>/ under the repository root (created if missing): every generated file goes here.
func BuildDir(mapName string) (string, error) {
	r, err := Repo()
	if err != nil {
		return "", err
	}
	d := filepath.Join(r, "build", mapName)
	return d, os.MkdirAll(d, 0o750)
}

// FontPath is a font shipped in tools/mapkit/fonts (the only fonts textures may use; licences are there too).
func FontPath(file string) (string, error) {
	r, err := Repo()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "tools", "mapkit", "fonts", file), nil
}
