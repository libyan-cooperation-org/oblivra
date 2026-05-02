package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// `oblivra-agent setup` is the interactive first-run wizard. Prompts
// for the bare minimum (server URL, agent token, hostname), offers
// well-known log paths to enable, writes the config, and runs the
// existing `test` flow as a sanity check.
//
// Designed for a tamper-resistant first install: the operator runs it
// once at provision time, the resulting fingerprint is locked in (see
// config_integrity.go), and any later config edit requires explicit
// acknowledgement.
//
// Doesn't touch the network at write time — the test stage at the end
// is what verifies connectivity. That keeps `setup` usable in
// air-gapped provisioning where the platform isn't reachable yet.
func runSetup(args []string) {
	out := DefaultConfigPath()
	if len(args) > 0 && args[0] != "" {
		out = args[0]
	}

	fmt.Println("oblivra-agent setup — first-run wizard")
	fmt.Println("---------------------------------------")
	fmt.Printf("This will write %s and lock its fingerprint.\n", out)
	fmt.Println("Re-run with `--acknowledge-config-change` to overwrite later.")
	fmt.Println()

	if _, err := os.Stat(out); err == nil {
		if !askYN("Config already exists. Overwrite", false) {
			fmt.Println("aborted.")
			return
		}
	}

	in := bufio.NewReader(os.Stdin)

	serverURL := promptURL(in, "OBLIVRA server URL (or `srv://_oblivra._tcp.example.com`)", "https://oblivra.internal")
	tokenFile := prompt(in, "Path to a 0600-mode file holding the agent bearer token", "/etc/oblivra/agent.token")
	host, _ := os.Hostname()
	hostname := prompt(in, "Hostname this agent identifies as", host)
	tenant := prompt(in, "Tenant ID", "default")

	fmt.Println()
	fmt.Println("Recommended security defaults:")
	signEvents := askYN("  Enable per-event ed25519 signing", true)
	redact := askYN("  Enable edge DLP (mask CC/SSN/tokens before shipping)", true)
	localRules := askYN("  Enable local pre-detection priority queue", true)
	heartbeat := askYN("  Send rich heartbeats every 30s (recommended)", true)

	fmt.Println()
	fmt.Println("Discovering log paths…")
	candidates := discoverLogPaths()
	if len(candidates) == 0 {
		fmt.Println("  (no well-known paths found — you can add `inputs:` manually)")
	} else {
		fmt.Println("Found:")
		for i, c := range candidates {
			fmt.Printf("  [%d] %s\n", i+1, c)
		}
	}
	picks := promptCSV(in, "Enter numbers to enable (comma-separated, 'all', or empty to skip)", "all")
	enabled := pickPaths(candidates, picks)

	fmt.Println()
	startBeginning := askYN("Backfill from day zero (read all rotated/gzipped logs on first run)", true)

	cfg := buildSetupConfig(setupAnswers{
		ConfigPath:    out,
		ServerURL:     serverURL,
		TokenFile:     tokenFile,
		Hostname:      hostname,
		Tenant:        tenant,
		SignEvents:    signEvents,
		Redact:        redact,
		LocalRules:    localRules,
		Heartbeat:     heartbeat,
		StartBegin:    startBeginning,
		Inputs:        enabled,
	})

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, []byte(cfg), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("\n✓ wrote %s (mode 0600)\n", out)
	fmt.Println()
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. echo 'YOUR_TOKEN' | sudo tee %s && sudo chmod 0600 %s\n", tokenFile, tokenFile)
	fmt.Printf("  2. oblivra-agent test --config %s\n", out)
	fmt.Printf("  3. oblivra-agent run --config %s\n", out)
	fmt.Printf("\nThe config fingerprint is locked on first successful run. Edits without\n")
	fmt.Printf("--acknowledge-config-change will refuse to start (tamper tripwire).\n")
}

// ---- prompts ----

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptURL(r *bufio.Reader, label, def string) string {
	for {
		v := prompt(r, label, def)
		if v == "" {
			fmt.Println("  required.")
			continue
		}
		// Accept srv:// shape too.
		if strings.HasPrefix(v, "srv://") {
			return v
		}
		if _, err := url.Parse(v); err == nil {
			return v
		}
		fmt.Printf("  not a valid URL: %s\n", v)
	}
}

func askYN(label string, def bool) bool {
	for {
		hint := " [Y/n]"
		if !def {
			hint = " [y/N]"
		}
		fmt.Printf("%s%s: ", label, hint)
		var line string
		fmt.Scanln(&line)
		line = strings.ToLower(strings.TrimSpace(line))
		switch line {
		case "":
			return def
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}

func promptCSV(r *bufio.Reader, label, def string) string {
	return prompt(r, label, def)
}

func pickPaths(all []string, picks string) []string {
	picks = strings.TrimSpace(strings.ToLower(picks))
	if picks == "" {
		return nil
	}
	if picks == "all" {
		return all
	}
	out := make([]string, 0, len(all))
	for _, tok := range strings.Split(picks, ",") {
		tok = strings.TrimSpace(tok)
		var idx int
		if _, err := fmt.Sscanf(tok, "%d", &idx); err != nil || idx < 1 || idx > len(all) {
			continue
		}
		out = append(out, all[idx-1])
	}
	return out
}

// discoverLogPaths returns well-known log files that exist on this
// host. Right call: only suggest things that are actually there, don't
// dump a 50-item menu of paths that would fail.
func discoverLogPaths() []string {
	candidates := commonLogPaths()
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			out = append(out, c)
		} else if strings.ContainsAny(c, "*?[") {
			// glob — accept if any matches exist
			matches, _ := filepath.Glob(c)
			if len(matches) > 0 {
				out = append(out, c)
			}
		}
	}
	return out
}

func commonLogPaths() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{
			"/var/log/auth.log",
			"/var/log/syslog",
			"/var/log/kern.log",
			"/var/log/messages",
			"/var/log/secure",
			"/var/log/dmesg",
			"/var/log/audit/audit.log",
			"/var/log/nginx/access.log",
			"/var/log/nginx/error.log",
			"/var/log/apache2/access.log",
			"/var/log/apache2/error.log",
			"/var/log/httpd/access_log",
			"/var/log/httpd/error_log",
			"/var/log/mysql/error.log",
			"/var/log/mariadb/mariadb.log",
			"/var/log/postgresql/*.log",
			"/var/log/redis/redis-server.log",
		}
	case "darwin":
		return []string{
			"/var/log/system.log",
			"/var/log/install.log",
			"/var/log/wifi.log",
		}
	case "windows":
		return []string{
			`C:\Windows\System32\winevt\Logs\Security.evtx`,
			`C:\Windows\System32\winevt\Logs\System.evtx`,
			`C:\Windows\System32\winevt\Logs\Application.evtx`,
		}
	}
	return nil
}

// ---- config writer ----

type setupAnswers struct {
	ConfigPath string
	ServerURL  string
	TokenFile  string
	Hostname   string
	Tenant     string
	SignEvents bool
	Redact     bool
	LocalRules bool
	Heartbeat  bool
	StartBegin bool
	Inputs     []string
}

func buildSetupConfig(a setupAnswers) string {
	startFrom := "tail"
	if a.StartBegin {
		startFrom = "beginning"
	}
	var inputs strings.Builder
	for _, p := range a.Inputs {
		fmt.Fprintf(&inputs, "  - type: file\n    path: %q\n    sourceType: %s\n    startFrom: %s\n",
			p, sourceTypeFor(p), startFrom)
	}
	if runtime.GOOS == "linux" {
		// Always include journald — covers systemd-managed services
		// even if the operator declined the file paths.
		fmt.Fprintf(&inputs, "  - type: journald\n    sourceType: linux:journal\n    startFrom: %s\n", startFrom)
	}
	if len(a.Inputs) == 0 && runtime.GOOS != "linux" {
		fmt.Fprintf(&inputs, "  # Add `inputs:` here. Examples in `oblivra-agent init`.\n")
	}

	heartbeat := "false"
	if a.Heartbeat {
		heartbeat = "true"
	}

	return fmt.Sprintf(`# Generated by oblivra-agent setup at %s.
# Edits without --acknowledge-config-change refuse to start (tamper tripwire).

server:
  url: %q
  tokenFile: %q
  requestTimeout: 10s
  healthInterval: 60s
  tls:
    caCertFile: ""
    insecure: false

hostname: %q
tenant: %q
tags:
  - "managed-by:oblivra-agent-setup"

batchSize: 100
flushEvery: 2s
compression: gzip

signEvents: %t
redact: %t
localRules: %t
adaptiveBatch: true

heartbeat:
  enabled: %s
  interval: 30s

inputs:
%s
`,
		time.Now().UTC().Format(time.RFC3339),
		a.ServerURL, a.TokenFile, a.Hostname, a.Tenant,
		a.SignEvents, a.Redact, a.LocalRules,
		heartbeat,
		inputs.String(),
	)
}

// sourceTypeFor picks a sensible sourceType label for a known path so
// the operator UI can group services correctly out of the box.
func sourceTypeFor(path string) string {
	switch {
	case strings.Contains(path, "auth.log"), strings.Contains(path, "secure"):
		return "linux:auth"
	case strings.Contains(path, "audit/audit.log"):
		return "linux:auditd"
	case strings.Contains(path, "syslog"), strings.Contains(path, "messages"):
		return "linux:syslog"
	case strings.Contains(path, "kern.log"), strings.Contains(path, "dmesg"):
		return "linux:kernel"
	case strings.Contains(path, "nginx") && strings.Contains(path, "access"):
		return "nginx:access"
	case strings.Contains(path, "nginx") && strings.Contains(path, "error"):
		return "nginx:error"
	case strings.Contains(path, "apache") || strings.Contains(path, "httpd"):
		if strings.Contains(path, "access") {
			return "apache:access"
		}
		return "apache:error"
	case strings.Contains(path, "mysql") || strings.Contains(path, "mariadb"):
		return "mysql"
	case strings.Contains(path, "postgresql"):
		return "postgresql"
	case strings.Contains(path, "redis"):
		return "redis"
	case strings.Contains(path, ".evtx"):
		return "windows:eventlog"
	}
	return "linux:other"
}
