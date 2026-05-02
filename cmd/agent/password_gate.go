package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/term"
)

// requireAdminPassword checks the admin password before allowing a
// sensitive subcommand. Three input paths, in priority:
//
//   1. OBLIVRA_AGENT_ADMIN_PASSWORD env var
//   2. OBLIVRA_AGENT_ADMIN_PASSWORD_FILE (mode 0600 file)
//   3. interactive prompt (no-echo)
//
// `purpose` is shown to the operator so they know what they're
// authorizing — same UF idea: "agent restart requires password".
//
// If no password is set on this host (HasAdminPassword returns false),
// the function is a no-op — the agent only enforces the gate when an
// operator opted in via setup.
func requireAdminPassword(stateDir, purpose string) {
	if !HasAdminPassword(stateDir) {
		return
	}
	pw := readPasswordFromEnv()
	if pw == "" {
		// Interactive prompt with no-echo when on a TTY.
		fmt.Fprintf(os.Stderr, "admin password required to %s.\n", purpose)
		if isatty(os.Stdin) {
			fmt.Fprint(os.Stderr, "password: ")
			if v, ok := readSecretNoEcho(); ok {
				fmt.Fprintln(os.Stderr)
				pw = v
			}
		}
		if pw == "" {
			r := bufio.NewReader(os.Stdin)
			fmt.Fprint(os.Stderr, "password (echoed — non-TTY): ")
			line, _ := r.ReadString('\n')
			pw = stringTrim(line)
		}
	}
	_ = runtime.GOOS
	if err := VerifyAdminPassword(stateDir, pw); err != nil {
		if IsBadPassword(err) {
			fmt.Fprintln(os.Stderr, "invalid admin password")
		} else {
			fmt.Fprintln(os.Stderr, "password check:", err)
		}
		os.Exit(1)
	}
}

// stringTrim — local helper so we don't pull strings into yet another file.
func stringTrim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}

// isatty wraps term.IsTerminal so the call site reads nicely.
func isatty(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
