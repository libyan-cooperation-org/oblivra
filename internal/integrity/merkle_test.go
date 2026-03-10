package integrity

import (
	"fmt"
	"sync"
	"testing"
)

func TestAddLeafAndRoot(t *testing.T) {
	tree := New()

	// Empty tree
	if root := tree.Root(); root != "" {
		t.Errorf("expected empty root, got %s", root)
	}

	// Single leaf
	h1, idx1, err := tree.AddLeaf([]byte("event_1"))
	if err != nil {
		t.Fatal(err)
	}
	if idx1 != 0 {
		t.Errorf("expected index 0, got %d", idx1)
	}
	if h1 == "" {
		t.Error("expected non-empty hash")
	}

	root1 := tree.Root()
	if root1 != h1 {
		t.Errorf("single leaf root should equal leaf hash")
	}

	// Second leaf changes root
	_, _, err = tree.AddLeaf([]byte("event_2"))
	if err != nil {
		t.Fatal(err)
	}
	root2 := tree.Root()
	if root2 == root1 {
		t.Error("root should change after adding second leaf")
	}
}

func TestVerify(t *testing.T) {
	tree := New()

	data := [][]byte{
		[]byte("audit_log_1"),
		[]byte("audit_log_2"),
		[]byte("audit_log_3"),
		[]byte("audit_log_4"),
		[]byte("audit_log_5"),
	}

	for _, d := range data {
		if _, _, err := tree.AddLeaf(d); err != nil {
			t.Fatal(err)
		}
	}

	// Verify all leaves
	for i, d := range data {
		if !tree.Verify(i, d) {
			t.Errorf("leaf %d should verify", i)
		}
	}

	// Tampered data should fail
	if tree.Verify(0, []byte("tampered_data")) {
		t.Error("tampered data should not verify")
	}

	// Out of range should fail
	if tree.Verify(999, []byte("anything")) {
		t.Error("out of range index should not verify")
	}
}

func TestGenerateAndVerifyProof(t *testing.T) {
	tree := New()

	for i := 0; i < 8; i++ {
		if _, _, err := tree.AddLeaf([]byte(fmt.Sprintf("event_%d", i))); err != nil {
			t.Fatal(err)
		}
	}

	root := tree.Root()

	// Generate and verify proof for each leaf
	for i := 0; i < 8; i++ {
		proof, err := tree.GenerateProof(i)
		if err != nil {
			t.Fatal(err)
		}

		data := []byte(fmt.Sprintf("event_%d", i))
		if !VerifyExternal(data, proof, root) {
			t.Errorf("external verification failed for leaf %d", i)
		}

		// Tampered data should fail external verification
		if VerifyExternal([]byte("tampered"), proof, root) {
			t.Errorf("tampered data should fail external verification for leaf %d", i)
		}
	}
}

func TestExportImportState(t *testing.T) {
	tree := New()

	for i := 0; i < 10; i++ {
		if _, _, err := tree.AddLeaf([]byte(fmt.Sprintf("log_%d", i))); err != nil {
			t.Fatal(err)
		}
	}

	originalRoot := tree.Root()
	originalCount := tree.LeafCount()

	// Export
	state, err := tree.ExportState()
	if err != nil {
		t.Fatal(err)
	}

	// Import into new tree
	tree2 := New()
	if err := tree2.ImportState(state); err != nil {
		t.Fatal(err)
	}

	if tree2.Root() != originalRoot {
		t.Error("imported tree root should match original")
	}
	if tree2.LeafCount() != originalCount {
		t.Errorf("expected %d leaves, got %d", originalCount, tree2.LeafCount())
	}
}

func TestPersistenceCallback(t *testing.T) {
	var persisted []struct {
		index int
		hash  Hash
	}
	var mu sync.Mutex

	tree := NewWithPersistence(func(index int, hash Hash, data []byte) error {
		mu.Lock()
		defer mu.Unlock()
		persisted = append(persisted, struct {
			index int
			hash  Hash
		}{index, hash})
		return nil
	})

	for i := 0; i < 5; i++ {
		if _, _, err := tree.AddLeaf([]byte(fmt.Sprintf("data_%d", i))); err != nil {
			t.Fatal(err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(persisted) != 5 {
		t.Errorf("expected 5 persisted entries, got %d", len(persisted))
	}
}

func TestConcurrentAccess(t *testing.T) {
	tree := New()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _, _ = tree.AddLeaf([]byte(fmt.Sprintf("concurrent_%d", n)))
		}(i)
	}

	wg.Wait()

	if tree.LeafCount() != 100 {
		t.Errorf("expected 100 leaves, got %d", tree.LeafCount())
	}

	// Root should be computable
	root := tree.Root()
	if root == "" {
		t.Error("root should not be empty after 100 inserts")
	}
}

func TestOddNumberOfLeaves(t *testing.T) {
	// Odd counts require duplication of the last node
	for _, count := range []int{1, 3, 5, 7, 13} {
		tree := New()
		for i := 0; i < count; i++ {
			if _, _, err := tree.AddLeaf([]byte(fmt.Sprintf("odd_%d", i))); err != nil {
				t.Fatal(err)
			}
		}
		root := tree.Root()
		if root == "" {
			t.Errorf("root should not be empty for %d leaves", count)
		}

		// Verify proofs work with odd counts
		for i := 0; i < count; i++ {
			proof, err := tree.GenerateProof(i)
			if err != nil {
				t.Fatal(err)
			}
			if !VerifyExternal([]byte(fmt.Sprintf("odd_%d", i)), proof, root) {
				t.Errorf("proof verification failed for leaf %d with %d total leaves", i, count)
			}
		}
	}
}
