package main

import (
	"fmt"
	"os"
	"strings"
	"github.com/kingknull/oblivra/internal/sigma"
)
func main() {
	for _, dir := range os.Args[1:] {
		rules, errs := sigma.LoadDir(dir)
		fmt.Printf("\n=== %s ===\n", dir)
		fmt.Printf("loaded:  %d\nerrors:  %d\n", len(rules), len(errs))
		buckets := map[string]int{}
		for _, e := range errs {
			m := e.Error()
			k := "other"
			switch {
			case strings.Contains(m, "unsupported condition"): k = "unsupported"
			case strings.Contains(m, "matched no blocks"): k = "glob 0"
			case strings.Contains(m, "AND between blocks"): k = "AND blocks"
			case strings.Contains(m, "use `1 of"): k = "OR blocks"
			case strings.Contains(m, "no such block"): k = "no such block"
			case strings.Contains(m, "empty selection"): k = "empty selection"
			case strings.Contains(m, "non-map selection"): k = "non-map"
			case strings.Contains(m, "yaml:"): k = "yaml parse"
			case strings.Contains(m, "missing condition"): k = "no condition"
			}
			buckets[k]++
		}
		for k, v := range buckets {
			fmt.Printf("  %4d  %s\n", v, k)
		}
	}
}
