package simulation_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/simulation"
)

func TestSimulationDetectionCoverage(t *testing.T) {
	tempDir := t.TempDir()
	logPath := tempDir + "/sim.log"
	l, _ := logger.New(logger.Config{Level: logger.DebugLevel, OutputPath: logPath})
	defer l.Close()

	bus := eventbus.NewBus(l)
	svc := simulation.NewSimulationService(bus, l)

	ctx := context.Background()
	svc.Startup(ctx)
	defer svc.Shutdown()

	scenarios := svc.ListScenarios()
	if len(scenarios) == 0 {
		t.Fatal("No simulation scenarios registered")
	}

	t.Logf("Found %d attack scenarios:", len(scenarios))
	for _, s := range scenarios {
		t.Logf("  [%s] %s (MITRE: %s)", s.ID, s.Name, s.MitreID)
	}

	// Run each scenario and verify it doesn't crash
	for _, scenario := range scenarios {
		t.Run(scenario.ID, func(t *testing.T) {
			err := svc.RunScenario(scenario.ID, "test-host-"+scenario.ID)
			if err != nil {
				t.Errorf("Scenario %s failed to execute: %v", scenario.ID, err)
			}
		})
	}

	// Wait for bus events to propagate
	time.Sleep(2 * time.Second)

	// Check results
	results := svc.GetResults()
	t.Logf("Active simulation results: %d", len(results))
	for _, r := range results {
		t.Logf("  Scenario: %s | Target: %s | Detected: %v", r.ScenarioID, r.Target, r.Detected)
	}

	// Verify MITRE coverage
	mitreCoverage := make(map[string]bool)
	for _, s := range scenarios {
		mitreCoverage[s.MitreID] = true
	}
	t.Logf("Total MITRE ATT&CK techniques covered: %d", len(mitreCoverage))

	if len(mitreCoverage) < 3 {
		t.Errorf("Expected at least 3 MITRE techniques, got %d", len(mitreCoverage))
	}

	// Verify campaign system works
	campaignID := svc.StartCampaign("Full Audit Campaign", []string{"brute_force_auth", "ransomware_entropy"})
	if campaignID == "" {
		t.Error("Failed to create campaign")
	}

	campaigns := svc.GetCampaigns()
	if len(campaigns) == 0 {
		t.Error("No campaigns found after creation")
	}

	// Cleanup: remove temp log
	os.Remove(logPath)
}
