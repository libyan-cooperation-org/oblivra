package main

import (
	"regexp"
	"strings"
)

// LocalRule is a tiny subset of the platform's Sigma engine. The agent
// runs these locally so high-priority events (LSASS dump attempts,
// auditd disable, ransomware shadow-delete) jump the queue when the
// outbound link is throttled or down.
//
// Splunk UF ships everything FIFO. We tier — critical events first.
type LocalRule struct {
	ID        string
	Severity  int    // 1=info → 4=critical
	AnyOf     []string
	Compiled  *regexp.Regexp // optional; if set, AnyOf is ignored
}

// DefaultLocalRules is what every agent ships with. Kept tiny on purpose
// — the platform has the full rule engine; this is just the
// "wake the operator NOW" subset.
func DefaultLocalRules() []LocalRule {
	return []LocalRule{
		{ID: "tamper-auditd", Severity: 4, AnyOf: []string{
			"auditctl -D", "auditctl --delete", "auditd stopped",
		}},
		{ID: "tamper-eventlog", Severity: 4, AnyOf: []string{
			"wevtutil cl", "Clear-EventLog", "fsutil usn deletejournal",
		}},
		{ID: "lsass-dump", Severity: 4, AnyOf: []string{
			"lsass.exe", "MiniDump", "comsvcs.dll", "procdump",
		}},
		{ID: "ransomware-shadow-delete", Severity: 4, AnyOf: []string{
			"vssadmin delete shadows", "wmic shadowcopy delete",
			"wbadmin delete catalog",
		}},
		{ID: "encoded-powershell", Severity: 3, AnyOf: []string{
			"powershell -enc", "powershell.exe -encodedcommand",
		}},
		{ID: "ssh-bruteforce", Severity: 3, AnyOf: []string{
			"failed password", "authentication failure",
		}},
		{ID: "tamper-history", Severity: 3, AnyOf: []string{
			"unset histfile", "histfile=/dev/null", "history -c",
			"> ~/.bash_history", "> /root/.bash_history",
			"rm -f ~/.bash_history", "rm -f /root/.bash_history",
		}},
		{ID: "tamper-defender", Severity: 4, AnyOf: []string{
			"set-mppreference", "disablerealtimemonitoring",
			"disablebehaviormonitoring",
		}},
		{ID: "tamper-agent", Severity: 4, AnyOf: []string{
			"systemctl stop oblivra-agent", "systemctl mask oblivra-agent",
			"sc.exe stop oblivraagent", "sc.exe delete oblivraagent",
			"taskkill /im oblivra-agent.exe", "stop-service oblivraagent",
		}},
		{ID: "timestomp", Severity: 2, AnyOf: []string{
			"touch -t ", "touch -r ", "touch --date",
			".creationtime=", ".lastwritetime=", ".lastaccesstime=",
		}},
		{ID: "windows-eventlog-cleared", Severity: 4, AnyOf: []string{
			"event id 1102", "event id 104", "the audit log was cleared",
		}},
	}
}

// ScoreLine returns the highest severity any local rule fires on this
// raw event line. Zero means no match. Cheap substring scan — designed
// to run inside the tailer without measurably slowing it down.
func ScoreLine(line string, rules []LocalRule) int {
	if line == "" {
		return 0
	}
	low := strings.ToLower(line)
	best := 0
	for _, r := range rules {
		if r.Compiled != nil {
			if r.Compiled.MatchString(line) && r.Severity > best {
				best = r.Severity
			}
			continue
		}
		for _, needle := range r.AnyOf {
			if strings.Contains(low, strings.ToLower(needle)) {
				if r.Severity > best {
					best = r.Severity
				}
				break
			}
		}
	}
	return best
}
