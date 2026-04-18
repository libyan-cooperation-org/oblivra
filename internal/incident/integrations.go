package incident

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ─── Jira ────────────────────────────────────────────────────────────────────

// JiraConfig holds connection details for a Jira Cloud or Server instance.
type JiraConfig struct {
	BaseURL   string // e.g. "https://yourorg.atlassian.net"
	Email     string // Jira Cloud: user email
	APIToken  string // Jira Cloud: API token / Jira Server: password
	ProjectKey string // e.g. "SEC"
	IssueType  string // e.g. "Bug", "Incident", "Task"
}

// JiraClient is a minimal Jira REST API v3 client for incident ticket creation.
type JiraClient struct {
	cfg    JiraConfig
	http   *http.Client
}

// NewJiraClient creates a JiraClient. Returns an error if config is incomplete.
func NewJiraClient(cfg JiraConfig) (*JiraClient, error) {
	if cfg.BaseURL == "" || cfg.Email == "" || cfg.APIToken == "" || cfg.ProjectKey == "" {
		return nil, fmt.Errorf("jira: BaseURL, Email, APIToken and ProjectKey are all required")
	}
	if cfg.IssueType == "" {
		cfg.IssueType = "Task"
	}
	return &JiraClient{
		cfg:  cfg,
		http: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// jiraIssueRequest is the Jira REST API v3 create-issue payload.
type jiraIssueRequest struct {
	Fields jiraFields `json:"fields"`
}

type jiraFields struct {
	Project     jiraProjectRef   `json:"project"`
	Summary     string           `json:"summary"`
	Description jiraDescription  `json:"description"`
	IssueType   jiraIssueTypeRef `json:"issuetype"`
	Priority    jiraRef          `json:"priority"`
	Labels      []string         `json:"labels"`
}

type jiraProjectRef struct {
	Key string `json:"key"`
}

type jiraIssueTypeRef struct {
	Name string `json:"name"`
}

type jiraRef struct {
	Name string `json:"name"`
}

// jiraDescription uses Atlassian Document Format (ADF) for cloud compatibility.
type jiraDescription struct {
	Type    string      `json:"type"`
	Version int         `json:"version"`
	Content []jiraADFNode `json:"content"`
}

type jiraADFNode struct {
	Type    string        `json:"type"`
	Content []jiraADFNode `json:"content,omitempty"`
	Text    string        `json:"text,omitempty"`
}

// jiraIssueResponse is the subset of the Jira response we care about.
type jiraIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// CreateIncidentTicket opens a new Jira issue for a security incident.
// Returns the created issue key (e.g. "SEC-123") on success.
func (c *JiraClient) CreateIncidentTicket(ctx context.Context, title, description, severity string, labels []string) (string, error) {
	// Map OBLIVRA severity to Jira priority
	priority := "Medium"
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		priority = "Highest"
	case "HIGH":
		priority = "High"
	case "LOW":
		priority = "Low"
	}

	body := jiraIssueRequest{
		Fields: jiraFields{
			Project:   jiraProjectRef{Key: c.cfg.ProjectKey},
			Summary:   fmt.Sprintf("[OBLIVRA] %s", title),
			IssueType: jiraIssueTypeRef{Name: c.cfg.IssueType},
			Priority:  jiraRef{Name: priority},
			Labels:    append(labels, "oblivra", "security"),
			Description: jiraDescription{
				Type:    "doc",
				Version: 1,
				Content: []jiraADFNode{
					{
						Type: "paragraph",
						Content: []jiraADFNode{
							{Type: "text", Text: description},
						},
					},
					{
						Type: "paragraph",
						Content: []jiraADFNode{
							{Type: "text", Text: fmt.Sprintf("Severity: %s | Generated: %s", severity, time.Now().UTC().Format(time.RFC3339))},
						},
					},
				},
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("jira: marshal payload: %w", err)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/rest/api/3/issue"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("jira: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.cfg.Email, c.cfg.APIToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("jira: POST /issue: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("jira: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result jiraIssueResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("jira: decode response: %w", err)
	}

	return result.Key, nil
}

// ─── ServiceNow ──────────────────────────────────────────────────────────────

// ServiceNowConfig holds connection details for a ServiceNow instance.
type ServiceNowConfig struct {
	InstanceURL string // e.g. "https://yourorg.service-now.com"
	Username    string
	Password    string
	// Caller ID is the sys_id of the default caller — optional.
	CallerSysID string
}

// ServiceNowClient is a minimal ServiceNow Table API client for incident creation.
type ServiceNowClient struct {
	cfg  ServiceNowConfig
	http *http.Client
}

// NewServiceNowClient creates a ServiceNowClient.
func NewServiceNowClient(cfg ServiceNowConfig) (*ServiceNowClient, error) {
	if cfg.InstanceURL == "" || cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("servicenow: InstanceURL, Username and Password are all required")
	}
	return &ServiceNowClient{
		cfg:  cfg,
		http: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// snowIncidentRequest is the ServiceNow incident table record payload.
type snowIncidentRequest struct {
	ShortDescription string `json:"short_description"`
	Description      string `json:"description"`
	Urgency          string `json:"urgency"`    // 1=High, 2=Medium, 3=Low
	Impact           string `json:"impact"`     // 1=High, 2=Medium, 3=Low
	Category         string `json:"category"`
	Subcategory      string `json:"subcategory"`
	CallerID         string `json:"caller_id,omitempty"`
}

// snowIncidentResponse is the subset of the ServiceNow response we care about.
type snowIncidentResponse struct {
	Result struct {
		SysID  string `json:"sys_id"`
		Number string `json:"number"`
		State  string `json:"state"`
	} `json:"result"`
}

// CreateIncident opens a new ServiceNow incident record.
// Returns the incident number (e.g. "INC0010042") on success.
func (c *ServiceNowClient) CreateIncident(ctx context.Context, title, description, severity string) (string, error) {
	// Map OBLIVRA severity to ServiceNow urgency/impact (1=High, 2=Medium, 3=Low)
	urgency, impact := "2", "2"
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		urgency, impact = "1", "1"
	case "HIGH":
		urgency, impact = "1", "2"
	case "LOW":
		urgency, impact = "3", "3"
	}

	body := snowIncidentRequest{
		ShortDescription: fmt.Sprintf("[OBLIVRA] %s", title),
		Description:      fmt.Sprintf("%s\n\nSeverity: %s\nGenerated by OBLIVRA at %s", description, severity, time.Now().UTC().Format(time.RFC3339)),
		Urgency:          urgency,
		Impact:           impact,
		Category:         "Security",
		Subcategory:      "Intrusion",
		CallerID:         c.cfg.CallerSysID,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("servicenow: marshal payload: %w", err)
	}

	url := strings.TrimRight(c.cfg.InstanceURL, "/") + "/api/now/table/incident"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("servicenow: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("servicenow: POST incident: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("servicenow: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result snowIncidentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("servicenow: decode response: %w", err)
	}

	return result.Result.Number, nil
}

// UpdateIncidentState updates a ServiceNow incident's state (e.g. close or resolve).
// stateCode: 1=New, 2=In Progress, 6=Resolved, 7=Closed
func (c *ServiceNowClient) UpdateIncidentState(ctx context.Context, sysID string, stateCode int, closeNotes string) error {
	body := map[string]interface{}{
		"state":       stateCode,
		"close_notes": closeNotes,
	}
	if stateCode == 6 || stateCode == 7 {
		body["close_code"] = "Resolved by Caller"
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("servicenow: marshal patch: %w", err)
	}

	url := strings.TrimRight(c.cfg.InstanceURL, "/") + "/api/now/table/incident/" + sysID
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("servicenow: build patch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("servicenow: PATCH incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("servicenow: patch status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
