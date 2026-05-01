//go:build !airgap

package cold

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestS3PutGetListDelete drives the local SigV4-only S3 adapter against an
// httptest server that records calls and replies with realistic XML. We
// don't validate the signature exactly (that path is in the AWS SDK's own
// tests); we validate that headers were set and the round-trip works.
func TestS3PutGetListDelete(t *testing.T) {
	var stored atomic.Pointer[[]byte]

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Every request must carry the AWS SigV4 markers.
		if r.Header.Get("X-Amz-Date") == "" {
			t.Errorf("%s %s: missing X-Amz-Date", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Amz-Content-Sha256") == "" {
			t.Errorf("%s %s: missing X-Amz-Content-Sha256", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Errorf("%s %s: bad authorization header", r.Method, r.URL.Path)
		}

		switch r.Method {
		case "PUT":
			body, _ := io.ReadAll(r.Body)
			stored.Store(&body)
			w.WriteHeader(200)
		case "GET":
			if strings.Contains(r.URL.RawQuery, "list-type=2") {
				// ListObjectsV2 response — minimum XML the adapter parses.
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<?xml version="1.0"?>
<ListBucketResult>
  <Contents>
    <Key>foo.parquet</Key>
    <LastModified>2026-04-30T10:00:00Z</LastModified>
    <Size>1234</Size>
  </Contents>
</ListBucketResult>`))
				return
			}
			b := stored.Load()
			if b == nil {
				w.WriteHeader(404)
				return
			}
			_, _ = w.Write(*b)
		case "DELETE":
			stored.Store(nil)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer srv.Close()

	store, err := NewS3Store(S3Config{
		Endpoint: srv.URL, Bucket: "oblivra", AccessKey: "AK", SecretKey: "sk",
		Region: "us-east-1", PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.Backend() != "s3" {
		t.Errorf("backend = %q", store.Backend())
	}
	ctx := context.Background()

	if err := store.Put(ctx, "tenants/a/2026/01.parquet", strings.NewReader("payload")); err != nil {
		t.Fatal(err)
	}
	r, err := store.Get(ctx, "tenants/a/2026/01.parquet")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if string(got) != "payload" {
		t.Errorf("get returned %q, want %q", got, "payload")
	}

	infos, err := store.List(ctx, "tenants/a")
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 || infos[0].Key != "foo.parquet" {
		t.Errorf("list = %+v", infos)
	}

	if err := store.Delete(ctx, "tenants/a/2026/01.parquet"); err != nil {
		t.Fatal(err)
	}
}

// TestS3RejectsBadConfig confirms the constructor refuses obvious invalid
// configurations rather than silently producing a broken client.
func TestS3RejectsBadConfig(t *testing.T) {
	if _, err := NewS3Store(S3Config{Endpoint: "", Bucket: "x"}); err == nil {
		t.Error("expected error for empty endpoint")
	}
	if _, err := NewS3Store(S3Config{Endpoint: "x", Bucket: ""}); err == nil {
		t.Error("expected error for empty bucket")
	}
}
