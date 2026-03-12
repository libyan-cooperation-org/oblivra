package simulation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Scenario represents a predefined attack pattern
type Scenario struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TargetType  string   `json:"target_type"` // e.g., "host", "user", "file"
	MitreID     string   `json:"mitre_id"`
	Tactics     []string `json:"tactics"`
}

// SimulationResult tracks the outcome of a simulation
type SimulationResult struct {
	ScenarioID string    `json:"scenario_id"`
	Target     string    `json:"target"`
	Timestamp  string    `json:"timestamp"`
	Detected   bool      `json:"detected"`
	AlertID    string    `json:"alert_id,omitempty"`
}

// SimulationService manages threat simulations
type SimulationService struct {
	bus *eventbus.Bus
	log *logger.Logger
	ctx context.Context

	mu                *sync.Mutex
	activeSimulations map[string]*SimulationResult // scenarioID+target -> result
	campaigns         *CampaignManager
	validationHistory []ValidationRun
}

// NewSimulationService creates a new simulation service
func NewSimulationService(bus *eventbus.Bus, log *logger.Logger) *SimulationService {
	return &SimulationService{
		bus:               bus,
		log:               log.WithPrefix("simulation"),
		activeSimulations: make(map[string]*SimulationResult),
		campaigns:         NewCampaignManager(),
		validationHistory: make([]ValidationRun, 0),
		mu:                &sync.Mutex{},
	}
}

func (s *SimulationService) Name() string {
	return "SimulationService"
}

func (s *SimulationService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("Threat Simulation Service starting...")

	// Listen for alerts to verify if simulations are detected
	s.bus.Subscribe("siem.alert_fired", func(event eventbus.Event) {
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return
		}

		// Only care about alerts marked as simulation or where we can correlate target
		isSim := data["simulation"] == true
		if !isSim {
			return
		}

		// Simple correlation: check if any active simulation matches the type/target
		// In a real system, we'd use a unique simulation correlation ID
		for key, result := range s.activeSimulations {
			if strings.Contains(fmt.Sprintf("%v", data["description"]), result.ScenarioID) ||
				data["source_ip"] == result.Target || data["host_id"] == result.Target {
				result.Detected = true
				result.AlertID = fmt.Sprintf("%v", data["id"])
				s.log.Info("✅ Simulation Detected: %s on %s", result.ScenarioID, result.Target)
				delete(s.activeSimulations, key)
			}
		}
	})
}

func (s *SimulationService) Shutdown() {
	s.log.Info("Threat Simulation Service shutting down...")
}

// ListScenarios returns available attack patterns
func (s *SimulationService) ListScenarios() []Scenario {
	return []Scenario{
		{
			ID:          "brute_force_auth",
			Name:        "SSH Brute Force",
			Description: "Simulates multiple failed SSH login attempts from a single IP.",
			TargetType:  "host",
			MitreID:     "T1110.001",
			Tactics:     []string{"Credential Access"},
		},
		{
			ID:          "ransomware_entropy",
			Name:        "Ransomware Entropy Spike",
			Description: "Simulates a rapid burst of high-entropy file writes (FIM events).",
			TargetType:  "host",
			MitreID:     "T1486",
			Tactics:     []string{"Impact"},
		},
		{
			ID:          "canary_tamper",
			Name:        "Canary File Tampering",
			Description: "Simulates modification of a protected canary file.",
			TargetType:  "file",
			MitreID:     "T1070",
			Tactics:     []string{"Defense Evasion"},
		},
		{
			ID:          "dns_tunneling",
			Name:        "DNS Tunneling (DGA)",
			Description: "Simulates high-entropy, random DNS queries to mimic C2 tunneling.",
			TargetType:  "host",
			MitreID:     "T1071.004",
			Tactics:     []string{"Command and Control"},
		},
		{
			ID:          "lateral_port_sweep",
			Name:        "Lateral Port Sweep",
			Description: "Simulates internal reconnaissance across multiple hosts.",
			TargetType:  "host",
			MitreID:     "T1046",
			Tactics:     []string{"Discovery"},
		},
	}
}

// RunScenario triggers a simulation
func (s *SimulationService) RunScenario(id string, target string) error {
	s.log.Info("Running simulation scenario: %s on %s", id, target)

	// Register active simulation for detection feedback
	s.activeSimulations[id+target] = &SimulationResult{
		ScenarioID: id,
		Target:     target,
		Timestamp:  time.Now().Format(time.RFC3339),
		Detected:   false,
	}

	switch id {
	case "brute_force_auth":
		return s.simulateBruteForce(target)
	case "ransomware_entropy":
		return s.simulateRansomware(target)
	case "canary_tamper":
		return s.simulateCanaryTamper(target)
	case "dns_tunneling":
		return s.simulateDNSTunneling(target)
	case "lateral_port_sweep":
		return s.simulateLateralPortSweep(target)
	default:
		return fmt.Errorf("unknown scenario: %s", id)
	}
}

func (s *SimulationService) simulateBruteForce(hostID string) error {
	// Simulate 5 failed login attempts
	for i := 0; i < 5; i++ {
		s.bus.Publish(eventbus.EventSSHLoginFailed, map[string]interface{}{
			"host_id":    hostID,
			"source_ip":  "192.168.1.50",
			"user":       "admin",
			"timestamp":  time.Now().Format(time.RFC3339),
			"simulation": true,
		})
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (s *SimulationService) simulateRansomware(hostID string) error {
	// Simulate high-entropy writes across multiple files
	for i := 0; i < 10; i++ {
		filePath := fmt.Sprintf("/data/documents/file_%d.docx", i)
		s.bus.Publish(eventbus.EventFIMModified, map[string]interface{}{
			"host_id":    hostID,
			"path":       filePath,
			"entropy":    7.8, // High entropy
			"size":       1024 * 1024,
			"timestamp":  time.Now().Format(time.RFC3339),
			"simulation": true,
		})
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (s *SimulationService) simulateCanaryTamper(filePath string) error {
	s.bus.Publish(eventbus.EventFIMModified, map[string]interface{}{
		"host_id":    "local-node",
		"path":       filePath,
		"user":       "intruder",
		"timestamp":  time.Now().Format(time.RFC3339),
		"is_canary":  true,
		"simulation": true,
	})
	return nil
}
func (s *SimulationService) simulateDNSTunneling(hostID string) error {
	// Simulate 20 high-entropy DNS queries
	for i := 0; i < 20; i++ {
		query := fmt.Sprintf("%x.c2-command-node.sovereign.local", time.Now().UnixNano())
		s.bus.Publish("ndr.dns_query", map[string]string{
			"host_id":    hostID,
			"query":      query,
			"answer":     "127.0.0.1",
			"simulation": "true",
		})
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (s *SimulationService) simulateLateralPortSweep(sourceIP string) error {
	// Simulate contacting 15 unique internal hosts (threshold is 10)
	for i := 1; i <= 15; i++ {
		destIP := fmt.Sprintf("192.168.1.%d", i+20)
		s.bus.Publish("ndr.flow_captured", map[string]interface{}{
			"Timestamp":  time.Now().Format(time.RFC3339),
			"SourceIP":   sourceIP,
			"SourcePort": 54321,
			"DestIP":     destIP,
			"DestPort":   445, // SMB
			"Protocol":   "TCP",
			"simulation": true,
		})
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (s *SimulationService) GetResults() []*SimulationResult {
	results := make([]*SimulationResult, 0, len(s.activeSimulations))
	for _, r := range s.activeSimulations {
		results = append(results, r)
	}
	return results
}

func (s *SimulationService) StartCampaign(name string, scenarios []string) string {
	id := fmt.Sprintf("camp_%d", time.Now().Unix())
	s.campaigns.StartCampaign(id, name, scenarios)
	return id
}

func (s *SimulationService) GetCampaigns() []*Campaign {
	return s.campaigns.ListCampaigns()
}

// ─── Purple Team: Coverage & Validation ───────────────────────────────────────

// mitreMatrix is the full MITRE ATT&CK technique set we measure coverage against.
// Each entry maps a tactic to its child techniques.
var mitreMatrix = map[string][]struct{ ID, Name string }{
	"Initial Access":       {{"T1078", "Valid Accounts"}, {"T1190", "Exploit Public-Facing App"}, {"T1566", "Phishing"}},
	"Execution":            {{"T1059", "Command and Scripting Interpreter"}, {"T1053", "Scheduled Task/Job"}},
	"Persistence":          {{"T1098", "Account Manipulation"}, {"T1136", "Create Account"}, {"T1543", "Create or Modify System Process"}},
	"Privilege Escalation": {{"T1548", "Abuse Elevation Control"}, {"T1068", "Exploitation for Privilege Escalation"}},
	"Defense Evasion":      {{"T1070", "Indicator Removal"}, {"T1562", "Impair Defenses"}, {"T1036", "Masquerading"}},
	"Credential Access":    {{"T1110", "Brute Force"}, {"T1003", "OS Credential Dumping"}, {"T1552", "Unsecured Credentials"}},
	"Discovery":            {{"T1046", "Network Service Discovery"}, {"T1082", "System Information Discovery"}, {"T1018", "Remote System Discovery"}},
	"Lateral Movement":     {{"T1021", "Remote Services"}, {"T1570", "Lateral Tool Transfer"}},
	"Collection":           {{"T1005", "Data from Local System"}, {"T1114", "Email Collection"}},
	"Command and Control":  {{"T1071", "Application Layer Protocol"}, {"T1105", "Ingress Tool Transfer"}, {"T1572", "Protocol Tunneling"}},
	"Exfiltration":         {{"T1041", "Exfiltration Over C2 Channel"}, {"T1048", "Exfiltration Over Alternative Protocol"}},
	"Impact":               {{"T1486", "Data Encrypted for Impact"}, {"T1489", "Service Stop"}, {"T1529", "System Shutdown/Reboot"}},
}

// GetCoverageReport computes detection coverage across the MITRE ATT&CK matrix.
func (s *SimulationService) GetCoverageReport() CoverageReport {
	scenarios := s.ListScenarios()

	// Map technique ID (base, without sub-technique suffix) to its scenario
	coveredMap := make(map[string]string) // techniqueID -> scenarioID
	for _, sc := range scenarios {
		baseID := sc.MitreID
		if idx := strings.Index(baseID, "."); idx > 0 {
			baseID = baseID[:idx]
		}
		coveredMap[baseID] = sc.ID
	}

	var report CoverageReport
	var allCovered, allGap []string

	// Tactical ordering for deterministic output
	tacticOrder := []string{
		"Initial Access", "Execution", "Persistence", "Privilege Escalation",
		"Defense Evasion", "Credential Access", "Discovery", "Lateral Movement",
		"Collection", "Command and Control", "Exfiltration", "Impact",
	}

	tacticIDMap := make(map[string]string)
	for id, name := range detection.Tactics {
		tacticIDMap[name] = id
	}

	for _, tactic := range tacticOrder {
		techniques := mitreMatrix[tactic]
		tc := TacticCoverage{
			TacticID: tacticIDMap[tactic],
			Tactic:   tactic,
			Total:    len(techniques),
		}
		for _, tech := range techniques {
			scenarioID, ok := coveredMap[tech.ID]
			ts := TechniqueStatus{
				ID:      tech.ID,
				Name:    tech.Name,
				Covered: ok,
			}
			if ok {
				ts.Scenario = scenarioID
				tc.Covered++
				allCovered = append(allCovered, tech.ID)
			} else {
				allGap = append(allGap, tech.ID)
			}
			tc.Techniques = append(tc.Techniques, ts)
		}
		if tc.Total > 0 {
			tc.Percent = float64(tc.Covered) / float64(tc.Total) * 100
		}
		report.TacticBreakdown = append(report.TacticBreakdown, tc)
		report.TotalTechniques += tc.Total
		report.CoveredTechniques += tc.Covered
	}

	report.GapTechniques = report.TotalTechniques - report.CoveredTechniques
	if report.TotalTechniques > 0 {
		report.CoveragePercent = float64(report.CoveredTechniques) / float64(report.TotalTechniques) * 100
	}
	report.CoveredIDs = allCovered
	report.GapIDs = allGap
	return report
}

// RunContinuousValidation executes all scenarios and records the outcome.
func (s *SimulationService) RunContinuousValidation() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()
	scenarios := s.ListScenarios()
	detected := 0

	for _, sc := range scenarios {
		target := "validation-node"
		if sc.TargetType == "file" {
			target = "/tmp/.canary_validation"
		}
		if err := s.RunScenario(sc.ID, target); err != nil {
			s.log.Warn("Validation scenario %s failed: %v", sc.ID, err)
			continue
		}
		// Allow a brief window for detection correlation
		time.Sleep(500 * time.Millisecond)

		// Check if it was detected (removed from activeSimulations by the alert listener)
		key := sc.ID + target
		if _, stillPending := s.activeSimulations[key]; !stillPending {
			detected++
		} else {
			// Clean up — mark as missed
			delete(s.activeSimulations, key)
		}
	}

	duration := time.Since(start)
	passRate := float64(0)
	if len(scenarios) > 0 {
		passRate = float64(detected) / float64(len(scenarios)) * 100
	}

	coverage := s.GetCoverageReport()
	run := ValidationRun{
		ID:             fmt.Sprintf("val_%d", time.Now().Unix()),
		Timestamp:      start.Format(time.RFC3339),
		TotalScenarios: len(scenarios),
		Detected:       detected,
		Missed:         len(scenarios) - detected,
		PassRate:       passRate,
		CoverageIndex:  coverage.CoveragePercent,
		DurationMs:     duration.Milliseconds(),
	}

	// Bounded history (keep last 100 runs)
	if len(s.validationHistory) >= 100 {
		s.validationHistory = s.validationHistory[1:]
	}
	s.validationHistory = append(s.validationHistory, run)

	s.bus.Publish("simulation.validation_complete", map[string]interface{}{
		"run_id":         run.ID,
		"pass_rate":      run.PassRate,
		"coverage_index": run.CoverageIndex,
		"detected":       run.Detected,
		"total":          run.TotalScenarios,
	})

	s.log.Info("Validation complete: %d/%d detected (%.1f%%), coverage %.1f%%",
		detected, len(scenarios), passRate, coverage.CoveragePercent)
	return nil
}

// GetValidationHistory returns past validation runs.
func (s *SimulationService) GetValidationHistory() []ValidationRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ValidationRun, len(s.validationHistory))
	copy(out, s.validationHistory)
	return out
}

// GetPurpleTeamReport builds the composite Purple Team scoring report.
func (s *SimulationService) GetPurpleTeamReport() PurpleTeamReport {
	coverage := s.GetCoverageReport()
	history := s.GetValidationHistory()

	report := PurpleTeamReport{
		Coverage:          coverage,
		CoverageIndex:     coverage.CoveragePercent,
		ValidationHistory: history,
	}

	if len(history) > 0 {
		last := history[len(history)-1]
		report.LastValidation = &last
		report.DetectionRate = last.PassRate
		report.MeanResponseMs = last.DurationMs / int64(max(last.TotalScenarios, 1))
	}

	// Resilience = 40% detection rate + 40% coverage + 20% response speed bonus
	detectionWeight := report.DetectionRate * 0.4
	coverageWeight := report.CoverageIndex * 0.4
	// Speed bonus: full marks if mean response < 2000ms, degrades linearly
	speedBonus := float64(20)
	if report.MeanResponseMs > 2000 {
		speedBonus = max(0, 20-float64(report.MeanResponseMs-2000)/500)
	}
	report.ResilienceScore = detectionWeight + coverageWeight + speedBonus
	report.ResilienceGrade = gradeFromScore(report.ResilienceScore)

	return report
}
