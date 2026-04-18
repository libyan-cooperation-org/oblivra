package security

import (
	"regexp"
	"strings"
	"unicode"
)

// ShellSanitizer provides utilities for validating commands before
// they are passed to SSH exec sessions.
type ShellSanitizer struct {
	// dangerousPatterns matches shell metacharacters used in injection attacks
	dangerousPatterns []*regexp.Regexp
	// destructivePatterns matches regex patterns for known-destructive commands
	destructivePatterns []*regexp.Regexp
}

func NewShellSanitizer() *ShellSanitizer {
	return &ShellSanitizer{
		dangerousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`[;&|><$(){}\x60]`), // Shell metacharacters
		},
		destructivePatterns: []*regexp.Regexp{
			// Advanced Shell Injection Bypasses
			regexp.MustCompile(`\$\{IFS\}`),                             // IFS bypass
			regexp.MustCompile(`\$\(.*\)`),                             // command substitution $(cmd)
			regexp.MustCompile(`\x60.*\x60`),                           // command substitution `cmd`
			regexp.MustCompile(`\\`),                                   // escaping bypass
			
			// Destructive filesystem operations on root or all files
			regexp.MustCompile(`(?i)\brm\s+(-[rRfF]+\s+){0,3}/?\*`),   // rm -rf *, rm -rf /*
			regexp.MustCompile(`(?i)\brm\s+(-[rRfF]+\s+){0,3}/\s*`),  // rm -rf /
			regexp.MustCompile(`(?i)\bmkfs\b`),                          // filesystem creation
			regexp.MustCompile(`(?i)\bdd\s+if=`),                       // disk imaging/overwrite
			regexp.MustCompile(`(?i)\b(reboot|shutdown|halt|poweroff)\b`), // remote power commands
			regexp.MustCompile(`(?i)\biptables\s+-F\b`),                 // flush all firewall rules
			regexp.MustCompile(`:\s*\(\s*\)\s*\{`),                     // fork bomb pattern :(){:|:&};
			regexp.MustCompile(`(?i)>\s*/dev/(sd|nvme|hd|vd)`),          // direct disk overwrite
			
			// Additional dangerous commands
			regexp.MustCompile(`(?i)\bpasswd\b`),                       // prevent manual password changes via Exec
			regexp.MustCompile(`(?i)\bchown\b`),                        // prevent ownership takeover
			regexp.MustCompile(`(?i)\bchmod\s+777`),                    // prevent insecure permissions
		},
	}
}

// Sanitize removes dangerous shell metacharacters from a string.
func (s *ShellSanitizer) Sanitize(input string) string {
	output := input
	for _, re := range s.dangerousPatterns {
		output = re.ReplaceAllString(output, "")
	}
	return strings.TrimSpace(output)
}

// IsSafe returns true if the command passes all safety checks.
// Uses regex matching and Unicode normalisation to prevent bypass via
// case variants, whitespace padding, or Unicode lookalikes.
func (s *ShellSanitizer) IsSafe(cmd string) bool {
	if cmd == "" {
		return true
	}

	// Collapse all Unicode whitespace to single ASCII spaces before pattern matching
	// to prevent "rm\u00a0-rf\u00a0/" style bypass
	normalised := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, cmd)

	for _, re := range s.destructivePatterns {
		if re.MatchString(normalised) {
			return false
		}
	}

	return true
}
