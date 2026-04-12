package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/identity"
)

func init() {
	Register("okta", NewOktaConnector)
}

type OktaConfig struct {
	Domain string `json:"domain"`
	APIKey string `json:"api_key"`
}

type OktaConnector struct {
	id     string
	config OktaConfig
	client *http.Client
}

func NewOktaConnector(id string, configJSON string) (IdentityConnector, error) {
	var cfg OktaConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid okta config: %w", err)
	}

	return &OktaConnector{
		id:     id,
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *OktaConnector) ID() string   { return c.id }
func (c *OktaConnector) Type() string { return "okta" }

func (c *OktaConnector) Verify(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/api/v1/users?limit=1", c.config.Domain), nil)
	c.setAuth(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("okta verification failed with status: %d", resp.StatusCode)
	}
	return nil
}

func (c *OktaConnector) FetchUsers(ctx context.Context) ([]*identity.UserResource, error) {
	url := fmt.Sprintf("https://%s/api/v1/users?limit=200", c.config.Domain)
	var allUsers []*identity.UserResource

	for url != "" {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		c.setAuth(req)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		
		var oktaUsers []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Profile struct {
				Login       string `json:"login"`
				FirstName   string `json:"firstName"`
				LastName    string `json:"lastName"`
				Email       string `json:"email"`
				DisplayName string `json:"displayName"`
				Title       string `json:"title"`
				Department  string `json:"department"`
				Organization string `json:"organization"`
			} `json:"profile"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&oktaUsers); err != nil {
			resp.Body.Close()
			return nil, err
		}
		
		linkHeader := resp.Header.Get("Link")
		resp.Body.Close()

		for _, ou := range oktaUsers {
			res := &identity.UserResource{
				ID:         ou.ID,
				ExternalID: ou.ID,
				UserName:   ou.Profile.Login,
				DisplayName: ou.Profile.DisplayName,
				Active:      ou.Status == "ACTIVE",
				Title:       ou.Profile.Title,
			}
			res.Name.Formatted = ou.Profile.FirstName + " " + ou.Profile.LastName
			res.Name.GivenName = ou.Profile.FirstName
			res.Name.FamilyName = ou.Profile.LastName
			res.Emails = []identity.Email{{Value: ou.Profile.Email, Primary: true}}
			
			// Note: We can expand this with custom attribute mapping in a future PR
			allUsers = append(allUsers, res)
		}

		url = getNextLink(linkHeader)
	}

	return allUsers, nil
}

func (c *OktaConnector) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "SSWS "+c.config.APIKey)
	req.Header.Set("Accept", "application/json")
}

func getNextLink(header string) string {
	if header == "" {
		return ""
	}
	links := strings.Split(header, ",")
	for _, link := range links {
		if strings.Contains(link, `rel="next"`) {
			parts := strings.Split(link, ";")
			if len(parts) > 0 {
				return strings.Trim(parts[0], "<> ")
			}
		}
	}
	return ""
}
