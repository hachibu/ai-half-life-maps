package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cmdPackage makes a built map a drop-in zip. build/<MAP>/valve/ is laid out like the game's valve/ folder
// (maps/, gfx/env/, sprites/, sound/). This copies the compiled .bsp into it, writes maps/<MAP>.res so servers
// send the custom files to joining players, and zips valve/ plus the map's README into build/<MAP>/<MAP>.zip.
func cmdPackage(root, name string) error {
	bd := filepath.Join(root, "build", name)
	valve := filepath.Join(bd, "valve")
	if err := os.MkdirAll(filepath.Join(valve, "maps"), 0o750); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(bd, name+".bsp"), filepath.Join(valve, "maps", name+".bsp")); err != nil {
		return err
	}
	var extras []string
	for _, sub := range []string{"gfx", "sprites", "sound"} {
		files, err := filesUnder(valve, sub)
		if err != nil {
			return err
		}
		extras = append(extras, files...)
	}
	sort.Strings(extras)
	res := filepath.Join(valve, "maps", name+".res")
	if len(extras) > 0 {
		text := fmt.Sprintf("// %s custom content (auto-downloaded by clients)\n", name) + strings.Join(extras, "\n") + "\n"
		if err := os.WriteFile(res, []byte(text), 0o644); err != nil {
			return err
		}
	} else if err := os.Remove(res); err != nil && !os.IsNotExist(err) {
		return err
	}
	zpath := filepath.Join(bd, name+".zip")
	if err := writeZip(zpath, bd, valve, filepath.Join(root, "maps", name, "README.md")); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, zpath)
	fmt.Printf("packaged %s  (%d custom files)  -> %s\n", name, len(extras), filepath.ToSlash(rel))
	return nil
}

// filesUnder lists the files below base/sub as slash paths relative to base (none if sub doesn't exist).
func filesUnder(base, sub string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(filepath.Join(base, sub), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if d.Type().IsRegular() {
			rel, _ := filepath.Rel(base, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}

// writeZip stores every file under valve (named relative to bd, e.g. valve/maps/x.bsp) and the README.
func writeZip(zpath, bd, valve, readme string) (err error) {
	files, err := filesUnder(bd, filepath.Base(valve))
	if err != nil {
		return err
	}
	sort.Strings(files)
	f, err := os.Create(zpath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	z := zip.NewWriter(f)
	for _, rel := range files {
		if err := addToZip(z, filepath.Join(bd, filepath.FromSlash(rel)), rel); err != nil {
			return err
		}
	}
	if _, serr := os.Stat(readme); serr == nil {
		if err := addToZip(z, readme, "README.md"); err != nil {
			return err
		}
	}
	return z.Close()
}

func addToZip(z *zip.Writer, path, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck // read-only
	w, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}

func copyFile(from, to string) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, b, 0o644)
}
