package security

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
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
			regexp.MustCompile(`\$\{IFS\}`),                                // IFS bypass
			regexp.MustCompile(`\$\(.*\)`),                                 // command substitution $(cmd)
			regexp.MustCompile(`\x60.*\x60`),                               // command substitution `cmd`
			regexp.MustCompile(`\\`),                                       // escaping bypass

			// Destructive filesystem operations on root or all files
			regexp.MustCompile(`(?i)\brm\s+(-[rRfF]+\s+){0,3}/?\*`),   // rm -rf *, rm -rf /*
			regexp.MustCompile(`(?i)\brm\s+(-[rRfF]+\s+){0,3}/\s*`),   // rm -rf /
			regexp.MustCompile(`(?i)\bmkfs\b`),                          // filesystem creation
			regexp.MustCompile(`(?i)\bdd\s+if=`),                        // disk imaging/overwrite
			regexp.MustCompile(`(?i)\b(reboot|shutdown|halt|poweroff)\b`), // remote power commands
			regexp.MustCompile(`(?i)\biptables\s+-F\b`),                  // flush all firewall rules
			regexp.MustCompile(`:\s*\(\s*\)\s*\{`),                      // fork bomb pattern :(){:|:&};
			regexp.MustCompile(`(?i)>\s*/dev/(sd|nvme|hd|vd)`),           // direct disk overwrite

			// Additional dangerous commands (CS-03 expansion)
			regexp.MustCompile(`(?i)\bpasswd\b`),          // prevent manual password changes via Exec
			regexp.MustCompile(`(?i)\bchown\b`),           // prevent ownership takeover
			regexp.MustCompile(`(?i)\bchmod\s+0?777`),     // prevent insecure permissions
			regexp.MustCompile(`(?i)\buser(add|del|mod)\b`), // prevent user manipulation
			regexp.MustCompile(`(?i)\bgroup(add|del|mod)\b`), // prevent group manipulation
			regexp.MustCompile(`(?i)\bsu\b`),              // prevent user switching
			regexp.MustCompile(`(?i)\bsudo\b`),            // prevent privilege escalation
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

// normaliseCmd applies NFKD Unicode normalisation and collapses all Unicode
// whitespace to plain ASCII spaces. Used by both IsSafe and IsAllowlisted.
func normaliseCmd(cmd string) string {
	b := norm.NFKD.Bytes([]byte(cmd))
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, string(b))
}

// IsSafe returns true if the command passes all safety checks.
// Uses Unicode normalisation + denylist matching to prevent bypass via case
// variants, whitespace padding, or Unicode lookalikes (CS-03).
//
// SA-05: Pure denylist approaches are inherently incomplete. IsSafe remains the
// default guard; callers that operate in a known-safe command space should prefer
// IsAllowlisted for stricter "known-good only" enforcement.
func (s *ShellSanitizer) IsSafe(cmd string) bool {
	if cmd == "" {
		return true
	}
	normalised := normaliseCmd(cmd)
	for _, re := range s.destructivePatterns {
		if re.MatchString(normalised) {
			return false
		}
	}
	return true
}

// safeCommandAllowlist contains patterns for commands that are broadly safe for
// remote execution. SA-05: used by IsAllowlisted to provide a stricter guard.
var safeCommandAllowlist = []*regexp.Regexp{
	regexp.MustCompile(`^(ls|ll|la)(\s+-[a-zA-Z]+)*(\s+[\w./~-]+)*$`),            // directory listing
	regexp.MustCompile(`^cat\s+[\w./~-]+$`),                                        // read single file
	regexp.MustCompile(`^echo\s+[\w .@:/_-]*$`),                                    // safe echo (no $())
	regexp.MustCompile(`^(pwd|whoami|hostname|uptime|date|uname(\s+-[a-zA-Z]+)*)$`), // info commands
	regexp.MustCompile(`^ps(\s+(aux|axu|-ef|-e))?$`),                               // process listing
	regexp.MustCompile(`^df(\s+-[a-zA-Z]*)?(\s+[\w./~-]+)?$`),                      // disk usage
	regexp.MustCompile(`^free(\s+-[a-zA-Z]+)?$`),                                   // memory info
	regexp.MustCompile(`^(top|htop)$`),                                              // process monitor
	regexp.MustCompile(`^(systemctl|service)\s+(status)\s+[\w.-]+$`),               // service status only
	regexp.MustCompile(`^tail(\s+-[nf]\s*\d*)?(\s+[\w./~-]+)+$`),                  // tail log files
	regexp.MustCompile(`^grep(\s+-[a-zA-Z]+)*\s+\w+\s+[\w./~-]+$`),                // basic grep
	regexp.MustCompile(`^(netstat|ss)(\s+-[a-zA-Z]+)*$`),                           // network status
	regexp.MustCompile(`^ip\s+(addr|link|route)(\s+show)?$`),                       // ip info
}

// IsAllowlisted returns true only if the command matches one of the explicitly safe
// allowlist patterns. This is stricter than IsSafe and should be preferred in
// automated or high-risk contexts where only well-known commands are expected (SA-05).
func (s *ShellSanitizer) IsAllowlisted(cmd string) bool {
	if cmd == "" {
		return false
	}
	normalised := strings.TrimSpace(normaliseCmd(cmd))
	for _, re := range safeCommandAllowlist {
		if re.MatchString(normalised) {
			return true
		}
	}
	return false
}

// SanitizeLogLine strips CRLF and control characters from a string to prevent log injection (CS-09).
func (s *ShellSanitizer) SanitizeLogLine(input string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, input)
}

// SanitizeCSV prevents CSV formula injection by prefixing dangerous characters (CS-12).
func (s *ShellSanitizer) SanitizeCSV(input string) string {
	if input == "" {
		return ""
	}
	// Dangerous characters: =, +, -, @
	first := input[0]
	if first == '=' || first == '+' || first == '-' || first == '@' {
		return "'" + input
	}
	return input
}
