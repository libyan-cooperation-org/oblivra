package security

import (
	"regexp"
	"strings"
)

// ShellSanitizer provides utilities for cleaning user-provided strings before
// they are passed to shell executors.
type ShellSanitizer struct {
	dangerousPatterns []*regexp.Regexp
}

func NewShellSanitizer() *ShellSanitizer {
	return &ShellSanitizer{
		dangerousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`[;&|><$(){}\x60]`), // Shell metacharacters
		},
	}
}

// Sanitize removes dangerous characters from a string.
func (s *ShellSanitizer) Sanitize(input string) string {
	// Simple replacement for basic safety
	output := input
	for _, re := range s.dangerousPatterns {
		output = re.ReplaceAllString(output, "")
	}
	return strings.TrimSpace(output)
}

// IsSafe returns true if the command doesn't contain obvious injection vectors.
func (s *ShellSanitizer) IsSafe(cmd string) bool {
	// 1. Check for directory traversal or destructive root commands
	dangerous := []string{
		"rm -rf /",
		"mv /*",
		":(){:|:&};:", // Fork bomb
		"> /dev/",
	}

	lowerCmd := strings.ToLower(cmd)
	for _, d := range dangerous {
		if strings.Contains(lowerCmd, d) {
			return false
		}
	}

	return true
}
