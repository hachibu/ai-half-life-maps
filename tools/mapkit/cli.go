package mapkit

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Program is one map's generator: Generate writes the .map, .wad and the valve/ assets into out (build/<name>);
// Routes and Views are the walk routes the checks walk and the preview shots the renderer takes.
type Program struct {
	Name     string
	Generate func(out string) error
	Routes   func() []Route
	Views    func() []View
}

// RoutesJSON is the routes as mapcheck reads them: [[name, floor_z, [[x, y], ...]], ...].
func RoutesJSON(rs []Route) ([]byte, error) {
	out := make([][]any, len(rs))
	for i, r := range rs {
		pts := make([][2]float64, len(r.Points))
		for j, p := range r.Points {
			pts[j] = [2]float64{p[0], p[1]}
		}
		out[i] = []any{r.Name, r.FloorZ, pts}
	}
	return json.Marshal(out)
}

// ViewsJSON is the views as the renderer reads them: [[name, [x, y, z], [x, y, z]], ...].
func ViewsJSON(vs []View) ([]byte, error) {
	out := make([][]any, len(vs))
	for i, v := range vs {
		out[i] = []any{v.Name, [3]float64(v.Eye), [3]float64(v.Target)}
	}
	return json.Marshal(out)
}

// Main runs a map program:
//
//	go run ./maps/<name>           generate build/<name>/ (the .map, .wad and valve/ assets)
//	go run ./maps/<name> routes    print the walk routes as JSON (what `make test` feeds mapcheck)
//	go run ./maps/<name> views     print the preview shots as JSON (what `make views` feeds the renderer)
func Main(p Program) {
	if err := run(p, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, p.Name+":", err)
		os.Exit(1)
	}
}

func run(p Program, args []string) error {
	mode := ""
	if len(args) > 0 {
		mode = args[0]
	}
	switch mode {
	case "routes", "views":
		var b []byte
		var err error
		if mode == "routes" {
			b, err = RoutesJSON(p.Routes())
		} else {
			b, err = ViewsJSON(p.Views())
		}
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(b, '\n'))
		return err
	case "":
		out, err := BuildDir(p.Name)
		if err != nil {
			return err
		}
		t0 := time.Now()
		if err := p.Generate(out); err != nil {
			return err
		}
		fmt.Printf("generated build/%s in %.1fs\n", p.Name, time.Since(t0).Seconds())
		return nil
	default:
		return fmt.Errorf("%w %q (want no argument, routes or views)", errUsage, mode)
	}
}
