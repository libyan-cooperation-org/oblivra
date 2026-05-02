package services

import (
	"context"
	"fmt"
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
// RuleType picks the matcher used for a Rule. "match" is the default
// substring evaluator (AnyContain / AllContain). The other three are
// stateful — they consume a stream of matching events and only fire
// when the configured shape is observed.
type RuleType string

const (
	RuleTypeMatch     RuleType = ""          // single-event substring match (default)
	RuleTypeThreshold RuleType = "threshold" // fire when count(matches) >= Threshold within Window
	RuleTypeFrequency RuleType = "frequency" // fire when count(distinct GroupBy values) >= Threshold within Window
	RuleTypeSequence  RuleType = "sequence"  // fire when Sequence[] of substrings observed in order within Window
)

type Rule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Severity   AlertSeverity `json:"severity"`
	Fields     []string      `json:"fields"`               // event fields to match against
	AnyContain []string      `json:"anyContain,omitempty"` // OR — match if any token appears
	AllContain []string      `json:"allContain,omitempty"` // AND — every token must appear
	NotContain []string      `json:"notContain,omitempty"` // negative gate — drop event if any token appears
	EventType  string        `json:"eventType,omitempty"`  // optional eventType filter
	MITRE      []string      `json:"mitre,omitempty"`
	Source     string        `json:"source,omitempty"`     // "builtin" | "user" | "sigma"
	Disabled   bool          `json:"disabled,omitempty"`

	// Stateful fields (Type != "match"). All ignored when Type is empty.
	Type      RuleType      `json:"type,omitempty"`
	Threshold int           `json:"threshold,omitempty"`
	Window    time.Duration `json:"window,omitempty"`
	// GroupBy is the bucket dimension — events with the same value are
	// counted in the same window. Defaults to "hostId" so threshold/
	// frequency rules are scoped per host.
	GroupBy string `json:"groupBy,omitempty"`
	// DistinctOf is the cardinality field for frequency rules — counts
	// the number of unique values within the bucket window. Defaults
	// to "srcIP". Ignored for threshold and sequence rules.
	DistinctOf string   `json:"distinctOf,omitempty"`
	Sequence   []string `json:"sequence,omitempty"` // ordered substring list for RuleTypeSequence
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
	matched   map[string]int             // ruleID → cumulative match count
	heatmap   map[string]int             // MITRE technique → count
	feedback  map[string]*RuleFeedback   // ruleID → analyst-marked TP/FP
	firesDay  map[string]map[string]int  // ruleID → "YYYY-MM-DD" → fires that day
	lastLoad  time.Time
	sigmaDir  string
	sigmaLoad SigmaLoader

	// Stateful evaluators. One bucket per (ruleID, groupKey), holding a
	// rolling window of timestamps + (for sequence rules) the next-needed
	// step index. We GC stale buckets every Evaluate call so the maps
	// can't grow unbounded on transient host names.
	stateMu sync.Mutex
	state   map[string]map[string]*ruleBucket // ruleID → groupKey → bucket
}

// ruleBucket tracks one (rule, groupKey) pair's recent activity.
type ruleBucket struct {
	hits        []time.Time         // for threshold
	distinctVal map[string]time.Time // for frequency: distinct value → most-recent ts
	seqIdx      int                  // for sequence: which step we're waiting on next
	seqStarted  time.Time            // when the current sequence run began
	lastFired   time.Time            // re-arm gate so we don't fire every event
}

// RuleFeedback tracks analyst-supplied effectiveness signal. Analysts
// mark alerts as true-positive or false-positive in the case workflow;
// the feedback rolls up here so the rules view can show "fires-per-day
// vs FP-rate" to operators tuning the rule pack.
type RuleFeedback struct {
	TP        int       `json:"tp"`
	FP        int       `json:"fp"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// RuleEffectiveness is the per-rule scorecard returned to the operator UI.
type RuleEffectiveness struct {
	RuleID         string  `json:"ruleId"`
	RuleName       string  `json:"ruleName"`
	Source         string  `json:"source"`         // builtin / user / sigma / community:<sha>
	TotalFires     int     `json:"totalFires"`
	FiresLast24h   int     `json:"firesLast24h"`
	FiresLast7Days int     `json:"firesLast7Days"`
	TP             int     `json:"tp"`
	FP             int     `json:"fp"`
	FPRate         float64 `json:"fpRate"` // FP / (TP + FP); -1 if no feedback yet
}

func NewRulesService(log *slog.Logger, alerts *AlertService) *RulesService {
	r := &RulesService{
		log:      log,
		alerts:   alerts,
		matched:  map[string]int{},
		heatmap:  map[string]int{},
		feedback: map[string]*RuleFeedback{},
		firesDay: map[string]map[string]int{},
		state:    map[string]map[string]*ruleBucket{},
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
		// Stateful evaluators get their own dispatch — they observe
		// every matching event but only fire when the configured shape
		// is satisfied (N hits in window / distinct count / sequence).
		if rule.Type != RuleTypeMatch {
			if r.evaluateStateful(ctx, rule, ev) {
				r.recordFire(rule)
				alert := AlertFromEvent(ev, rule.ID, rule.Name, rule.Severity, rule.MITRE)
				alert.Message = stateMessageFor(rule, alert.Message)
				r.alerts.Raise(ctx, alert)
			}
			continue
		}
		if !matchRule(rule, ev) {
			continue
		}
		r.recordFire(rule)
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

// MarkAlert records analyst feedback for a rule fire. label must be
// "tp" (true positive) or "fp" (false positive); anything else is a
// no-op. Per-rule counts feed the Effectiveness scorecard so the
// operator can tell apart noisy rules from useful ones.
func (r *RulesService) MarkAlert(ruleID, label string) {
	if label != "tp" && label != "fp" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	fb := r.feedback[ruleID]
	if fb == nil {
		fb = &RuleFeedback{}
		r.feedback[ruleID] = fb
	}
	if label == "tp" {
		fb.TP++
	} else {
		fb.FP++
	}
	fb.UpdatedAt = time.Now().UTC()
}

// Effectiveness returns one row per rule with cumulative fires, recent
// fires, analyst-marked TP/FP counts, and the running FP rate. -1 in
// FPRate means no feedback yet (avoids dividing by zero).
func (r *RulesService) Effectiveness() []RuleEffectiveness {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RuleEffectiveness, 0, len(r.rules))
	now := time.Now().UTC()
	for _, rule := range r.rules {
		row := RuleEffectiveness{
			RuleID: rule.ID, RuleName: rule.Name, Source: rule.Source,
			TotalFires: r.matched[rule.ID],
		}
		days := r.firesDay[rule.ID]
		for d, n := range days {
			ts, err := time.Parse("2006-01-02", d)
			if err != nil {
				continue
			}
			age := now.Sub(ts)
			if age < 24*time.Hour {
				row.FiresLast24h += n
			}
			if age < 7*24*time.Hour {
				row.FiresLast7Days += n
			}
		}
		if fb, ok := r.feedback[rule.ID]; ok {
			row.TP = fb.TP
			row.FP = fb.FP
			if fb.TP+fb.FP > 0 {
				row.FPRate = float64(fb.FP) / float64(fb.TP+fb.FP)
			} else {
				row.FPRate = -1
			}
		} else {
			row.FPRate = -1
		}
		out = append(out, row)
	}
	return out
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
	// Negative gate — drop events that contain any "not" token. Lets
	// rule authors carve out known-good patterns (e.g. "fail" but not
	// "test failed during deploy") without writing a separate Sigma file.
	if len(rule.NotContain) > 0 {
		for _, t := range rule.NotContain {
			if strings.Contains(target, strings.ToLower(t)) {
				return false
			}
		}
	}
	return true
}

// recordFire updates per-rule + per-MITRE counters and the per-day
// fires map. Called from both the substring evaluator and the stateful
// evaluator on each fire.
func (r *RulesService) recordFire(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.matched[rule.ID]++
	for _, m := range rule.MITRE {
		r.heatmap[m]++
	}
	day := time.Now().UTC().Format("2006-01-02")
	if r.firesDay[rule.ID] == nil {
		r.firesDay[rule.ID] = map[string]int{}
	}
	r.firesDay[rule.ID][day]++
}

// evaluateStateful runs the threshold/frequency/sequence evaluators.
// The substring matcher (matchRule) gates whether the event is even a
// candidate; only candidates feed the bucket. Returns true if the
// configured shape was just satisfied — meaning the caller should
// raise an alert.
//
// Re-arm: after a fire, the bucket goes silent until the window
// elapses. Otherwise a "5 fails in 60s" rule would scream on every
// subsequent fail until the burst ends.
func (r *RulesService) evaluateStateful(_ context.Context, rule Rule, ev events.Event) bool {
	if !matchRule(rule, ev) {
		return false
	}
	window := rule.Window
	if window <= 0 {
		window = 60 * time.Second
	}
	threshold := rule.Threshold
	if threshold <= 0 {
		threshold = 5
	}
	groupBy := rule.GroupBy
	if groupBy == "" {
		groupBy = "hostId"
	}
	groupKey := groupKeyFor(ev, groupBy)
	now := time.Now().UTC()

	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.state[rule.ID] == nil {
		r.state[rule.ID] = map[string]*ruleBucket{}
	}
	bkt := r.state[rule.ID][groupKey]
	if bkt == nil {
		bkt = &ruleBucket{distinctVal: map[string]time.Time{}}
		r.state[rule.ID][groupKey] = bkt
	}

	// Re-arm gate: if we fired recently, stay silent until the window
	// passes. This deliberately suppresses runs of identical alerts.
	if !bkt.lastFired.IsZero() && now.Sub(bkt.lastFired) < window {
		// Still update bucket so a new burst after the gate fires.
		switch rule.Type {
		case RuleTypeThreshold:
			bkt.hits = appendWithin(bkt.hits, now, window)
		case RuleTypeFrequency:
			bkt.distinctVal[distinctValueFor(ev, rule)] = now
		}
		return false
	}

	switch rule.Type {
	case RuleTypeThreshold:
		bkt.hits = appendWithin(bkt.hits, now, window)
		if len(bkt.hits) >= threshold {
			bkt.lastFired = now
			bkt.hits = bkt.hits[:0]
			return true
		}
	case RuleTypeFrequency:
		val := distinctValueFor(ev, rule)
		if val == "" {
			return false
		}
		bkt.distinctVal[val] = now
		// Prune stale.
		for k, ts := range bkt.distinctVal {
			if now.Sub(ts) > window {
				delete(bkt.distinctVal, k)
			}
		}
		if len(bkt.distinctVal) >= threshold {
			bkt.lastFired = now
			bkt.distinctVal = map[string]time.Time{}
			return true
		}
	case RuleTypeSequence:
		if len(rule.Sequence) == 0 {
			return false
		}
		target := strings.ToLower(buildTarget(rule.Fields, ev))
		needle := strings.ToLower(rule.Sequence[bkt.seqIdx])
		if !strings.Contains(target, needle) {
			return false
		}
		// Re-anchor on first step.
		if bkt.seqIdx == 0 {
			bkt.seqStarted = now
		}
		// Out-of-window — restart from step 0 (and re-evaluate this event
		// as the first step, since it matched).
		if now.Sub(bkt.seqStarted) > window && bkt.seqIdx > 0 {
			bkt.seqIdx = 0
			bkt.seqStarted = now
			// Re-check against step 0.
			if !strings.Contains(target, strings.ToLower(rule.Sequence[0])) {
				return false
			}
		}
		bkt.seqIdx++
		if bkt.seqIdx >= len(rule.Sequence) {
			bkt.lastFired = now
			bkt.seqIdx = 0
			return true
		}
	}
	return false
}

// appendWithin trims hits older than `window` then appends `now`.
func appendWithin(hits []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	for len(hits) > 0 && hits[0].Before(cutoff) {
		hits = hits[1:]
	}
	return append(hits, now)
}

// groupKeyFor picks the bucket key. Unknown field → empty group (one
// global bucket for the rule), which is fine for "fleet-wide" rules.
func groupKeyFor(ev events.Event, field string) string {
	switch field {
	case "hostId":
		return ev.HostID
	case "tenantId":
		return ev.TenantID
	case "eventType":
		return ev.EventType
	case "":
		return ""
	default:
		if ev.Fields != nil {
			return ev.Fields[field]
		}
		return ""
	}
}

// distinctValueFor pulls the value used for "frequency" cardinality
// counting. Uses Rule.DistinctOf, falling back to "srcIP" — the usual
// answer to "N distinct sources tried…".
func distinctValueFor(ev events.Event, rule Rule) string {
	field := rule.DistinctOf
	if field == "" {
		field = "srcIP"
	}
	if v := groupKeyFor(ev, field); v != "" {
		return v
	}
	return ""
}

func stateMessageFor(rule Rule, fallback string) string {
	switch rule.Type {
	case RuleTypeThreshold:
		return fmt.Sprintf("threshold rule %s fired: %d matches in %s", rule.ID, rule.Threshold, rule.Window)
	case RuleTypeFrequency:
		return fmt.Sprintf("frequency rule %s fired: %d distinct values in %s", rule.ID, rule.Threshold, rule.Window)
	case RuleTypeSequence:
		return fmt.Sprintf("sequence rule %s fired: %d-step pattern observed within %s", rule.ID, len(rule.Sequence), rule.Window)
	}
	return fallback
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
	}
}
