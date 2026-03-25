package forensics

import (
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// mockSignerForTest simulates signing behavior for testing without hardware.
type mockSignerForTest struct {
	signatures []string
}

func (m *mockSignerForTest) SignEntry(payload string) (string, error) {
	end := len(payload)
	if end > 10 {
		end = 10
	}
	sig := "mock-sig-" + payload[:end]
	m.signatures = append(m.signatures, sig)
	return sig, nil
}

func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	l, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	return l
}

func TestTPMSignerFallback(t *testing.T) {
	// On CI/test machines, TPM is almost certainly unavailable.
	// Verify graceful fallback to HMAC.
	signer := NewTPMSigner(TPMSignerConfig{
		FallbackKey: []byte("test-hmac-key-32-bytes-exactly!!"),
	}, testLogger(t))

	sig, err := signer.SignEntry("test-payload-for-signing")
	if err != nil {
		t.Fatalf("SignEntry failed: %v", err)
	}

	if len(sig) < 6 {
		t.Errorf("Signature too short: %q", sig)
	}

	t.Logf("IsHardwareRooted: %v, Signature prefix: %s...", signer.IsHardwareRooted(), sig[:20])
}

func TestTPMSignerDeterministic(t *testing.T) {
	key := []byte("deterministic-key-for-test-00000")
	signer := NewTPMSigner(TPMSignerConfig{
		FallbackKey: key,
	}, testLogger(t))

	payload := "action|actor|2024-01-01T00:00:00Z|notes|prev_hash"

	sig1, _ := signer.SignEntry(payload)
	sig2, _ := signer.SignEntry(payload)

	if sig1 != sig2 {
		t.Errorf("Signatures not deterministic: %s != %s", sig1, sig2)
	}
}

func TestEvidenceLockerCollectAndVerify(t *testing.T) {
	signer := &mockSignerForTest{}
	locker := NewEvidenceLocker(signer, testLogger(t))

	item, err := locker.Collect("INC-001", EvidenceFile, "malicious.exe", []byte("binary-data"), "analyst-1", "Found on compromised host")
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if item.ID == "" {
		t.Error("Expected non-empty evidence ID")
	}
	if item.IncidentID != "INC-001" {
		t.Errorf("Wrong incident ID: %s", item.IncidentID)
	}
	if len(item.ChainOfCustody) != 1 {
		t.Errorf("Expected 1 chain entry, got %d", len(item.ChainOfCustody))
	}
	if item.ChainOfCustody[0].Action != ActionCollected {
		t.Errorf("Expected 'collected' action, got %s", item.ChainOfCustody[0].Action)
	}

	// Transfer
	if err := locker.Transfer(item.ID, "analyst-2", "Transferred for analysis"); err != nil {
		t.Fatalf("Transfer: %v", err)
	}

	// Verify chain
	valid, err := locker.Verify(item.ID)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !valid {
		t.Error("Chain-of-custody verification failed — should be valid")
	}

	// Seal
	if err := locker.Seal(item.ID, "analyst-1", "Sealed for court"); err != nil {
		t.Fatalf("Seal: %v", err)
	}

	// Verify sealed evidence can't be modified
	if err := locker.Transfer(item.ID, "analyst-3", "Should fail"); err == nil {
		t.Error("Expected error when transferring sealed evidence")
	}
}

func TestEvidenceLockerListAll(t *testing.T) {
	signer := &mockSignerForTest{}
	locker := NewEvidenceLocker(signer, testLogger(t))

	locker.Collect("INC-001", EvidenceFile, "file1.exe", []byte("data1"), "analyst", "")
	locker.Collect("INC-001", EvidenceLog, "auth.log", []byte("data2"), "analyst", "")
	locker.Collect("INC-002", EvidencePCAP, "capture.pcap", []byte("data3"), "analyst", "")

	all := locker.ListAll()
	if len(all) != 3 {
		t.Errorf("Expected 3 items, got %d", len(all))
	}

	inc1 := locker.ListByIncident("INC-001")
	if len(inc1) != 2 {
		t.Errorf("Expected 2 items for INC-001, got %d", len(inc1))
	}
}
