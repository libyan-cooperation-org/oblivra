package threatintel

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// STIX models for Threat Intelligence representations
// We use a simplified subset of STIX 2.1 for matching purposes

type Bundle struct {
	Type    string   `json:"type"`
	ID      string   `json:"id"`
	Objects []Object `json:"objects"`
}

type Object struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Pattern     string    `json:"pattern"`
	PatternType string    `json:"pattern_type"`
	ValidFrom   time.Time `json:"valid_from"`
	Labels      []string  `json:"labels"`
}

// TAXIIClient pulls intelligence feeds from remote TAXII 2.1 servers
type TAXIIClient struct {
	endpoint string
	username string
	password string
	client   *http.Client
}

func NewTAXIIClient(endpoint, username, password string, skipVerify bool) *TAXIIClient {
	if skipVerify {
		log.Printf("⚠️  SECURITY: TLS certificate verification disabled for TAXII endpoint %s", endpoint)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	}
	return &TAXIIClient{
		endpoint: endpoint,
		username: username,
		password: password,
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// FetchCollection retrieves a STIX bundle from a specific TAXII collection
func (c *TAXIIClient) FetchCollection(collectionID string) (*Bundle, error) {
	url := fmt.Sprintf("%s/taxii2/collections/%s/objects", c.endpoint, collectionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/taxii+json;version=2.1")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("taxii request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("taxii returned status %d: %s", resp.StatusCode, string(body))
	}

	var bundle Bundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("decode stix bundle: %w", err)
	}

	return &bundle, nil
}

// LoadLocalBundle loads a STIX bundle from a local filesystem for air-gapped ingestion
func (c *TAXIIClient) LoadLocalBundle(filePath string) (*Bundle, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read local bundle: %w", err)
	}

	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("decode local stix bundle: %w", err)
	}

	return &bundle, nil
}
