package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	sshpkg "github.com/kingknull/oblivrashell/internal/ssh"
	"github.com/kingknull/oblivrashell/internal/vault"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "connect":
		cmdConnect(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "exec":
		cmdExec(os.Args[2:])
	case "tunnel":
		cmdTunnel(os.Args[2:])
	case "vault":
		cmdVault(os.Args[2:])
	case "import":
		cmdImport(os.Args[2:])
	case "export":
		cmdExport(os.Args[2:])
	case "auth":
		cmdAuth(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	case "stream":
		cmdStream(os.Args[2:])
	case "version":
		fmt.Printf("Sovereign Terminal CLI v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Sovereign Terminal CLI

Usage: sovereign-cli <command> [options]

Commands:
  connect    Connect to an SSH host
  list       List hosts, sessions, or credentials
  exec       Execute a command on one or more hosts
  tunnel     Manage port forwarding tunnels
  vault      Manage the credential vault
  import     Import hosts from various sources
  export     Export hosts, settings, or reports
  auth       Authenticate with the REST API
  search     Search SIEM events via REST API
  stream     Stream real-time events via WebSocket
  version    Show version information
  help       Show this help

Examples:
  sovereign-cli connect my-server
  sovereign-cli connect user@host:22
  sovereign-cli list hosts
  sovereign-cli list hosts --tag production
  sovereign-cli exec "uptime" --hosts prod-1,prod-2
  sovereign-cli exec "df -h" --tag production
  sovereign-cli tunnel -L 8080:localhost:80 my-server
  sovereign-cli vault unlock
  sovereign-cli vault list
  sovereign-cli import ssh-config
  sovereign-cli export hosts --format json

Use "sovereign-cli <command> --help" for more information.`)
}

func cmdConnect(args []string) {
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	user := fs.String("u", "", "SSH username")
	port := fs.Int("p", 22, "SSH port")
	keyFile := fs.String("i", "", "Identity file (private key)")
	// password := fs.Bool("password", false, "Use password authentication")
	jumpHost := fs.String("J", "", "Jump host (user@host:port)")
	fs.Parse(args)

	target := fs.Arg(0)
	if target == "" {
		fmt.Fprintln(os.Stderr, "Error: no target specified")
		fmt.Fprintln(os.Stderr, "Usage: sovereign-cli connect [options] <host|alias>")
		os.Exit(1)
	}

	// Try to find host by alias in database
	db, err := openDB()
	v, _ := openVault()
	if err == nil {
		hostRepo := database.NewHostRepository(db, v)
		hosts, _ := hostRepo.Search(context.Background(), target)
		if len(hosts) > 0 {
			host := hosts[0]
			fmt.Printf("Connecting to %s (%s@%s:%d)...\n",
				host.Label, host.Username, host.Hostname, host.Port)

			cfg := sshpkg.DefaultConfig()
			cfg.Host = host.Hostname
			cfg.Port = host.Port
			cfg.Username = host.Username

			// Load credentials from vault if available
			if host.CredentialID != "" {
				loadCredentials(&cfg, host.CredentialID)
			} else {
				setDefaultAuth(&cfg, *keyFile)
			}

			connectAndShell(cfg)
			return
		}
		db.Close()
	}

	// Parse user@host:port
	hostname := target
	username := *user

	if strings.Contains(target, "@") {
		parts := strings.SplitN(target, "@", 2)
		username = parts[0]
		hostname = parts[1]
	}

	if strings.Contains(hostname, ":") {
		parts := strings.SplitN(hostname, ":", 2)
		hostname = parts[0]
		fmt.Sscanf(parts[1], "%d", port)
	}

	if username == "" {
		username = os.Getenv("USER")
	}

	fmt.Printf("Connecting to %s@%s:%d...\n", username, hostname, *port)

	cfg := sshpkg.DefaultConfig()
	cfg.Host = hostname
	cfg.Port = *port
	cfg.Username = username

	setDefaultAuth(&cfg, *keyFile)

	// Handle jump host
	if *jumpHost != "" {
		jh := parseJumpHost(*jumpHost)
		cfg.JumpHosts = []sshpkg.JumpHostConfig{jh}
	}

	connectAndShell(cfg)
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	tag := fs.String("tag", "", "Filter by tag")
	format := fs.String("format", "table", "Output format (table, json, csv)")
	fs.Parse(args)

	resource := fs.Arg(0)
	if resource == "" {
		resource = "hosts"
	}

	db, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	v, _ := openVault()
	switch resource {
	case "hosts":
		hostRepo := database.NewHostRepository(db, v)
		var hosts []database.Host

		if *tag != "" {
			hosts, err = hostRepo.GetByTag(context.Background(), *tag)
		} else {
			hosts, err = hostRepo.GetAll(context.Background())
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		switch *format {
		case "json":
			data, _ := json.MarshalIndent(hosts, "", "  ")
			fmt.Println(string(data))
		case "csv":
			fmt.Println("id,label,hostname,port,username,auth_method,tags,favorite")
			for _, h := range hosts {
				fmt.Printf("%s,%s,%s,%d,%s,%s,%s,%v\n",
					h.ID, h.Label, h.Hostname, h.Port,
					h.Username, h.AuthMethod,
					strings.Join(h.Tags, ";"), h.IsFavorite)
			}
		default:
			// Table format
			fmt.Printf("%-20s %-30s %-6s %-15s %-8s %s\n",
				"LABEL", "HOST", "PORT", "USER", "AUTH", "TAGS")
			fmt.Println(strings.Repeat("-", 100))
			for _, h := range hosts {
				fav := ""
				if h.IsFavorite {
					fav = "⭐ "
				}
				fmt.Printf("%-20s %-30s %-6d %-15s %-8s %s%s\n",
					fav+h.Label, h.Hostname, h.Port,
					h.Username, h.AuthMethod,
					strings.Join(h.Tags, ", "), "")
			}
			fmt.Printf("\n%d hosts\n", len(hosts))
		}

	case "sessions":
		sessRepo := database.NewSessionRepository(db)
		sessions, err := sessRepo.GetRecent(context.Background(), 50)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%-12s %-20s %-20s %-10s %-10s\n",
			"ID", "HOST", "STARTED", "DURATION", "STATUS")
		fmt.Println(strings.Repeat("-", 80))
		for _, s := range sessions {
			duration := time.Duration(s.DurationSeconds) * time.Second
			fmt.Printf("%-12s %-20s %-20s %-10s %-10s\n",
				s.ID[:12], s.HostID[:20],
				s.StartedAt.Format("2006-01-02 15:04"),
				duration.String(), s.Status)
		}

	case "tags":
		v, _ := openVault()
		hostRepo := database.NewHostRepository(db, v)
		tags, err := hostRepo.GetAllTags(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, tag := range tags {
			fmt.Println(tag)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown resource: %s\n", resource)
		fmt.Fprintln(os.Stderr, "Available: hosts, sessions, tags")
		os.Exit(1)
	}
}

func cmdExec(args []string) {
	fs := flag.NewFlagSet("exec", flag.ExitOnError)
	hosts := fs.String("hosts", "", "Comma-separated host names or IDs")
	tag := fs.String("tag", "", "Execute on all hosts with this tag")
	timeout := fs.Int("timeout", 30, "Timeout in seconds")
	// parallel := fs.Bool("parallel", true, "Execute in parallel")
	fs.Parse(args)

	command := fs.Arg(0)
	if command == "" {
		fmt.Fprintln(os.Stderr, "Error: no command specified")
		fmt.Fprintln(os.Stderr, "Usage: sovereign-cli exec [options] \"command\"")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	v, _ := openVault()
	hostRepo := database.NewHostRepository(db, v)
	var targetHosts []database.Host

	if *tag != "" {
		targetHosts, err = hostRepo.GetByTag(context.Background(), *tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *hosts != "" {
		hostNames := strings.Split(*hosts, ",")
		for _, name := range hostNames {
			name = strings.TrimSpace(name)
			results, _ := hostRepo.Search(context.Background(), name)
			if len(results) > 0 {
				targetHosts = append(targetHosts, results[0])
			} else {
				fmt.Fprintf(os.Stderr, "Warning: host '%s' not found\n", name)
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "Error: specify --hosts or --tag")
		os.Exit(1)
	}

	if len(targetHosts) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no matching hosts found")
		os.Exit(1)
	}

	fmt.Printf("Executing on %d hosts: %s\n\n", len(targetHosts), command)

	for _, host := range targetHosts {
		fmt.Printf("=== %s (%s) ===\n", host.Label, host.Hostname)

		cfg := sshpkg.DefaultConfig()
		cfg.Host = host.Hostname
		cfg.Port = host.Port
		cfg.Username = host.Username
		cfg.ConnectTimeout = time.Duration(*timeout) * time.Second

		setDefaultAuth(&cfg, "")

		client := sshpkg.NewClient(cfg)
		if err := client.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n\n", err)
			continue
		}

		output, err := client.ExecuteCommand(command)
		client.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
		}
		if len(output) > 0 {
			fmt.Print(string(output))
		}
		fmt.Println()
	}
}

func cmdTunnel(args []string) {
	fs := flag.NewFlagSet("tunnel", flag.ExitOnError)
	localForward := fs.String("L", "", "Local forward: local_port:remote_host:remote_port")
	remoteForward := fs.String("R", "", "Remote forward: remote_port:local_host:local_port")
	dynamicForward := fs.String("D", "", "Dynamic SOCKS: local_port")
	fs.Parse(args)

	target := fs.Arg(0)
	if target == "" {
		fmt.Fprintln(os.Stderr, "Error: no target host specified")
		os.Exit(1)
	}

	fmt.Printf("Setting up tunnel to %s...\n", target)

	if *localForward != "" {
		fmt.Printf("Local forward: %s\n", *localForward)
	}
	if *remoteForward != "" {
		fmt.Printf("Remote forward: %s\n", *remoteForward)
	}
	if *dynamicForward != "" {
		fmt.Printf("Dynamic SOCKS: %s\n", *dynamicForward)
	}

	// Keep running until interrupted
	fmt.Println("Tunnel active. Press Ctrl+C to stop.")
	select {}
}

func cmdVault(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: sovereign-cli vault <subcommand>")
		fmt.Println("Subcommands: unlock, lock, list, status")
		return
	}

	switch args[0] {
	case "status":
		v, err := openVault()
		if err != nil {
			fmt.Printf("Vault: NOT CONFIGURED\n")
			return
		}
		if v.IsSetup() {
			fmt.Printf("Vault: CONFIGURED (locked)\n")
		} else {
			fmt.Printf("Vault: NOT INITIALIZED\n")
		}
	case "list":
		fmt.Println("Vault entries (unlock required)")
	default:
		fmt.Fprintf(os.Stderr, "Unknown vault command: %s\n", args[0])
	}
}

func cmdImport(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: sovereign-cli import <source>")
		fmt.Println("Sources: ssh-config, terraform, ansible, known-hosts")
		return
	}

	source := args[0]
	fmt.Printf("Importing from %s...\n", source)

	switch source {
	case "ssh-config":
		entries, err := sshpkg.ParseSSHConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Found %d hosts in SSH config\n", len(entries))
		for _, e := range entries {
			fmt.Printf("  %s -> %s:%d (user: %s)\n",
				e.Alias, e.Hostname, e.Port, e.User)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown source: %s\n", source)
	}
}

func cmdExport(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: sovereign-cli export <resource> [options]")
		fmt.Println("Resources: hosts, settings, report")
		return
	}

	fmt.Printf("Exporting %s...\n", args[0])
}

// Helper functions

func openDB() (database.DatabaseStore, error) {
	v, err := openVault()
	if err != nil {
		return nil, err
	}

	db, err := database.New(platform.DatabasePath())
	if err != nil {
		return nil, err
	}

	// Try auto-unlock via keychain to get the DB key
	if uv, ok := v.(*vault.Vault); ok {
		if err := uv.UnlockWithKeychain(); err == nil {
			err = uv.AccessMasterKey(func(key []byte) error {
				return db.Open(platform.DatabasePath(), key)
			})
			if err != nil {
				return nil, fmt.Errorf("open encrypted database: %w", err)
			}
		}
	}

	return db, nil
}

func openVault() (vault.Provider, error) {
	platform.EnsureDirectories()
	l, _ := logger.New(logger.Config{
		Level:      logger.ErrorLevel,
		OutputPath: platform.LogPath(),
	})
	return vault.New(vault.Config{
		StorePath: platform.VaultPath(),
	}, l)
}

func setDefaultAuth(cfg *sshpkg.ConnectionConfig, keyFile string) {
	if keyFile != "" {
		keyData, err := os.ReadFile(keyFile)
		if err == nil {
			cfg.AuthMethod = sshpkg.AuthPublicKey
			cfg.PrivateKey = keyData
			return
		}
	}

	// Try default keys
	keyData, err := sshpkg.LoadDefaultKeys()
	if err == nil {
		cfg.AuthMethod = sshpkg.AuthPublicKey
		cfg.PrivateKey = keyData
	}
}

func loadCredentials(cfg *sshpkg.ConnectionConfig, _ string) {
	// Would unlock vault and load credentials
	// For now, fall back to default keys
	setDefaultAuth(cfg, "")
}

func connectAndShell(cfg sshpkg.ConnectionConfig) {
	client := sshpkg.NewClient(cfg)

	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected. Starting shell...")

	if err := client.StartShell(); err != nil {
		fmt.Fprintf(os.Stderr, "Shell failed: %v\n", err)
		client.Close()
		os.Exit(1)
	}

	// In a real CLI, we'd set up raw terminal mode
	// and pipe stdin/stdout
	fmt.Println("Interactive shell started. Press Ctrl+D to exit.")

	// Wait for session to end
	<-make(chan struct{})
}

func parseJumpHost(spec string) sshpkg.JumpHostConfig {
	jh := sshpkg.JumpHostConfig{
		Port:       22,
		AuthMethod: sshpkg.AuthPublicKey,
	}

	if strings.Contains(spec, "@") {
		parts := strings.SplitN(spec, "@", 2)
		jh.Username = parts[0]
		spec = parts[1]
	}

	if strings.Contains(spec, ":") {
		parts := strings.SplitN(spec, ":", 2)
		jh.Host = parts[0]
		fmt.Sscanf(parts[1], "%d", &jh.Port)
	} else {
		jh.Host = spec
	}

	// Try default keys
	keyData, _ := sshpkg.LoadDefaultKeys()
	jh.PrivateKey = keyData

	return jh
}
