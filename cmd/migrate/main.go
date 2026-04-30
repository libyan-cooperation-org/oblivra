// oblivra-migrate — schema-migration runner for OBLIVRA's persisted
// artifacts.
//
// Usage:
//
//	oblivra-migrate plan PATH       # show what would be migrated
//	oblivra-migrate run PATH        # migrate a single file
//	oblivra-migrate run --all DIR   # migrate every *.wal/*.log under DIR
//
// On a successful migration the original file is preserved as
// `<path>.pre-migrate` so the operator can roll back. No-op migrations leave
// the file untouched.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/migrate"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	all := false
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.BoolVar(&all, "all", false, "treat PATH as a directory and migrate every *.wal/*.log under it")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	if fs.NArg() < 1 {
		usage()
		os.Exit(2)
	}
	path := fs.Arg(0)

	switch cmd {
	case "plan":
		fmt.Printf("target schema version: %d\n", events.SchemaVersion)
		fmt.Println("(plan is informational only — actual schema changes are listed when an upgrader is registered)")
	case "run":
		if all {
			results, err := migrate.Dir(path)
			emit(results, err)
		} else {
			r, err := migrate.File(path)
			emit([]migrate.Stats{r}, err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  oblivra-migrate plan PATH
  oblivra-migrate run PATH
  oblivra-migrate run --all DIR`)
}

func emit(results []migrate.Stats, err error) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

