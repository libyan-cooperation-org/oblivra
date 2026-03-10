package memory

import (
	"fmt"
	"testing"
)

func TestSecureBuffer_Lifecycle(t *testing.T) {
	size := 32
	sb := NewSecureBuffer(size)
	defer sb.Wipe()

	if len(sb.Data()) != size {
		t.Fatalf("Expected size %d, got %d", size, len(sb.Data()))
	}

	// Write data
	secret := "sovereign-secret-12345"
	copy(sb.Data(), secret)

	if string(sb.Data()[:len(secret)]) != secret {
		t.Fatalf("Data mismatch: expected %s, got %s", secret, string(sb.Data()))
	}

	if sb.IsWiped() {
		t.Fatal("Buffer should not be wiped yet")
	}

	// Get active count
	count := GetActiveCount()
	if count < 1 {
		t.Fatalf("Expected at least 1 active allocation, got %d", count)
	}

	// Wipe
	sb.Wipe()

	if !sb.IsWiped() {
		t.Fatal("Buffer should be marked as wiped")
	}

	if sb.Data() != nil {
		t.Fatal("Data slice should be nil after Wipe")
	}

	newCount := GetActiveCount()
	if newCount != count-1 {
		t.Fatalf("Active count did not decrement correctly: %d -> %d", count, newCount)
	}
}

func TestSecureBuffer_FromString(t *testing.T) {
	secret := "adversarial-key-material"
	sb := FromString(secret)
	defer sb.Wipe()

	if string(sb.Data()) != secret {
		t.Fatalf("Expected %s, got %s", secret, string(sb.Data()))
	}
}

func TestSecureBuffer_DoubleWipe(t *testing.T) {
	sb := NewSecureBuffer(16)
	sb.Wipe()
	// Should not panic or double-decrement
	sb.Wipe()
}

func ExampleSecureBuffer() {
	sb := NewSecureBuffer(32)
	defer sb.Wipe()

	copy(sb.Data(), "sensitive-key")
	fmt.Printf("Buffer length: %d\n", len(sb.Data()))
	// Output: Buffer length: 32
}
