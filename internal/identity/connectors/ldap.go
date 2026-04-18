package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/kingknull/oblivrashell/internal/identity"
)

func init() {
	Register("ldap", NewLDAPConnector)
	Register("active_directory", NewLDAPConnector)
}

type LDAPConfig struct {
	URL          string `json:"url"`
	BindDN       string `json:"bind_dn"`
	BindPassword string `json:"bind_password"`
	UserBaseDN   string `json:"user_base_dn"`
	UserFilter   string `json:"user_filter"` // e.g., "(objectClass=person)"
	EmailAttr    string `json:"email_attr"`   // e.g., "mail"
	NameAttr     string `json:"name_attr"`    // e.g., "displayName"
}

type LDAPConnector struct {
	id     string
	config LDAPConfig
}

func NewLDAPConnector(id string, configJSON string) (IdentityConnector, error) {
	var cfg LDAPConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid ldap config: %w", err)
	}
	if cfg.EmailAttr == "" {
		cfg.EmailAttr = "mail"
	}
	if cfg.NameAttr == "" {
		cfg.NameAttr = "displayName"
	}

	return &LDAPConnector{id: id, config: cfg}, nil
}

func (c *LDAPConnector) ID() string   { return c.id }
func (c *LDAPConnector) Type() string { return "ldap" }

func (c *LDAPConnector) Verify(ctx context.Context) error {
	l, err := ldap.DialURL(c.config.URL)
	if err != nil {
		return fmt.Errorf("dial ldap: %w", err)
	}
	defer l.Close()

	if err := l.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		return fmt.Errorf("bind ldap: %w", err)
	}
	return nil
}

func (c *LDAPConnector) FetchUsers(ctx context.Context) ([]*identity.UserResource, error) {
	l, err := ldap.DialURL(c.config.URL)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	if err := l.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		return nil, err
	}

	searchRequest := ldap.NewSearchRequest(
		c.config.UserBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		c.config.UserFilter,
		[]string{"dn", c.config.EmailAttr, c.config.NameAttr, "title", "department", "company", "sAMAccountName", "userPrincipalName"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var users []*identity.UserResource
	for _, entry := range sr.Entries {
		email := entry.GetAttributeValue(c.config.EmailAttr)
		if email == "" {
			email = entry.GetAttributeValue("userPrincipalName")
		}
		if email == "" {
			continue // Mandatory for OQL correlation
		}

		res := &identity.UserResource{
			ID:          entry.DN,
			ExternalID:  entry.DN,
			UserName:    email,
			DisplayName: entry.GetAttributeValue(c.config.NameAttr),
			Title:       entry.GetAttributeValue("title"),
			Active:      true, // LDAP logic for active usually involves userAccountControl bitmask
		}
		res.Name.Formatted = entry.GetAttributeValue(c.config.NameAttr)
		res.Emails = []identity.Email{{Value: email, Primary: true}}
		
		// Map groups if memberOf is present
		groups := entry.GetAttributeValues("memberOf")
		for _, g := range groups {
			res.Groups = append(res.Groups, identity.Group{Value: g, Display: extractCN(g)})
		}

		users = append(users, res)
	}

	return users, nil
}

func extractCN(dn string) string {
	parts := strings.Split(dn, ",")
	if len(parts) > 0 {
		return strings.TrimPrefix(parts[0], "CN=")
	}
	return dn
}
