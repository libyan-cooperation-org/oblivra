// clean is a cross-platform `make clean`. Removes build artifacts and rebuilds
// the placeholder webassets/dist/index.html so the Go embed always compiles.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const placeholder = `<!doctype html>
<html><head><meta charset="utf-8"><title>OBLIVRA</title></head>
<body><p>Build the frontend: <code>cd frontend &amp;&amp; npm install &amp;&amp; npm run build</code></p></body>
</html>
`

func main() {
	for _, p := range []string{
		"build/bin",
		"webassets/dist",
		"frontend/node_modules",
	} {
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := os.MkdirAll("webassets/dist", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join("webassets", "dist", "index.html"), []byte(placeholder), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
