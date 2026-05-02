package main

import (
	"bufio"
	"fmt"
	"os"
)

// `oblivra-agent password {set|clear|test}` — Splunk-UF-style admin
// password management.
//
//   set    — prompt for a new password (no-echo) + confirm. Verifies
//            current password first if one is already set (rotation).
//   clear  — remove the password (requires current password).
//   test   — verify a password against the stored hash; exit 0 on
//            match, 1 on mismatch. Useful for non-interactive tests.
func runPasswordCmd(args []string) {
	stateDir := defaultStateDir()
	if len(args) == 0 {
		passwordUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "set":
		passwordSet(stateDir)
	case "clear":
		passwordClear(stateDir)
	case "test":
		passwordTest(stateDir)
	default:
		passwordUsage()
		os.Exit(2)
	}
}

func passwordUsage() {
	fmt.Fprintln(os.Stderr, `oblivra-agent password — admin password management

Subcommands:
  set     Set or rotate the admin password (prompts; rotation requires the current one)
  clear   Remove the password (requires the current one)
  test    Verify a password against the stored hash (exit 0 = match)

The password gates: setup, reload, encrypt-config, and the loopback
/status endpoint. Sources for the password (priority order):
  1. OBLIVRA_AGENT_ADMIN_PASSWORD env var
  2. OBLIVRA_AGENT_ADMIN_PASSWORD_FILE (mode 0600)
  3. interactive prompt (no-echo)`)
}

func passwordSet(stateDir string) {
	if HasAdminPassword(stateDir) {
		requireAdminPassword(stateDir, "rotate the admin password")
	}
	r := bufio.NewReader(os.Stdin)
	pw := promptSecret(r, "New admin password (≥8 chars)")
	if len(pw) < 8 {
		fmt.Fprintln(os.Stderr, "password too short.")
		os.Exit(1)
	}
	confirm := promptSecret(r, "Confirm password")
	if confirm != pw {
		fmt.Fprintln(os.Stderr, "passwords don't match.")
		os.Exit(1)
	}
	if err := SetAdminPassword(stateDir, pw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("✓ admin password updated (Argon2id) → %s\n", passwordPath(stateDir))
}

func passwordClear(stateDir string) {
	if !HasAdminPassword(stateDir) {
		fmt.Println("no admin password set; nothing to clear.")
		return
	}
	requireAdminPassword(stateDir, "clear the admin password")
	if err := ClearAdminPassword(stateDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✓ admin password cleared.")
}

func passwordTest(stateDir string) {
	if !HasAdminPassword(stateDir) {
		fmt.Println("no admin password set.")
		os.Exit(2)
	}
	pw := readPasswordFromEnv()
	if pw == "" {
		if v, ok := readSecretNoEcho(); ok {
			pw = v
		}
	}
	if pw == "" {
		fmt.Fprintln(os.Stderr, "no password supplied (env or stdin).")
		os.Exit(2)
	}
	if err := VerifyAdminPassword(stateDir, pw); err != nil {
		if IsBadPassword(err) {
			fmt.Println("INVALID")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Println("OK")
}
