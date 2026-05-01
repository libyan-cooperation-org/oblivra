//go:build !airgap

// Package cold S3 adapter — built only when the `airgap` build tag is NOT
// set. Air-gapped deployments skip this file entirely so the binary doesn't
// link an HTTP-out cloud SDK.
//
// This adapter is deliberately minimal and SDK-free: it speaks raw HTTP +
// AWS Signature V4 for PutObject / GetObject / ListObjectsV2 / DeleteObject.
// That keeps the binary small and the dependency graph short. For richer S3
// usage (versioning, KMS keys, multipart uploads), bring an SDK build-tag.
package cold

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type S3Config struct {
	Endpoint        string // e.g. https://s3.amazonaws.com  or  https://minio.local:9000
	Region          string
	Bucket          string
	AccessKey       string
	SecretKey       string
	PathStyle       bool          // true for MinIO / non-AWS endpoints
	HTTPClient      *http.Client  // nil → default (10s timeout)
	UploadTimeout   time.Duration // per-Put deadline
}

// S3Store implements ObjectStore against any S3-API-compatible endpoint.
type S3Store struct {
	cfg S3Config
}

func NewS3Store(cfg S3Config) (*S3Store, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, errors.New("cold/s3: endpoint and bucket required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.UploadTimeout == 0 {
		cfg.UploadTimeout = 60 * time.Second
	}
	return &S3Store{cfg: cfg}, nil
}

func (s *S3Store) Backend() string { return "s3" }

func (s *S3Store) objectURL(key string) string {
	endpoint := strings.TrimRight(s.cfg.Endpoint, "/")
	key = strings.TrimLeft(key, "/")
	if s.cfg.PathStyle {
		return endpoint + "/" + s.cfg.Bucket + "/" + key
	}
	// virtual-host: https://bucket.s3.amazonaws.com/key
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint + "/" + s.cfg.Bucket + "/" + key
	}
	u.Host = s.cfg.Bucket + "." + u.Host
	u.Path = "/" + key
	return u.String()
}

func (s *S3Store) Put(ctx context.Context, key string, body io.Reader) error {
	buf, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	uploadCtx, cancel := context.WithTimeout(ctx, s.cfg.UploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(uploadCtx, "PUT", s.objectURL(key), bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if err := s.sign(req, buf); err != nil {
		return err
	}
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cold/s3 put: %s: %s", resp.Status, string(body))
	}
	return nil
}

func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.objectURL(key), nil)
	if err != nil {
		return nil, err
	}
	if err := s.sign(req, nil); err != nil {
		return nil, err
	}
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cold/s3 get: %s: %s", resp.Status, string(body))
	}
	return resp.Body, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.objectURL(key), nil)
	if err != nil {
		return err
	}
	if err := s.sign(req, nil); err != nil {
		return err
	}
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cold/s3 delete: %s: %s", resp.Status, string(body))
	}
	return nil
}

type listResult struct {
	XMLName  xml.Name `xml:"ListBucketResult"`
	Contents []struct {
		Key          string    `xml:"Key"`
		LastModified time.Time `xml:"LastModified"`
		Size         int64     `xml:"Size"`
	} `xml:"Contents"`
}

func (s *S3Store) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	endpoint := strings.TrimRight(s.cfg.Endpoint, "/")
	listURL := endpoint
	if s.cfg.PathStyle {
		listURL += "/" + s.cfg.Bucket
	}
	q := url.Values{}
	q.Set("list-type", "2")
	if prefix != "" {
		q.Set("prefix", prefix)
	}
	listURL += "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, err
	}
	if err := s.sign(req, nil); err != nil {
		return nil, err
	}
	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cold/s3 list: %s: %s", resp.Status, string(body))
	}
	var lr listResult
	if err := xml.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	out := make([]ObjectInfo, 0, len(lr.Contents))
	for _, c := range lr.Contents {
		out = append(out, ObjectInfo{Key: c.Key, Size: c.Size, Modified: c.LastModified})
	}
	return out, nil
}

// ---- AWS Signature V4 (the minimum subset we need) ----

func (s *S3Store) sign(req *http.Request, body []byte) error {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	host := req.URL.Host
	bodyHash := hashHex(body)

	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", bodyHash)

	signedHeaders, canonHeaders := buildHeaders(req)
	canonReq := strings.Join([]string{
		req.Method,
		canonURI(req.URL),
		canonQuery(req.URL),
		canonHeaders + "\n",
		signedHeaders,
		bodyHash,
	}, "\n")
	scope := dateStamp + "/" + s.cfg.Region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hashHex([]byte(canonReq)),
	}, "\n")

	kDate := hmacBytes([]byte("AWS4"+s.cfg.SecretKey), []byte(dateStamp))
	kRegion := hmacBytes(kDate, []byte(s.cfg.Region))
	kService := hmacBytes(kRegion, []byte("s3"))
	kSigning := hmacBytes(kService, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacBytes(kSigning, []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.cfg.AccessKey, scope, signedHeaders, signature,
	))
	return nil
}

func hashHex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacBytes(key, msg []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return mac.Sum(nil)
}

func canonURI(u *url.URL) string {
	if u.Path == "" {
		return "/"
	}
	return u.EscapedPath()
}

func canonQuery(u *url.URL) string {
	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func buildHeaders(req *http.Request) (signed, canon string) {
	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		lk := strings.ToLower(k)
		if lk == "host" || strings.HasPrefix(lk, "x-amz-") || lk == "content-type" {
			keys = append(keys, lk)
		}
	}
	sort.Strings(keys)
	canonLines := make([]string, 0, len(keys))
	for _, k := range keys {
		var v string
		if k == "host" {
			v = req.Header.Get("Host")
		} else {
			v = strings.TrimSpace(req.Header.Get(k))
		}
		canonLines = append(canonLines, k+":"+v)
	}
	return strings.Join(keys, ";"), strings.Join(canonLines, "\n")
}
