package parsers

import (
	"regexp"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// LinuxAuthParser parses standard /var/log/auth.log lines (sshd, sudo)
type LinuxAuthParser struct {
	// Regex for standard syslog header + sshd failed login
	// e.g., "sshd[1234]: Failed password for invalid user admin from 10.0.0.1 port 22 ssh2"
	sshdFailedRegex *regexp.Regexp
	// Regex for sshd successful login
	// e.g., "sshd[1234]: Accepted publickey for root from 192.168.1.10 port 54322 ssh2"
	sshdSuccessRegex *regexp.Regexp
	// Sudo execution
	// e.g., "sudo:  username : TTY=pts/0 ; PWD=/home/user ; USER=root ; COMMAND=/bin/bash"
	sudoRegex *regexp.Regexp
}

func NewLinuxAuthParser() *LinuxAuthParser {
	return &LinuxAuthParser{
		sshdFailedRegex:  regexp.MustCompile(`sshd\[\d+\]: Failed \w+ for (invalid user )?(\w+) from ([\d\.]+)`),
		sshdSuccessRegex: regexp.MustCompile(`sshd\[\d+\]: Accepted \w+ for (\w+) from ([\d\.]+)`),
		// ReDoS Fix: Bounds backtracking to the specific execution parameter group `[^;]*`
		sudoRegex: regexp.MustCompile(`sudo:\s+(\w+)\s+:[^;]*USER=root\s+;\s+COMMAND=(.*)`),
	}
}

func (p *LinuxAuthParser) Name() string {
	return "LinuxAuth"
}

func (p *LinuxAuthParser) CanParse(line string) bool {
	return strings.Contains(line, "sshd[") || strings.Contains(line, "sudo:")
}

func (p *LinuxAuthParser) Parse(info Info, event *database.HostEvent) error {
	line := info.RawLine

	if match := p.sshdFailedRegex.FindStringSubmatch(line); match != nil {
		event.EventType = "failed_login"
		event.User = match[2]
		event.SourceIP = match[3]
		return nil
	}

	if match := p.sshdSuccessRegex.FindStringSubmatch(line); match != nil {
		event.EventType = "successful_login"
		event.User = match[1]
		event.SourceIP = match[2]
		return nil
	}

	if match := p.sudoRegex.FindStringSubmatch(line); match != nil {
		event.EventType = "sudo_exec"
		event.User = match[1]
		// Source IP might be blank if it was a local TTY, rely on origin IP
		return nil
	}

	event.EventType = "linux_auth_event"
	return nil
}
