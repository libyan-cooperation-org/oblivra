package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/sigma"
)

// `oblivra-cli sigma import <src> <dst>` — audits a Sigma rule
// directory and (with --apply) copies just the loadable rules into
// <dst>. Useful for adopting a third-party pack like SigmaHQ's public
// corpus without manually filtering out the rules our parser rejects.
//
// Default mode is dry-run: prints the breakdown, writes nothing.
// --apply actually copies the loadable .yml/.yaml files preserving
// their relative paths under <src>, so a consumer can drop the result
// straight into the platform's sigma directory and the hot-reload
// watcher picks them up.
//
// Skipped rules are listed with their reason so an operator can
// decide whether to write a custom rule for the missing detections.
func sigmaCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "sigma: subcommand required (import)")
		os.Exit(2)
	}
	switch args[0] {
	case "import":
		sigmaImport(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "sigma: unknown subcommand", args[0])
		os.Exit(2)
	}
}

func sigmaImport(args []string) {
	fs := flag.NewFlagSet("sigma-import", flag.ExitOnError)
	apply := fs.Bool("apply", false, "actually copy loadable rules to <dst> (default: dry-run)")
	verbose := fs.Bool("v", false, "list every skipped file with its error")
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: oblivra-cli sigma import [--apply] [-v] <src> <dst>")
		os.Exit(2)
	}
	src := fs.Arg(0)
	dst := fs.Arg(1)

	report := importDir(src, dst, *apply, *verbose)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
	if report.Loaded == 0 {
		os.Exit(1)
	}
}

type sigmaImportReport struct {
	Src         string    `json:"src"`
	Dst         string    `json:"dst"`
	Apply       bool      `json:"apply"`
	GeneratedAt time.Time `json:"generatedAt"`
	Total       int       `json:"total"`
	Loaded      int       `json:"loaded"`
	LoadRate    string    `json:"loadRate"`
	Skipped     int       `json:"skipped"`
	Copied      int       `json:"copied,omitempty"`
	// breakdown by reason, biggest bucket first
	Reasons []reasonBucket `json:"reasons,omitempty"`
	// detail list (only when -v)
	SkippedFiles []skippedFile `json:"skippedFiles,omitempty"`
}

type reasonBucket struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type skippedFile struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

func importDir(src, dst string, apply, verbose bool) sigmaImportReport {
	r := sigmaImportReport{
		Src: src, Dst: dst, Apply: apply,
		GeneratedAt: time.Now().UTC(),
	}

	if apply {
		if err := os.MkdirAll(dst, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir dst:", err)
			os.Exit(1)
		}
	}

	buckets := map[string]int{}
	_ = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		r.Total++
		_, perr := sigma.LoadFile(path)
		if perr != nil {
			r.Skipped++
			reason := classify(perr.Error())
			buckets[reason]++
			if verbose {
				rel, _ := filepath.Rel(src, path)
				r.SkippedFiles = append(r.SkippedFiles, skippedFile{File: rel, Error: perr.Error()})
			}
			return nil
		}
		r.Loaded++
		if apply {
			rel, err := filepath.Rel(src, path)
			if err != nil {
				rel = filepath.Base(path)
			}
			out := filepath.Join(dst, rel)
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err == nil {
				if err := copyFile(path, out); err == nil {
					r.Copied++
				}
			}
		}
		return nil
	})

	for k, v := range buckets {
		r.Reasons = append(r.Reasons, reasonBucket{Reason: k, Count: v})
	}
	sort.Slice(r.Reasons, func(i, j int) bool { return r.Reasons[i].Count > r.Reasons[j].Count })

	if r.Total > 0 {
		r.LoadRate = fmt.Sprintf("%.1f%%", float64(r.Loaded)*100/float64(r.Total))
	}
	return r
}

// classify maps a sigma loader error to a short reason label so the
// summary's reason buckets are scannable. Mirrors the categories the
// loader explicitly produces.
func classify(msg string) string {
	switch {
	case strings.Contains(msg, "AND between blocks"):
		return "AND-between-blocks"
	case strings.Contains(msg, "use `1 of"):
		return "OR-between-blocks"
	case strings.Contains(msg, "matched no blocks"):
		return "glob-pattern-matched-zero"
	case strings.Contains(msg, "no such block"):
		return "term-references-missing-block"
	case strings.Contains(msg, "non-map selection"):
		return "non-map-selection"
	case strings.Contains(msg, "empty selection"):
		return "empty-selection"
	case strings.Contains(msg, "missing condition"):
		return "missing-condition"
	case strings.Contains(msg, "missing title"):
		return "missing-title"
	case strings.Contains(msg, "yaml:"):
		return "yaml-parse-error"
	case strings.Contains(msg, "unsupported condition"):
		return "unsupported-condition"
	default:
		return "other"
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
