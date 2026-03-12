package services

import (
	"testing"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type mockHardwareProvider struct {
	shouldFail bool
}

func (m *mockHardwareProvider) SignIdentity(email string, nonce []byte) ([]byte, error) {
	return []byte("dummy-signature"), nil
}

func (m *mockHardwareProvider) VerifyIdentity(email string, nonce []byte, signature []byte) (bool, error) {
	if m.shouldFail {
		return false, nil
	}
	return string(signature) == "dummy-signature", nil
}

func TestIdentityService_LoginHardwareBound(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: "NUL", // Use NUL on Windows for discarding output
	})
	bus := eventbus.NewBus(log)
	
	// Mock DB components (Simplified for test)
	// In a real scenario, we'd use a real database.Database instance with an in-memory driver.
	// For this test, we'll focus on the logic flow in IdentityService.
	
	hw := &mockHardwareProvider{}
	service := &IdentityService{
		hw:  hw,
		log: log.WithPrefix("test-identity"),
		bus: bus,
		// userRepo would be needed here for a full integration test
	}

	t.Run("Hardware provider missing", func(t *testing.T) {
		service.hw = nil
		_, err := service.LoginHardwareBound("test@oblivra.com", []byte("nonce"), []byte("sig"))
		if err == nil || err.Error() != "hardware identity not enabled for this platform" {
			t.Errorf("expected error for missing hardware provider, got %v", err)
		}
		service.hw = hw
	})

	// Note: Fully testing the success case requires a functional userRepo.
	// This test confirms the service layer logic handles the hardware provider correctly.
}
