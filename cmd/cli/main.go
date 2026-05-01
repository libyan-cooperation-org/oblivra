// oblivra-cli — thin REST client for the headless server.
//
// Usage:
//
//	oblivra-cli ping
//	oblivra-cli stats
//	oblivra-cli ingest --host web-01 --severity warning --message "..."
//	oblivra-cli search [--q "severity:error"] [--limit 50]
//	oblivra-cli alerts [--limit 25]
//	oblivra-cli audit verify
//	oblivra-cli audit log [--limit 25]
//
// Configure with OBLIVRA_ADDR (default http://localhost:8080) and OBLIVRA_TOKEN.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "ping":
		do("GET", "/api/v1/system/ping", nil)
	case "stats":
		do("GET", "/api/v1/siem/stats", nil)
	case "ingest":
		ingest(os.Args[2:])
	case "search":
		search(os.Args[2:])
	case "alerts":
		alerts(os.Args[2:])
	case "audit":
		audit(os.Args[2:])
	case "fleet":
		do("GET", "/api/v1/agent/fleet", nil)
	case "rules":
		do("GET", "/api/v1/detection/rules", nil)
	case "intel":
		do("GET", "/api/v1/threatintel/indicators", nil)
	case "backup":
		backupCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `oblivra-cli — REST client

Commands:
  ping                     Liveness check
  stats                    Ingest pipeline stats
  ingest                   Send a single event (flags below)
  search [--q ...]         Query events (q = Bleve query string)
  alerts [--limit N]       Recent alerts
  audit verify             Verify Merkle audit chain
  audit log [--limit N]    Audit log entries
  fleet                    Registered agents
  rules                    Detection rules
  intel                    Threat-intel indicators
  backup verify <path>     Offline integrity check on a backup directory

Env: OBLIVRA_ADDR=http://localhost:8080  OBLIVRA_TOKEN=<api key>`)
}

func ingest(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	host := fs.String("host", "", "host id")
	severity := fs.String("severity", "info", "severity")
	eventType := fs.String("type", "manual", "event type")
	message := fs.String("message", "", "message body (required)")
	_ = fs.Parse(args)
	if *message == "" {
		fmt.Fprintln(os.Stderr, "error: --message is required")
		os.Exit(2)
	}
	body := map[string]any{
		"source":    "rest",
		"hostId":    *host,
		"severity":  *severity,
		"eventType": *eventType,
		"message":   *message,
	}
	do("POST", "/api/v1/siem/ingest", body)
}

func search(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	q := fs.String("q", "", "Bleve query string")
	limit := fs.Int("limit", 50, "max results")
	newest := fs.Bool("newest", true, "newest first")
	_ = fs.Parse(args)
	v := url.Values{}
	if *q != "" {
		v.Set("q", *q)
	}
	v.Set("limit", fmt.Sprintf("%d", *limit))
	if *newest {
		v.Set("newestFirst", "true")
	}
	do("GET", "/api/v1/siem/search?"+v.Encode(), nil)
}

func alerts(args []string) {
	fs := flag.NewFlagSet("alerts", flag.ExitOnError)
	limit := fs.Int("limit", 25, "max alerts")
	_ = fs.Parse(args)
	do("GET", fmt.Sprintf("/api/v1/alerts?limit=%d", *limit), nil)
}

func audit(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "audit: need verify | log")
		os.Exit(2)
	}
	switch args[0] {
	case "verify":
		do("GET", "/api/v1/audit/verify", nil)
	case "log":
		fs := flag.NewFlagSet("audit-log", flag.ExitOnError)
		limit := fs.Int("limit", 25, "max entries")
		_ = fs.Parse(args[1:])
		do("GET", fmt.Sprintf("/api/v1/audit/log?limit=%d", *limit), nil)
	default:
		fmt.Fprintln(os.Stderr, "audit: unknown subcommand", args[0])
		os.Exit(2)
	}
}

func do(method, path string, body any) {
	addr := os.Getenv("OBLIVRA_ADDR")
	if addr == "" {
		addr = "http://localhost:8080"
	}
	addr = strings.TrimRight(addr, "/")
	url := addr + path

	var bodyR io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			die(err)
		}
		bodyR = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyR)
	if err != nil {
		die(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := os.Getenv("OBLIVRA_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		die(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", method, path, resp.Status)
		os.Stderr.Write(out)
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
	// Pretty-print if JSON.
	var pretty any
	if json.Unmarshal(out, &pretty) == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(pretty)
	} else {
		os.Stdout.Write(out)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
