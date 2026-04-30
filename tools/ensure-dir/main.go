// ensure-dir is a cross-platform replacement for `mkdir -p`. It exists because
// Task's embedded shell (mvdan/sh) shells out to `mkdir`, which isn't in PATH
// on stock Windows. Usage: `go run ./tools/ensure-dir <path> [<path>...]`.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ensure-dir <path> [path ...]")
		os.Exit(2)
	}
	for _, p := range os.Args[1:] {
		if err := os.MkdirAll(p, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
