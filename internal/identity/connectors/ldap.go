package connectors

import (
	"context"
	"crypto/tls"
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
	UseTLS       bool   `json:"use_tls"`
	InsecureSkip bool   `json:"insecure_skip"`
	PageSize     uint32 `json:"page_size"`
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
	if cfg.PageSize == 0 {
		cfg.PageSize = 500
	}

	return &LDAPConnector{id: id, config: cfg}, nil
}

func (c *LDAPConnector) ID() string   { return c.id }
func (c *LDAPConnector) Type() string { return "ldap" }

func (c *LDAPConnector) dial() (*ldap.Conn, error) {
	l, err := ldap.DialURL(c.config.URL)
	if err != nil {
		return nil, err
	}

	if c.config.UseTLS {
		if err := l.StartTLS(&tls.Config{InsecureSkipVerify: c.config.InsecureSkip}); err != nil {
			l.Close()
			return nil, err
		}
	}

	if err := l.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		l.Close()
		return nil, err
	}
	return l, nil
}

func (c *LDAPConnector) Verify(ctx context.Context) error {
	l, err := c.dial()
	if err != nil {
		return err
	}
	defer l.Close()
	return nil
}

func (c *LDAPConnector) FetchUsers(ctx context.Context) ([]*identity.UserResource, error) {
	l, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer l.Close()

	searchRequest := ldap.NewSearchRequest(
		c.config.UserBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		c.config.UserFilter,
		[]string{"dn", c.config.EmailAttr, c.config.NameAttr, "title", "department", "company", "sAMAccountName", "userPrincipalName", "userAccountControl", "telephoneNumber", "streetAddress", "l", "st", "co"},
		nil,
	)

	// Use paged search for reliability in large environments
	pagingControl := ldap.NewControlPaging(c.config.PageSize)
	var users []*identity.UserResource

	for {
		searchRequest.Controls = []ldap.Control{pagingControl}
		sr, err := l.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("ldap search error: %w", err)
		}

		for _, entry := range sr.Entries {
			email := entry.GetAttributeValue(c.config.EmailAttr)
			if email == "" {
				email = entry.GetAttributeValue("userPrincipalName")
			}
			if email == "" {
				continue
			}

			// AD specific: check if account is disabled (bit 2 of userAccountControl)
			active := true
			if uac := entry.GetAttributeValue("userAccountControl"); uac != "" {
				var uacInt int
				fmt.Sscanf(uac, "%d", &uacInt)
				if uacInt&2 != 0 {
					active = false
				}
			}

			res := &identity.UserResource{
				ID:          entry.DN,
				ExternalID:  entry.DN,
				UserName:    email,
				DisplayName: entry.GetAttributeValue(c.config.NameAttr),
				Title:       entry.GetAttributeValue("title"),
				Active:      active,
				Department:  entry.GetAttributeValue("department"),
			}
			res.Name.Formatted = entry.GetAttributeValue(c.config.NameAttr)
			res.Emails = []identity.Email{{Value: email, Primary: true}}
			
			// Map Phone
			if phone := entry.GetAttributeValue("telephoneNumber"); phone != "" {
				res.PhoneNumbers = append(res.PhoneNumbers, identity.PhoneNumber{Value: phone, Type: "work", Primary: true})
			}

			// Map Address
			addr := identity.Address{
				StreetAddress: entry.GetAttributeValue("streetAddress"),
				Locality:      entry.GetAttributeValue("l"),
				Region:        entry.GetAttributeValue("st"),
				Country:       entry.GetAttributeValue("co"),
				Type:          "work",
				Primary:       true,
			}
			if addr.StreetAddress != "" || addr.Locality != "" {
				res.Addresses = append(res.Addresses, addr)
			}

			groups := entry.GetAttributeValues("memberOf")
			for _, g := range groups {
				res.Groups = append(res.Groups, identity.Group{Value: g, Display: extractCN(g)})
			}

			users = append(users, res)
		}

		// Check for next page
		updatedControl := ldap.FindControl(sr.Controls, ldap.ControlTypePaging)
		if updatedControl == nil {
			break
		}
		if cookie := updatedControl.(*ldap.ControlPaging).Cookie; len(cookie) > 0 {
			pagingControl.SetCookie(cookie)
		} else {
			break
		}
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
