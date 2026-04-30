package services

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// Rule is the OBLIVRA-native rule type. A subset of Sigma's expressiveness:
// every contains-token in any of the AnyOf groups must appear in at least one
// of the configured fields. This is enough for ~80% of common detections and
// keeps the engine boringly fast.
type Rule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Severity   AlertSeverity `json:"severity"`
	Fields     []string      `json:"fields"`               // event fields to match against
	AnyContain []string      `json:"anyContain,omitempty"` // OR — match if any token appears
	AllContain []string      `json:"allContain,omitempty"` // AND — every token must appear
	EventType  string        `json:"eventType,omitempty"`  // optional eventType filter
	MITRE      []string      `json:"mitre,omitempty"`
	Source     string        `json:"source,omitempty"`     // "builtin" | "user" | "sigma"
	Disabled   bool          `json:"disabled,omitempty"`
}

// SigmaLoader is the dependency injection seam for loading external Sigma
// rule bundles. We only define the interface here so the services package
// stays free of YAML/file I/O — internal/sigma plugs in via SetLoader.
type SigmaLoader func(dir string) (rules []Rule, errs []error)

type RulesService struct {
	log    *slog.Logger
	alerts *AlertService

	mu        sync.RWMutex
	rules     []Rule
	matched   map[string]int // ruleID → match count
	heatmap   map[string]int // MITRE technique → count
	lastLoad  time.Time
	sigmaDir  string
	sigmaLoad SigmaLoader
}

func NewRulesService(log *slog.Logger, alerts *AlertService) *RulesService {
	r := &RulesService{
		log:     log,
		alerts:  alerts,
		matched: map[string]int{},
		heatmap: map[string]int{},
	}
	r.rules = builtinRules()
	r.lastLoad = time.Now().UTC()
	return r
}

// AttachSigma points the rules service at a directory of *.yml Sigma rules
// and a loader function. Calling Reload() afterwards picks them up.
func (r *RulesService) AttachSigma(dir string, loader SigmaLoader) {
	r.mu.Lock()
	r.sigmaDir = dir
	r.sigmaLoad = loader
	r.mu.Unlock()
}

func (r *RulesService) ServiceName() string { return "RulesService" }

// Evaluate checks an event against every active rule and raises alerts on
// matches. Called from the ingest fan-out (post WAL/hot/Bleve).
func (r *RulesService) Evaluate(ctx context.Context, ev events.Event) {
	r.mu.RLock()
	rules := r.rules
	r.mu.RUnlock()

	for _, rule := range rules {
		if rule.Disabled {
			continue
		}
		if rule.EventType != "" && rule.EventType != ev.EventType {
			continue
		}
		if !matchRule(rule, ev) {
			continue
		}
		r.mu.Lock()
		r.matched[rule.ID]++
		for _, m := range rule.MITRE {
			r.heatmap[m]++
		}
		r.mu.Unlock()
		alert := AlertFromEvent(ev, rule.ID, rule.Name, rule.Severity, rule.MITRE)
		r.alerts.Raise(ctx, alert)
	}
}

func (r *RulesService) List() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	return out
}

// Reload re-imports built-ins and any attached Sigma directory.
func (r *RulesService) Reload() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := builtinRules()
	if r.sigmaLoad != nil && r.sigmaDir != "" {
		more, errs := r.sigmaLoad(r.sigmaDir)
		for _, e := range errs {
			r.log.Warn("sigma load", "err", e)
		}
		all = append(all, more...)
	}
	r.rules = all
	r.lastLoad = time.Now().UTC()
	return len(r.rules), nil
}

type HeatmapEntry struct {
	Technique string `json:"technique"`
	Count     int    `json:"count"`
}

func (r *RulesService) Heatmap() []HeatmapEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]HeatmapEntry, 0, len(r.heatmap))
	for k, v := range r.heatmap {
		out = append(out, HeatmapEntry{Technique: k, Count: v})
	}
	return out
}

func matchRule(rule Rule, ev events.Event) bool {
	target := buildTarget(rule.Fields, ev)
	target = strings.ToLower(target)

	if len(rule.AllContain) > 0 {
		for _, t := range rule.AllContain {
			if !strings.Contains(target, strings.ToLower(t)) {
				return false
			}
		}
	}
	if len(rule.AnyContain) > 0 {
		hit := false
		for _, t := range rule.AnyContain {
			if strings.Contains(target, strings.ToLower(t)) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

func buildTarget(fields []string, ev events.Event) string {
	if len(fields) == 0 {
		fields = []string{"message", "raw"}
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		switch f {
		case "message":
			parts = append(parts, ev.Message)
		case "raw":
			parts = append(parts, ev.Raw)
		case "hostId":
			parts = append(parts, ev.HostID)
		case "eventType":
			parts = append(parts, ev.EventType)
		case "severity":
			parts = append(parts, string(ev.Severity))
		default:
			if v, ok := ev.Fields[f]; ok {
				parts = append(parts, v)
			}
		}
	}
	return strings.Join(parts, " ")
}

func builtinRules() []Rule {
	mk := func(id, name string, sev AlertSeverity, mitre []string, any []string) Rule {
		return Rule{
			ID: id, Name: name, Severity: sev,
			Fields: []string{"message", "raw"}, AnyContain: any,
			MITRE: mitre, Source: "builtin",
		}
	}
	return []Rule{
		// Linux / SSH
		mk("builtin-ssh-bruteforce", "Possible SSH brute force", AlertSeverityHigh,
			[]string{"T1110.001"}, []string{"failed password", "authentication failure"}),
		mk("builtin-ssh-key-injected", "SSH key injection into authorized_keys", AlertSeverityHigh,
			[]string{"T1098.004"}, []string{"authorized_keys", "ssh-rsa AAAA"}),
		mk("builtin-sudo-failed", "Failed sudo attempt", AlertSeverityMedium,
			[]string{"T1548.003"}, []string{"sudo: pam_unix", "sudo: 3 incorrect"}),
		mk("builtin-cron-tamper", "Cron persistence written", AlertSeverityMedium,
			[]string{"T1053.003"}, []string{"crontab -e", "REPLACE crontab", "/etc/cron.d/"}),
		mk("builtin-ld-preload", "LD_PRELOAD hijack", AlertSeverityHigh,
			[]string{"T1574.006"}, []string{"LD_PRELOAD=", "/etc/ld.so.preload"}),
		mk("builtin-rootkit-load", "Suspicious kernel module load", AlertSeverityCritical,
			[]string{"T1547.006"}, []string{"insmod ", "modprobe ", "init_module"}),

		// Windows — LSASS rule keeps AllContain so plain "comsvcs.dll loaded"
		// (legitimate audit) doesn't fire; we want lsass + a dump-tooling token.
		{
			ID: "builtin-windows-lsass", Name: "LSASS access (possible credential dumping)",
			Severity: AlertSeverityCritical,
			Fields:   []string{"message", "raw"},
			AllContain: []string{"lsass"},
			AnyContain: []string{"lsass.exe", "MiniDump", "comsvcs.dll", "procdump", "rundll32 comsvcs"},
			MITRE:      []string{"T1003.001"},
			Source:     "builtin",
		},
		mk("builtin-powershell-encoded", "PowerShell encoded command", AlertSeverityHigh,
			[]string{"T1059.001"}, []string{"powershell -enc", "powershell.exe -encodedcommand", " -e JAB"}),
		mk("builtin-powershell-downloadstring", "PowerShell DownloadString", AlertSeverityHigh,
			[]string{"T1059.001", "T1105"}, []string{"DownloadString", "Invoke-Expression", "iex (", "(New-Object Net.WebClient)"}),
		mk("builtin-mshta-remote", "Mshta launching remote payload", AlertSeverityHigh,
			[]string{"T1218.005"}, []string{"mshta http", "mshta.exe javascript:"}),
		mk("builtin-rundll32-from-temp", "Rundll32 launched from temp", AlertSeverityHigh,
			[]string{"T1218.011"}, []string{"rundll32 \\appdata\\local\\temp", "rundll32.exe c:\\users\\public"}),
		mk("builtin-bitsadmin-transfer", "BITSAdmin transfer", AlertSeverityMedium,
			[]string{"T1197"}, []string{"bitsadmin /transfer", "bitsadmin.exe /create"}),
		mk("builtin-certutil-decode", "Certutil decode (LOLBin)", AlertSeverityMedium,
			[]string{"T1140"}, []string{"certutil -decode", "certutil.exe -urlcache -split -f"}),
		mk("builtin-wmic-process-create", "WMIC remote process create", AlertSeverityHigh,
			[]string{"T1047"}, []string{"wmic /node:", "wmic.exe process call create"}),
		mk("builtin-psexec-launch", "PsExec lateral movement", AlertSeverityHigh,
			[]string{"T1021.002"}, []string{"psexec.exe \\\\", "psexec -accepteula"}),
		mk("builtin-runkey-registry", "Registry Run key persistence", AlertSeverityMedium,
			[]string{"T1547.001"}, []string{"\\CurrentVersion\\Run", "reg add HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run"}),
		mk("builtin-dcsync", "DCSync access requested", AlertSeverityCritical,
			[]string{"T1003.006"}, []string{"DRSGetNCChanges", "GetNCChanges"}),
		mk("builtin-kerberoast", "Kerberos service-ticket harvest", AlertSeverityHigh,
			[]string{"T1558.003"}, []string{"Audit Failure 4769", "Ticket Encryption Type:0x17"}),

		// Ransomware / impact
		mk("builtin-ransomware-shadow-delete", "Volume shadow copy deletion", AlertSeverityCritical,
			[]string{"T1490"}, []string{"vssadmin delete shadows", "wmic shadowcopy delete", "bcdedit /set bootstatuspolicy"}),
		mk("builtin-ransomware-backup-delete", "Backup catalog wipe", AlertSeverityCritical,
			[]string{"T1490"}, []string{"wbadmin delete catalog", "wbadmin delete backup"}),
		mk("builtin-ransomware-recovery-disable", "Recovery disabled (bcdedit)", AlertSeverityCritical,
			[]string{"T1490"}, []string{"bcdedit /set recoveryenabled no", "bcdedit /set bootstatuspolicy ignoreallfailures"}),

		// Network / firewall
		mk("builtin-firewall-drop", "Firewall dropped traffic", AlertSeverityLow,
			[]string{"T1190"}, []string{"firewalld dropped", "iptables: drop", "kernel: ufw block"}),
		mk("builtin-dns-tunnel-suspect", "Suspect DNS tunneling", AlertSeverityMedium,
			[]string{"T1071.004"}, []string{"TXT IN A", "very long TXT", " query length 200"}),
		mk("builtin-smb-enum", "SMB host enumeration", AlertSeverityMedium,
			[]string{"T1018"}, []string{"net view \\\\", "smbclient -L \\\\"}),
		mk("builtin-rdp-bruteforce", "RDP brute force", AlertSeverityHigh,
			[]string{"T1110.001"}, []string{"event 4625 logon type 10", "TermService logon failed"}),
		mk("builtin-c2-beacon-ua", "Suspicious user-agent (cobalt strike)", AlertSeverityHigh,
			[]string{"T1071.001"}, []string{"User-Agent: Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; Trident/5.0)", "User-Agent: Cobalt"}),

		// Cloud
		mk("builtin-aws-iam-escalate", "AWS IAM privilege escalation", AlertSeverityHigh,
			[]string{"T1098"}, []string{"AttachUserPolicy", "PutUserPolicy", "CreateAccessKey"}),
		mk("builtin-azure-impossible-travel", "Azure impossible travel", AlertSeverityHigh,
			[]string{"T1078.004"}, []string{"signInLogs anomaly impossibleTravel"}),

		// Threat-intel cross-check (catches obvious indicator strings; the
		// async fan-out also handles the structured matcher).
		mk("builtin-ioc-match", "Threat-intel indicator matched in event", AlertSeverityHigh,
			[]string{"T1071"}, []string{"198.51.100.7", "malicious.example.com"}),
	}
}
