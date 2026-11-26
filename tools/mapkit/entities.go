package mapkit

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Val writes an entity value the way the maps always have: strings as is, integers in decimal, floats like
// Python's repr (2.5, 2.0).
func Val(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case float64:
		return Repr(x)
	default:
		panic(fmt.Sprintf("entity value of type %T", v))
	}
}

func keyValues(kv []any) string {
	if len(kv)%2 != 0 {
		panic("entity keys and values must come in pairs")
	}
	var b strings.Builder
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			panic(fmt.Sprintf("entity key %v is not a string", kv[i]))
		}
		b.WriteString(`"` + k + `" "` + Val(kv[i+1]) + "\"\n")
	}
	return b.String()
}

// Point is a point entity at (x, y, z); kv lists keys and values in pairs, in the order they're written.
func Point(cls string, x, y, z float64, kv ...any) string {
	return "{\n\"classname\" \"" + cls + "\"\n\"origin\" \"" + G(x) + " " + G(y) + " " + G(z) + "\"\n" + keyValues(kv) + "}\n"
}

// BrushEntity is a brush entity holding brushes; kv as for Point.
func BrushEntity(cls string, brushes []string, kv ...any) string {
	return "{\n\"classname\" \"" + cls + "\"\n" + keyValues(kv) + strings.Join(brushes, "") + "}\n"
}

// Worldspawn is the world entity: its keys, then every world brush.
func Worldspawn(brushes []string, kv ...any) string {
	return BrushEntity("worldspawn", brushes, kv...)
}

var planeLine = regexp.MustCompile(`^\s*\(([^)]*)\)\s*\(([^)]*)\)\s*\(([^)]*)\)`)

func parseV3(s string) V3 {
	var v V3
	for i, f := range strings.Fields(s) {
		v[i], _ = strconv.ParseFloat(f, 64)
	}
	return v
}

// BrushBBox is the bounding box of one brush, from its plane lines.
func BrushBBox(text string) (lo, hi V3) {
	var planes []Plane
	for line := range strings.SplitSeq(text, "\n") {
		m := planeLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		p0, p1, p2 := parseV3(m[1]), parseV3(m[2]), parseV3(m[3])
		n := p0.Sub(p1).Cross(p2.Sub(p1)).Unit()
		planes = append(planes, Plane{n, n.Dot(p1)})
	}
	lo = V3{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi = V3{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for i := range planes {
		for j := i + 1; j < len(planes); j++ {
			for k := j + 1; k < len(planes); k++ {
				p, ok := solve3([3]V3{planes[i].N, planes[j].N, planes[k].N}, V3{planes[i].D, planes[j].D, planes[k].D})
				if !ok || !inside(planes, p) {
					continue
				}
				for c := range 3 {
					lo[c], hi[c] = min(lo[c], p[c]), max(hi[c], p[c])
				}
			}
		}
	}
	return lo, hi
}

func inside(planes []Plane, p V3) bool {
	for _, q := range planes {
		if q.N.Dot(p) > q.D+0.01 {
			return false
		}
	}
	return true
}

// solve3 solves A x = b by LU with partial pivoting; ok is false when |det A| < 1e-6.
func solve3(a [3]V3, b V3) (V3, bool) {
	det := 1.0
	for k := range 3 {
		piv := k
		for i := k + 1; i < 3; i++ {
			if math.Abs(a[i][k]) > math.Abs(a[piv][k]) {
				piv = i
			}
		}
		if piv != k {
			a[k], a[piv] = a[piv], a[k]
			b[k], b[piv] = b[piv], b[k]
			det = -det
		}
		det *= a[k][k]
		if a[k][k] == 0 {
			return V3{}, false
		}
		for i := k + 1; i < 3; i++ {
			l := a[i][k] / a[k][k]
			for j := k; j < 3; j++ {
				a[i][j] -= l * a[k][j]
			}
			b[i] -= l * b[k]
		}
	}
	if math.Abs(det) < 1e-6 {
		return V3{}, false
	}
	var x V3
	for i := 2; i >= 0; i-- {
		s := b[i]
		for j := i + 1; j < 3; j++ {
			s -= a[i][j] * x[j]
		}
		x[i] = s / a[i][i]
	}
	return x, true
}

// SplitDetail splits world brushes into structural ones (at least two extents >= span: walls, floors, roofs,
// which seal the map and cut it into the spaces VIS reasons about) and detail (everything smaller: still solid,
// lit and shadow-casting, but ignored by VIS, which keeps full VIS fast). A detail brush can't seal the map:
// if one was needed, hlbsp reports a leak and the build fails.
func SplitDetail(brushes []string, span float64) (structural, detail []string) {
	for _, b := range brushes {
		lo, hi := BrushBBox(b)
		big := 0
		for c := range 3 {
			if hi[c]-lo[c] >= span {
				big++
			}
		}
		if big >= 2 {
			structural = append(structural, b)
		} else {
			detail = append(detail, b)
		}
	}
	return structural, detail
}

// DetailEntity is one func_detail holding brushes (empty when there are none).
func DetailEntity(brushes []string) string {
	if len(brushes) == 0 {
		return ""
	}
	return "{\n\"classname\" \"func_detail\"\n\"zhlt_detaillevel\" \"1\"\n" + strings.Join(brushes, "") + "}\n"
}

// Chunks groups brushes by the cell-sized XY square their centre falls in, so one big decorative entity becomes
// several small ones (the engine checks a brush entity's visibility by its bounding box). Groups come in cell
// order; brushes keep their order within a group.
func Chunks(brushes []string, cell float64) [][]string {
	groups := map[[2]int][]string{}
	var keys [][2]int
	for _, b := range brushes {
		lo, hi := BrushBBox(b)
		c := lo.Add(hi).Div(2)
		k := [2]int{FloorDiv(c[0], cell), FloorDiv(c[1], cell)}
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], b)
	}
	slices.SortFunc(keys, func(a, b [2]int) int {
		if a[0] != b[0] {
			return a[0] - b[0]
		}
		return a[1] - b[1]
	})
	out := make([][]string, len(keys))
	for i, k := range keys {
		out[i] = groups[k]
	}
	return out
}

// Glow is one glowing texture and how bright it glows (0..255).
type Glow struct {
	Name string
	Glow float64
}

// Texlights is the info_texlights entity: each texture in emissive lights in its own average colour (taken from
// the map's .wad), brightness glow*scale.
func Texlights(wadPath string, emissive []Glow, scale float64) (string, error) {
	data, err := os.ReadFile(wadPath)
	if err != nil {
		return "", err
	}
	want := map[string]bool{}
	for _, e := range emissive {
		want[strings.ToUpper(e.Name)] = true
	}
	le := binary.LittleEndian
	num, ofs := int(le.Uint32(data[4:])), int(le.Uint32(data[8:]))
	avg := map[string][3]float64{}
	for i := range num {
		at := ofs + 32*i
		pos := int(le.Uint32(data[at:]))
		name, _, _ := strings.Cut(string(data[at+16:at+32]), "\x00")
		name = strings.ToUpper(name)
		if !want[name] {
			continue
		}
		w, h := int(le.Uint32(data[pos+16:])), int(le.Uint32(data[pos+20:]))
		off0 := int(le.Uint32(data[pos+24:]))
		palAt := pos + 40 + w*h*85/64 + 2
		var tot [3]int
		for _, p := range data[pos+off0 : pos+off0+w*h] {
			for c := range 3 {
				tot[c] += int(data[palAt+3*int(p)+c])
			}
		}
		avg[name] = [3]float64{float64(tot[0]) / float64(w*h), float64(tot[1]) / float64(w*h), float64(tot[2]) / float64(w*h)}
	}
	var b strings.Builder
	for _, e := range emissive {
		rgb, ok := avg[strings.ToUpper(e.Name)]
		if !ok {
			continue
		}
		top := max(rgb[0], rgb[1], rgb[2])
		if top == 0 {
			top = 1
		}
		fmt.Fprintf(&b, "\"%s\" \"%d %d %d %d\"\n", e.Name, int(Round(255*rgb[0]/top)), int(Round(255*rgb[1]/top)),
			int(Round(255*rgb[2]/top)), int(Round(e.Glow*scale)))
	}
	if b.Len() == 0 {
		return "", nil
	}
	return "{\n\"classname\" \"info_texlights\"\n" + b.String() + "}\n", nil
}
