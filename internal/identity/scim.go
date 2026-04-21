package identity

import (
	"encoding/json"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
)

// SCIM Schemas
const (
	SchemaUser = "urn:ietf:params:scim:schemas:core:2.0:User"
	SchemaList = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
)

type PhoneNumber struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type Address struct {
	Type          string `json:"type,omitempty"`
	StreetAddress string `json:"streetAddress,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postalCode,omitempty"`
	Country       string `json:"country,omitempty"`
	Formatted     string `json:"formatted,omitempty"`
	Primary       bool   `json:"primary,omitempty"`
}

// UserResource represents a SCIM 2.0 User resource
type UserResource struct {
	Schemas    []string `json:"schemas"`
	ID         string   `json:"id"`
	ExternalID string   `json:"externalId,omitempty"`
	UserName   string   `json:"userName"`
	Name       struct {
		Formatted       string `json:"formatted,omitempty"`
		FamilyName      string `json:"familyName,omitempty"`
		GivenName       string `json:"givenName,omitempty"`
		MiddleName      string `json:"middleName,omitempty"`
		HonorificPrefix string `json:"honorificPrefix,omitempty"`
		HonorificSuffix string `json:"honorificSuffix,omitempty"`
	} `json:"name"`
	DisplayName       string        `json:"displayName,omitempty"`
	NickName          string        `json:"nickName,omitempty"`
	ProfileURL        string        `json:"profileUrl,omitempty"`
	Title             string        `json:"title,omitempty"`
	UserType          string        `json:"userType,omitempty"`
	PreferredLanguage string        `json:"preferredLanguage,omitempty"`
	Locale            string        `json:"locale,omitempty"`
	Timezone          string        `json:"timezone,omitempty"`
	Active            bool          `json:"active"`
	Department        string        `json:"department,omitempty"`
	Emails            []Email       `json:"emails,omitempty"`
	PhoneNumbers      []PhoneNumber `json:"phoneNumbers,omitempty"`
	Addresses         []Address     `json:"addresses,omitempty"`
	Groups            []Group       `json:"groups,omitempty"`
	Meta              Meta          `json:"meta"`
}

type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type Group struct {
	Value   string `json:"value"`
	Ref     string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
}

type Meta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
	Version      string `json:"version,omitempty"`
}

type ListResponse struct {
	Schemas      []string        `json:"schemas"`
	TotalResults int             `json:"totalResults"`
	StartIndex   int             `json:"startIndex"`
	ItemsPerPage int             `json:"itemsPerPage"`
	Resources    []*UserResource `json:"Resources"`
}

// ToUserResource converts a database.User to a SCIM UserResource
func ToUserResource(u *database.User, baseURL string) *UserResource {
	res := &UserResource{
		Schemas:    []string{SchemaUser},
		ID:         u.ID,
		ExternalID: u.ExternalID,
		UserName:   u.Email,
		DisplayName: u.DisplayName,
		Title:       u.Title,
		UserType:    u.UserType,
		PreferredLanguage: u.PreferredLanguage,
		Active:      u.Active,
		Meta: Meta{
			ResourceType: "User",
			Created:      u.CreatedAt,
			LastModified: u.UpdatedAt,
			Location:     fmt.Sprintf("%s/api/scim/v2/Users/%s", baseURL, u.ID),
		},
	}

	res.Name.Formatted = u.Name
	res.Emails = []Email{{Value: u.Email, Primary: true, Type: "work"}}
	
	if u.Phone != "" {
		res.PhoneNumbers = []PhoneNumber{{Value: u.Phone, Type: "work", Primary: true}}
	}
	if u.Address != "" {
		res.Addresses = []Address{{Formatted: u.Address, Type: "work", Primary: true}}
	}

	if u.GroupsJSON != "" {
		var groups []string
		if err := json.Unmarshal([]byte(u.GroupsJSON), &groups); err == nil {
			for _, g := range groups {
				res.Groups = append(res.Groups, Group{Value: g, Display: g})
			}
		}
	}

	return res
}

// FromUserResource maps a SCIM UserResource back to database.User fields
func FromUserResource(res *UserResource, u *database.User) {
	if res.ExternalID != "" {
		u.ExternalID = res.ExternalID
	}
	if res.UserName != "" {
		u.Email = res.UserName
	}
	if res.DisplayName != "" {
		u.DisplayName = res.DisplayName
	}
	if res.Name.Formatted != "" {
		u.Name = res.Name.Formatted
	} else if res.Name.GivenName != "" || res.Name.FamilyName != "" {
		u.Name = fmt.Sprintf("%s %s", res.Name.GivenName, res.Name.FamilyName)
	}
	
	u.Active = res.Active
	u.Title = res.Title
	u.UserType = res.UserType
	u.PreferredLanguage = res.PreferredLanguage

	// Phone mapping
	for _, p := range res.PhoneNumbers {
		if p.Primary || u.Phone == "" {
			u.Phone = p.Value
		}
	}
	// Address mapping
	for _, a := range res.Addresses {
		if a.Primary || u.Address == "" {
			if a.Formatted != "" {
				u.Address = a.Formatted
			} else {
				u.Address = fmt.Sprintf("%s, %s, %s %s, %s", a.StreetAddress, a.Locality, a.Region, a.PostalCode, a.Country)
			}
		}
	}

	if len(res.Groups) > 0 {
		var groupNames []string
		for _, g := range res.Groups {
			groupNames = append(groupNames, g.Display)
		}
		bytes, _ := json.Marshal(groupNames)
		u.GroupsJSON = string(bytes)
	}

	// Capture all other attributes as JSON for later normalization
	attrBytes, _ := json.Marshal(res)
	u.SCIMAttributes = string(attrBytes)
}

// Normalizer provides extensible mapping for identity attributes.
type Normalizer struct {
	Config map[string]string // Custom attribute mappings
}

func NewNormalizer(config map[string]string) *Normalizer {
	return &Normalizer{Config: config}
}

func (n *Normalizer) Normalize(res *UserResource, u *database.User) {
	FromUserResource(res, u)
	// Apply custom mappings from Config if needed
	// e.g. mapping a custom SCIM extension to a database field
}
