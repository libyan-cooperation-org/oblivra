package main

import (
	"os"

	"golang.org/x/term"
)

// readSecretNoEcho reads a single line from stdin with terminal echo
// disabled. Returns (value, true) when it could turn off echo;
// (zero, false) when the platform / non-TTY case forces a fallback.
func readSecretNoEcho() (string, bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", false
	}
	body, err := term.ReadPassword(fd)
	if err != nil {
		return "", false
	}
	return string(body), true
}
