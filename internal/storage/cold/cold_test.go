package cold

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestLocalRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.Put(ctx, "tenant/a.parquet", strings.NewReader("body-a")); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(ctx, "tenant/b.parquet", strings.NewReader("body-b")); err != nil {
		t.Fatal(err)
	}
	infos, err := s.List(ctx, "tenant")
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 2 {
		t.Errorf("list = %d", len(infos))
	}
	r, err := s.Get(ctx, "tenant/a.parquet")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(r)
	r.Close()
	if string(body) != "body-a" {
		t.Errorf("body = %q", body)
	}

	// Delete one and confirm List shrinks.
	if err := s.Delete(ctx, "tenant/a.parquet"); err != nil {
		t.Fatal(err)
	}
	infos2, _ := s.List(ctx, "tenant")
	if len(infos2) != 1 {
		t.Errorf("after delete: %d", len(infos2))
	}
}
