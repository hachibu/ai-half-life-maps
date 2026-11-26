// render: preview images and video frames from a compiled BSP (Go; see package renderer).
//
//	render views BSP ASSET_ROOT VIEWS.json OUTDIR
//	    VIEWS.json = [[name, [x,y,z] eye, [x,y,z] target], ...]; renders every view concurrently and writes
//	    OUTDIR/<name>.png plus OUTDIR/_sheet.png (half-size tiles, 3 per row)
//
// Videos use package renderer directly (see maps/prairieloop/video).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"halflifemaps/tools/renderer"
)

const (
	viewW, viewH = 854, 480
	viewT        = 1.3
	viewFOV      = 80.0
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: render views BSP ASSET_ROOT VIEWS.json OUTDIR")
		os.Exit(2)
	}
	r, err := renderer.New(os.Args[2], os.Args[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "views":
		if len(os.Args) < 6 {
			fmt.Fprintln(os.Stderr, "views needs VIEWS.json OUTDIR")
			os.Exit(2)
		}
		err = views(r, os.Args[4], os.Args[5])
	default:
		err = fmt.Errorf("%w %q", errUnknownCommand, os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	errUnknownCommand = errors.New("unknown command")
	errBadView        = errors.New("view must be [name, eye, target]")
)

type view struct {
	Name        string
	Eye, Target [3]float64
}

func (v *view) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 3 {
		return errBadView
	}
	if err := json.Unmarshal(raw[0], &v.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[1], &v.Eye); err != nil {
		return err
	}
	return json.Unmarshal(raw[2], &v.Target)
}

func toImage(px []byte, w, h int) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range w * h {
		im.Pix[4*i], im.Pix[4*i+1], im.Pix[4*i+2], im.Pix[4*i+3] = px[3*i], px[3*i+1], px[3*i+2], 255
	}
	return im
}

func savePNG(path string, im image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, im); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// views renders every view on its own goroutine (bounded by the number of cores) and writes PNGs.
func views(r *renderer.Renderer, viewsPath, outDir string) error {
	data, err := os.ReadFile(viewsPath)
	if err != nil {
		return err
	}
	var vs []view
	if err := json.Unmarshal(data, &vs); err != nil {
		return fmt.Errorf("%s: %w", viewsPath, err)
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return err
	}
	imgs := make([]*image.RGBA, len(vs))
	errs := make([]error, len(vs))
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i, v := range vs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			imgs[i] = toImage(r.Frame(v.Eye, v.Target, viewT, viewW, viewH, viewFOV, nil), viewW, viewH)
			errs[i] = savePNG(filepath.Join(outDir, v.Name+".png"), imgs[i])
		}()
	}
	wg.Wait()
	for i, v := range vs {
		if errs[i] != nil {
			return errs[i]
		}
		fmt.Println("preview", v.Name)
	}
	return savePNG(filepath.Join(outDir, "_sheet.png"), sheet(imgs))
}

// avg2 averages two bytes without overflow; avg4 averages four (a 2x2 box filter).
func avg2(a, b uint8) uint8       { return (a & b) + (a^b)>>1 }
func avg4(a, b, c, d uint8) uint8 { return avg2(avg2(a, b), avg2(c, d)) }

// sheet lays out half-size tiles (2x2 box filter), three per row.
func sheet(imgs []*image.RGBA) *image.RGBA {
	const cols, tw, th = 3, viewW / 2, viewH / 2
	rows := (len(imgs) + cols - 1) / cols
	out := image.NewRGBA(image.Rect(0, 0, cols*tw, max(rows, 1)*th))
	for i, im := range imgs {
		ox, oy := (i%cols)*tw, (i/cols)*th
		for y := range th {
			for x := range tw {
				a, b := im.RGBAAt(2*x, 2*y), im.RGBAAt(2*x+1, 2*y)
				c, d := im.RGBAAt(2*x, 2*y+1), im.RGBAAt(2*x+1, 2*y+1)
				out.SetRGBA(ox+x, oy+y, color.RGBA{avg4(a.R, b.R, c.R, d.R), avg4(a.G, b.G, c.G, d.G), avg4(a.B, b.B, c.B, d.B), 255})
			}
		}
	}
	return out
}
