// bspinfo: quick facts about a compiled map.
//
//	bspinfo lightbudget MAP.bsp [N]    light samples per texture, biggest first (top N, default 12)
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"halflifemaps/tools/bsp/bspfile"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "lightbudget" {
		fmt.Fprintln(os.Stderr, "usage: bspinfo lightbudget MAP.bsp [N]")
		os.Exit(2)
	}
	top := 12
	if len(os.Args) > 3 {
		n, err := strconv.Atoi(os.Args[3])
		if err != nil || n < 1 {
			fmt.Fprintln(os.Stderr, "N must be a positive number")
			os.Exit(2)
		}
		top = n
	}
	m, err := bspfile.Load(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	count := map[string]int{}
	for i, f := range m.Faces {
		ex := m.FExt[i]
		if ex == nil {
			continue
		}
		name := m.Textures[m.TexInfo[f.TexInfo].Miptex].Name
		count[name] += (ex.Size[0]/16 + 1) * (ex.Size[1]/16 + 1)
	}
	names := make([]string, 0, len(count))
	for n := range count {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if count[names[i]] != count[names[j]] {
			return count[names[i]] > count[names[j]]
		}
		return names[i] < names[j]
	})
	for i, n := range names {
		if i == top {
			break
		}
		fmt.Printf("%-16s %8d\n", n, count[n])
	}
	fmt.Printf("%-16s %8d / %d\n", "total", m.Lux, bspfile.Limits[0].Max)
}
