package main

// Preview screenshots on the `previews` branch, and the PR comment that shows them. CI-only: agents add views
// and run `make views`.

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	marker = "<!-- map-previews -->"
	// changedDiff is the mean absolute pixel difference (0-255) that counts as a visible change.
	changedDiff = 0.5
	// blankStd: a screenshot whose grey levels vary less than this is one flat colour.
	blankStd = 4.0
	sheet    = "_sheet"
)

func cmdPreviews(args []string) (int, error) {
	switch {
	case args[0] == "store" && len(args) == 4:
		return 0, store(args[1], args[2], args[3])
	case args[0] == "comment" && len(args) == 6:
		md, err := comment(args[1], args[2], args[3], args[4], args[5])
		if err != nil {
			return 1, err
		}
		fmt.Println(md)
		return 0, nil
	case args[0] == "gate" && len(args) == 4:
		return gate(args[1], args[2], args[3])
	}
	return 2, errUsage
}

// store copies ARTIFACTS/<map>/previews/*.png into WORKTREE/PREFIX/<map>/*.jpg (replacing that folder). PREFIX
// is "main" for merges and "pr-<number>" for pull requests.
func store(artifacts, worktree, prefix string) error {
	dirs, err := filepath.Glob(filepath.Join(artifacts, "*", "previews"))
	if err != nil {
		return err
	}
	for _, prev := range dirs {
		name := filepath.Base(filepath.Dir(prev))
		dest := filepath.Join(worktree, prefix, name)
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
		if err := os.MkdirAll(dest, 0o750); err != nil {
			return err
		}
		pngs, err := filepath.Glob(filepath.Join(prev, "*.png"))
		if err != nil {
			return err
		}
		for _, p := range pngs {
			if err := pngToJPEG(p, filepath.Join(dest, strings.TrimSuffix(filepath.Base(p), ".png")+".jpg")); err != nil {
				return err
			}
		}
		fmt.Fprintln(os.Stderr, "stored", name, len(pngs), "images")
	}
	return nil
}

func pngToJPEG(from, to string) (err error) {
	f, err := os.Open(from)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // read-only
	im, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("%s: %w", from, err)
	}
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()
	return jpeg.Encode(out, im, &jpeg.Options{Quality: 85})
}

func loadJPEG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	return jpeg.Decode(f)
}

func rgb(im image.Image, x, y int) (r, g, b int) {
	r32, g32, b32, _ := im.At(x, y).RGBA()
	return int(r32 >> 8), int(g32 >> 8), int(b32 >> 8)
}

// diff is the mean absolute difference over every pixel and channel (255 if the sizes differ).
func diff(a, b string) (float64, error) {
	x, err := loadJPEG(a)
	if err != nil {
		return 0, err
	}
	y, err := loadJPEG(b)
	if err != nil {
		return 0, err
	}
	bx, by := x.Bounds(), y.Bounds()
	if bx.Dx() != by.Dx() || bx.Dy() != by.Dy() {
		return 255, nil
	}
	var sum int64
	for j := range bx.Dy() {
		for i := range bx.Dx() {
			r1, g1, b1 := rgb(x, bx.Min.X+i, bx.Min.Y+j)
			r2, g2, b2 := rgb(y, by.Min.X+i, by.Min.Y+j)
			sum += int64(absInt(r1-r2) + absInt(g1-g2) + absInt(b1-b2))
		}
	}
	return float64(sum) / float64(3*bx.Dx()*bx.Dy()), nil
}

// greyStd is the standard deviation of the grey levels (ITU-R 601 luma, as image editors compute "L").
func greyStd(path string) (float64, error) {
	im, err := loadJPEG(path)
	if err != nil {
		return 0, err
	}
	bd := im.Bounds()
	var s, s2 float64
	n := float64(bd.Dx() * bd.Dy())
	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			r, g, b := rgb(im, x, y)
			l := float64((r*19595 + g*38470 + b*7471 + 0x8000) >> 16)
			s += l
			s2 += l * l
		}
	}
	m := s / n
	return math.Sqrt(math.Max(0, s2/n-m*m)), nil
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// views lists dir/*.jpg except the contact sheet, by name.
func views(dir string) []string {
	ps, _ := filepath.Glob(filepath.Join(dir, "*.jpg"))
	var out []string
	for _, p := range ps {
		if strings.TrimSuffix(filepath.Base(p), ".jpg") != sheet {
			out = append(out, p)
		}
	}
	return out
}

func stem(p string) string { return strings.TrimSuffix(filepath.Base(p), ".jpg") }

func subdirs(dir string) []string {
	es, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range es {
		if e.IsDir() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

type mapViews struct {
	changed   [][2]string // before, after
	new, same []string
	gone      []string
}

func classify(afterDir, beforeDir string) (mapViews, error) {
	var mv mapViews
	for _, a := range views(afterDir) {
		b := filepath.Join(beforeDir, filepath.Base(a))
		if !exists(b) {
			mv.new = append(mv.new, a)
			continue
		}
		d, err := diff(a, b)
		if err != nil {
			return mv, err
		}
		if d > changedDiff {
			mv.changed = append(mv.changed, [2]string{b, a})
		} else {
			mv.same = append(mv.same, a)
		}
	}
	for _, b := range views(beforeDir) {
		if !exists(filepath.Join(afterDir, filepath.Base(b))) {
			mv.gone = append(mv.gone, stem(b))
		}
	}
	return mv, nil
}

// comment writes the markdown comparing WORKTREE/main/<map> (before) with WORKTREE/PREFIX/<map> (after),
// changed and new views first, unchanged ones folded away.
func comment(worktree, prefix, repo, ref, runURL string) (string, error) {
	url := func(p string) string {
		rel, _ := filepath.Rel(worktree, p)
		return fmt.Sprintf("https://github.com/%s/blob/%s/%s?raw=true", repo, ref, filepath.ToSlash(rel))
	}
	img := func(p, alt string) string { return fmt.Sprintf(`<img src="%s" alt="%s" width="420">`, url(p), alt) }
	out := []string{marker, "## Screenshots", "",
		fmt.Sprintf("Rendered by CI from this branch ([run](%s)). *Before* is `main`, *after* is this pull request. ", runURL) +
			"To get a shot of your change, add a named view to your zone's `VIEWS` (or the map's `views.go`).", ""}
	maps := subdirs(filepath.Join(worktree, prefix))
	if len(maps) == 0 {
		out = append(out, "_No previews were produced. The build probably failed; check the run._")
	}
	for _, afterDir := range maps {
		mv, err := classify(afterDir, filepath.Join(worktree, "main", filepath.Base(afterDir)))
		if err != nil {
			return "", err
		}
		line := fmt.Sprintf("%d changed, %d new, %d unchanged", len(mv.changed), len(mv.new), len(mv.same))
		if len(mv.gone) > 0 {
			line += ", removed: " + strings.Join(mv.gone, ", ")
		}
		out = append(out, fmt.Sprintf("### `%s`", filepath.Base(afterDir)), "", line, "")
		if len(mv.changed) > 0 {
			out = append(out, "| view | before | after |", "|---|---|---|")
			for _, ba := range mv.changed {
				out = append(out, fmt.Sprintf("| `%s` | %s | %s |", stem(ba[1]), img(ba[0], "before"), img(ba[1], "after")))
			}
			out = append(out, "")
		}
		if len(mv.new) > 0 {
			out = append(out, "| new view | after |", "|---|---|")
			for _, a := range mv.new {
				out = append(out, fmt.Sprintf("| `%s` | %s |", stem(a), img(a, "after")))
			}
			out = append(out, "")
		}
		if len(mv.same) > 0 {
			out = append(out, "<details><summary>Unchanged views</summary>", "")
			for _, a := range mv.same {
				out = append(out, fmt.Sprintf("`%s`<br>%s<br>", stem(a), img(a, stem(a))))
			}
			out = append(out, "", "</details>", "")
		}
	}
	return strings.Join(out, "\n"), nil
}

// gate: every map whose own files changed has at least one changed or new view, and no screenshot is blank
// (one flat colour). Exit 1 with reasons otherwise.
func gate(worktree, prefix, changed string) (int, error) {
	var bad []string
	for _, afterDir := range subdirs(filepath.Join(worktree, prefix)) {
		ps, _ := filepath.Glob(filepath.Join(afterDir, "*.jpg"))
		for _, a := range ps {
			if stem(a) == sheet {
				continue
			}
			sd, err := greyStd(a)
			if err != nil {
				return 1, err
			}
			if sd < blankStd {
				bad = append(bad, fmt.Sprintf("%s/%s is blank (flat colour): camera inside a wall or nothing rendered", filepath.Base(afterDir), stem(a)))
			}
		}
	}
	for m := range strings.FieldsSeq(changed) {
		afterDir, beforeDir := filepath.Join(worktree, prefix, m), filepath.Join(worktree, "main", m)
		if !exists(afterDir) {
			bad = append(bad, m+": no screenshots were produced")
			continue
		}
		mv, err := classify(afterDir, beforeDir)
		if err != nil {
			return 1, err
		}
		if len(mv.changed)+len(mv.new) == 0 && exists(beforeDir) {
			bad = append(bad, m+" changed but no screenshot shows a difference: add a view aimed at the change")
		}
	}
	for _, b := range bad {
		fmt.Println("visual gate:", b)
	}
	if len(bad) > 0 {
		fmt.Println("visual gate: FAIL")
		return 1, nil
	}
	fmt.Println("visual gate: ok")
	return 0, nil
}
