package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kingknull/oblivrashell/internal/identity"
	"golang.org/x/oauth2/clientcredentials"
)

func init() {
	Register("azure_ad", NewAzureADConnector)
}

type AzureADConfig struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type AzureADConnector struct {
	id     string
	config AzureADConfig
	client *http.Client
}

func NewAzureADConnector(id string, configJSON string) (IdentityConnector, error) {
	var cfg AzureADConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid azure_ad config: %w", err)
	}

	return &AzureADConnector{
		id:     id,
		config: cfg,
	}, nil
}

func (c *AzureADConnector) ID() string   { return c.id }
func (c *AzureADConnector) Type() string { return "azure_ad" }

func (c *AzureADConnector) getClient(ctx context.Context) *http.Client {
	if c.client != nil {
		return c.client
	}
	conf := &clientcredentials.Config{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		TokenURL:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.config.TenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
	}
	c.client = conf.Client(ctx)
	return c.client
}

func (c *AzureADConnector) Verify(ctx context.Context) error {
	client := c.getClient(ctx)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/users?$top=1")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("azure_ad verification failed with status: %d", resp.StatusCode)
	}
	return nil
}

func (c *AzureADConnector) FetchUsers(ctx context.Context) ([]*identity.UserResource, error) {
	client := c.getClient(ctx)
	url := "https://graph.microsoft.com/v1.0/users?$select=id,displayName,givenName,surname,mail,userPrincipalName,jobTitle,businessPhones,usageLocation,accountEnabled"
	var allUsers []*identity.UserResource

	for url != "" {
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		
		var result struct {
			Value []struct {
				ID                string   `json:"id"`
				DisplayName       string   `json:"displayName"`
				GivenName         string   `json:"givenName"`
				Surname           string   `json:"surname"`
				Mail              string   `json:"mail"`
				UserPrincipalName string   `json:"userPrincipalName"`
				JobTitle          string   `json:"jobTitle"`
				BusinessPhones    []string `json:"businessPhones"`
				UsageLocation     string   `json:"usageLocation"`
				AccountEnabled    bool     `json:"accountEnabled"`
			} `json:"value"`
			NextLink string `json:"@odata.nextLink"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, au := range result.Value {
			email := au.Mail
			if email == "" {
				email = au.UserPrincipalName
			}
			res := &identity.UserResource{
				ID:         au.ID,
				ExternalID: au.ID,
				UserName:   email,
				DisplayName: au.DisplayName,
				Active:      au.AccountEnabled,
				Title:       au.JobTitle,
				Locale:      au.UsageLocation,
			}
			res.Name.Formatted = au.DisplayName
			res.Name.GivenName = au.GivenName
			res.Name.FamilyName = au.Surname
			res.Emails = []identity.Email{{Value: email, Primary: true}}
			
			if len(au.BusinessPhones) > 0 {
				res.PhoneNumbers = []identity.PhoneNumber{{Value: au.BusinessPhones[0], Type: "work", Primary: true}}
			}
			
			allUsers = append(allUsers, res)
		}

		url = result.NextLink
	}

	return allUsers, nil
}
