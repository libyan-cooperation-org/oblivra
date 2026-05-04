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

// `oblivra-agent setup` — interactive first-run wizard.
// Prompts for server URL, token file path, hostname, and which log sources
// to enable. Writes the config, locks the tamper fingerprint, and explains
// the next steps.
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
		requireAdminPassword(defaultStateDir(), "re-run setup")
	}

	in := bufio.NewReader(os.Stdin)

	serverURL := promptURL(in, "OBLIVRA server URL", "https://oblivra.internal")
	tokenFile := prompt(in, "Path to 0600-mode file holding the agent bearer token", tokenFilePath())
	host, _ := os.Hostname()
	hostname := prompt(in, "Hostname this agent identifies as", host)
	tenant := prompt(in, "Tenant ID", "default")

	fmt.Println()
	fmt.Println("Security defaults:")
	signEvents := askYN("  Enable per-event ed25519 signing", true)
	redact := askYN("  Enable edge DLP (mask PAN/SSN/tokens before shipping)", true)
	localRules := askYN("  Enable local pre-detection priority queue", true)
	heartbeat := askYN("  Send rich heartbeats every 30s", true)

	fmt.Println()
	fmt.Println("Discovering log sources…")
	candidates := discoverLogSources()
	if len(candidates) == 0 {
		fmt.Println("  (none found — you can add inputs: manually)")
	} else {
		fmt.Println("Found:")
		for i, c := range candidates {
			fmt.Printf("  [%d] %s\n", i+1, c.display())
		}
	}
	picks := promptCSV(in, "Enter numbers to enable (comma-separated, 'all', or blank to skip)", "all")
	enabled := pickSources(candidates, picks)

	fmt.Println()
	startBeginning := askYN("Backfill from day zero (ship all existing logs/events on first run)", true)

	fmt.Println()
	fmt.Println("Local admin password (gates `setup`, `reload`, loopback /status).")
	fmt.Println("Leave blank to skip.")
	adminPassword := promptSecret(in, "Admin password (≥8 chars, blank to skip)")
	if adminPassword != "" {
		confirm := promptSecret(in, "Confirm password")
		if confirm != adminPassword {
			fmt.Fprintln(os.Stderr, "  passwords don't match — aborting.")
			os.Exit(1)
		}
	}

	cfg := buildSetupConfig(setupAnswers{
		ConfigPath: out,
		ServerURL:  serverURL,
		TokenFile:  tokenFile,
		Hostname:   hostname,
		Tenant:     tenant,
		SignEvents: signEvents,
		Redact:     redact,
		LocalRules: localRules,
		Heartbeat:  heartbeat,
		StartBegin: startBeginning,
		Sources:    enabled,
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

	if adminPassword != "" {
		stateDir := defaultStateDir()
		if err := SetAdminPassword(stateDir, adminPassword); err != nil {
			fmt.Fprintln(os.Stderr, "  warning: failed to write admin password hash:", err)
		} else {
			fmt.Printf("✓ admin password hashed (Argon2id) → %s\n", passwordPath(stateDir))
		}
	}

	fmt.Printf(`
Next steps:
  1. echo 'YOUR_TOKEN' | sudo tee %s && sudo chmod 0600 %s
  2. oblivra-agent test  --config %s
  3. oblivra-agent run   --config %s

The config fingerprint is locked on first successful run. Edits without
--acknowledge-config-change will refuse to start (tamper tripwire).
`, tokenFile, tokenFile, out, out)
}

// ── Log source discovery ───────────────────────────────────────────────────

// logSource is a discovered log source (either a file path or a winlog channel).
type logSource struct {
	inputType  string // "file", "winlog", "journald"
	path       string // file path or UDP addr
	channel    string // winlog channel name
	sourceType string
	label      string
}

func (s logSource) display() string {
	switch s.inputType {
	case "winlog":
		return fmt.Sprintf("winlog channel: %s (%s)", s.channel, s.sourceType)
	case "journald":
		return "systemd journal (journald)"
	default:
		return s.path
	}
}

// discoverLogSources returns all log sources that exist on this host.
// On Windows it queries winlog channels; on Linux it finds journald + files.
func discoverLogSources() []logSource {
	switch runtime.GOOS {
	case "windows":
		return discoverWindowsSources()
	case "linux":
		return discoverLinuxSources()
	default:
		return discoverGenericSources()
	}
}

func discoverWindowsSources() []logSource {
	// Priority-ordered channel list. discoverWinlogChannels() probes each
	// with `wevtutil gl` so we only offer channels that actually exist.
	type chanDef struct {
		channel    string
		sourceType string
		label      string
	}
	want := []chanDef{
		{"Security", "windows:security", "win-security"},
		{"System", "windows:system", "win-system"},
		{"Application", "windows:application", "win-application"},
		{"Microsoft-Windows-Sysmon/Operational", "windows:sysmon", "win-sysmon"},
		{"Microsoft-Windows-PowerShell/Operational", "windows:powershell", "win-powershell"},
		{"Microsoft-Windows-Windows Defender/Operational", "windows:defender", "win-defender"},
		{"Microsoft-Windows-TaskScheduler/Operational", "windows:scheduler", "win-scheduler"},
		{"Microsoft-Windows-TerminalServices-RemoteConnectionManager/Operational", "windows:rdp", "win-rdp"},
		{"Microsoft-Windows-WMI-Activity/Operational", "windows:wmi", "win-wmi"},
	}

	available := make(map[string]bool)
	for _, ch := range discoverWinlogChannels() {
		available[ch] = true
	}

	var out []logSource
	for _, w := range want {
		if available[w.channel] {
			out = append(out, logSource{
				inputType:  "winlog",
				channel:    w.channel,
				sourceType: w.sourceType,
				label:      w.label,
			})
		}
	}
	return out
}

func discoverLinuxSources() []logSource {
	var out []logSource

	// Always offer journald on Linux if journalctl is present.
	out = append(out, logSource{
		inputType:  "journald",
		sourceType: "linux:journal",
		label:      "journal",
	})

	// File candidates — only include those that exist.
	type fileDef struct {
		path       string
		sourceType string
		label      string
	}
	files := []fileDef{
		{"/var/log/auth.log", "linux:auth", "auth"},
		{"/var/log/secure", "linux:auth", "auth"},
		{"/var/log/syslog", "linux:syslog", "syslog"},
		{"/var/log/messages", "linux:syslog", "syslog"},
		{"/var/log/kern.log", "linux:kernel", "kernel"},
		{"/var/log/dmesg", "linux:kernel", "dmesg"},
		{"/var/log/audit/audit.log", "linux:auditd", "auditd"},
		{"/var/log/nginx/access.log", "nginx:access", "nginx-access"},
		{"/var/log/nginx/error.log", "nginx:error", "nginx-error"},
		{"/var/log/apache2/access.log", "apache:access", "apache-access"},
		{"/var/log/apache2/error.log", "apache:error", "apache-error"},
		{"/var/log/httpd/access_log", "apache:access", "httpd-access"},
		{"/var/log/httpd/error_log", "apache:error", "httpd-error"},
		{"/var/log/mysql/error.log", "mysql", "mysql"},
		{"/var/log/postgresql/postgresql*.log", "postgresql", "postgresql"},
		{"/var/log/redis/redis-server.log", "redis", "redis"},
	}
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f.label] {
			continue // skip duplicates (e.g. both auth.log and secure)
		}
		if _, err := os.Stat(f.path); err == nil {
			out = append(out, logSource{
				inputType:  "file",
				path:       f.path,
				sourceType: f.sourceType,
				label:      f.label,
			})
			seen[f.label] = true
		}
	}
	return out
}

func discoverGenericSources() []logSource {
	var out []logSource
	for _, p := range []struct{ path, st, label string }{
		{"/var/log/system.log", "macos:system", "system"},
		{"/var/log/install.log", "macos:install", "install"},
	} {
		if _, err := os.Stat(p.path); err == nil {
			out = append(out, logSource{inputType: "file", path: p.path, sourceType: p.st, label: p.label})
		}
	}
	return out
}

func pickSources(all []logSource, picks string) []logSource {
	picks = strings.TrimSpace(strings.ToLower(picks))
	if picks == "" {
		return nil
	}
	if picks == "all" {
		return all
	}
	out := make([]logSource, 0, len(all))
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

// ── Config writer ──────────────────────────────────────────────────────────

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
	Sources    []logSource
}

func buildSetupConfig(a setupAnswers) string {
	startFrom := "tail"
	if a.StartBegin {
		startFrom = "beginning"
	}
	heartbeat := "false"
	if a.Heartbeat {
		heartbeat = "true"
	}

	var inputs strings.Builder
	for _, s := range a.Sources {
		switch s.inputType {
		case "winlog":
			fmt.Fprintf(&inputs,
				"  - type: winlog\n    channel: %q\n    sourceType: %q\n    startFrom: %s\n    label: %q\n\n",
				s.channel, s.sourceType, startFrom, s.label)
		case "journald":
			fmt.Fprintf(&inputs,
				"  - type: journald\n    sourceType: %q\n    startFrom: %s\n    label: %q\n\n",
				s.sourceType, startFrom, s.label)
		default: // file
			fmt.Fprintf(&inputs,
				"  - type: file\n    path: %q\n    sourceType: %q\n    startFrom: %s\n    label: %q\n\n",
				s.path, s.sourceType, startFrom, s.label)
		}
	}

	// On Linux, always include journald even if the user skipped all files.
	if runtime.GOOS == "linux" {
		hasJournald := false
		for _, s := range a.Sources {
			if s.inputType == "journald" {
				hasJournald = true
				break
			}
		}
		if !hasJournald {
			fmt.Fprintf(&inputs,
				"  - type: journald\n    sourceType: \"linux:journal\"\n    startFrom: %s\n    label: \"journal\"\n\n",
				startFrom)
		}
	}

	if inputs.Len() == 0 {
		inputs.WriteString("  # Add inputs here. Run `oblivra-agent init` for annotated examples.\n")
	}

	return fmt.Sprintf(`# Generated by oblivra-agent setup — %s
# Edits without --acknowledge-config-change will refuse to start (tamper tripwire).

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
%s`,
		time.Now().UTC().Format(time.RFC3339),
		a.ServerURL, a.TokenFile, a.Hostname, a.Tenant,
		a.SignEvents, a.Redact, a.LocalRules,
		heartbeat,
		inputs.String(),
	)
}

// ── Prompts (unchanged from original) ─────────────────────────────────────

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

func promptSecret(r *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	if v, ok := readSecretNoEcho(); ok {
		fmt.Println()
		return v
	}
	fmt.Print(" [echo not suppressed]: ")
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(strings.TrimRight(line, "\r\n"))
}

// discoverLogPaths is kept for backward compatibility with runTest which
// calls it. It delegates to discoverLogSources and returns file paths only.
func discoverLogPaths() []string {
	var out []string
	for _, s := range discoverLogSources() {
		if s.inputType == "file" && s.path != "" {
			out = append(out, s.path)
		}
	}
	return out
}

// sourceTypeFor is kept for backward compatibility.
func sourceTypeFor(path string) string {
	for _, s := range discoverLinuxSources() {
		if s.path == path {
			return s.sourceType
		}
	}
	return "unknown"
}
