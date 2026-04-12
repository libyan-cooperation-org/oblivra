package risk

import (
	"context"
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type mockHostStore struct {
	database.HostStore
	hosts map[string]database.Host
}

func (m *mockHostStore) GetByID(ctx context.Context, id string) (*database.Host, error) {
	h, ok := m.hosts[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &h, nil
}

func TestRiskScoring(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	bus := eventbus.NewBus(log)

	mockHosts := &mockHostStore{
		hosts: map[string]database.Host{
			"prod-01": {ID: "prod-01", Label: "Production"},
			"dc-01":   {ID: "dc-01", Label: "Domain Controller"},
			"dev-01":  {ID: "dev-01", Label: "Development"},
		},
	}
	engine := NewRiskEngine(bus, nil, mockHosts, nil, log)

	tests := []struct {
		name     string
		event    ConfigChangeEvent
		minScore int
	}{
		{
			name: "Fleet-wide change",
			event: ConfigChangeEvent{
				Target: "fleet",
			},
			minScore: 50,
		},
		{
			name: "Sensitive collector on production",
			event: ConfigChangeEvent{
				Target:  "prod-01",
				Changes: []string{"ebpf"},
			},
			minScore: 60,
		},
		{
			name: "Routine change on dev",
			event: ConfigChangeEvent{
				Target:  "dev-01",
				Changes: []string{"ui_theme"},
			},
			minScore: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.CalculateScore(tt.event)
			if score.Score < tt.minScore {
				t.Errorf("Expected minimum score %d, got %d", tt.minScore, score.Score)
			}
		})
	}
}
