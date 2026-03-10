package auth

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SAMLProvider wraps a SAML 2.0 Service Provider
type SAMLProvider struct {
	Name       string
	ProviderID string
	sp         *samlsp.Middleware
	log        *logger.Logger
}

// SAMLConfig holds parameters for SAML SP configuration
type SAMLConfig struct {
	ProviderID  string
	Name        string
	MetadataURL string
	EntityID    string // Usually the app's base URL
	ACSPath     string // e.g. /auth/saml/acs
	Certificate tls.Certificate
}

// SAMLIdentity represents the attributes extracted from a SAML assertion
type SAMLIdentity struct {
	NameID string
	Email  string
	Name   string
	Groups []string
}

// NewSAMLProvider initializes a SAML Service Provider using IdP metadata (URL or local File)
func NewSAMLProvider(ctx context.Context, cfg SAMLConfig, metadataPath string, log *logger.Logger) (*SAMLProvider, error) {
	rootURL, err := url.Parse(cfg.EntityID)
	if err != nil {
		return nil, fmt.Errorf("parse entity ID: %w", err)
	}

	var idpMetadata *saml.EntityDescriptor

	if metadataPath != "" && !strings.HasPrefix(metadataPath, "http") {
		// Load from local file
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("read local saml metadata: %w", err)
		}
		idpMetadata, err = samlsp.ParseMetadata(data)
		if err != nil {
			return nil, fmt.Errorf("parse local saml metadata: %w", err)
		}
	} else {
		// Fetch from remote URL
		idpMetadataURL, err := url.Parse(cfg.MetadataURL)
		if err != nil {
			return nil, fmt.Errorf("parse metadata URL: %w", err)
		}

		idpMetadata, err = samlsp.FetchMetadata(
			ctx,
			http.DefaultClient,
			*idpMetadataURL,
		)
		if err != nil {
			return nil, fmt.Errorf("fetch IdP metadata: %w", err)
		}
	}

	var privKey *rsa.PrivateKey
	if cfg.Certificate.PrivateKey != nil {
		privKey = cfg.Certificate.PrivateKey.(*rsa.PrivateKey)
	}

	var cert *x509.Certificate
	if len(cfg.Certificate.Certificate) > 0 {
		cert, _ = x509.ParseCertificate(cfg.Certificate.Certificate[0])
	}

	sp, err := samlsp.New(samlsp.Options{
		URL:               *rootURL,
		Key:               privKey,
		Certificate:       cert,
		IDPMetadata:       idpMetadata,
		AllowIDPInitiated: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create SAML SP: %w", err)
	}

	return &SAMLProvider{
		Name:       cfg.Name,
		ProviderID: cfg.ProviderID,
		sp:         sp,
		log:        log.WithPrefix("saml"),
	}, nil
}

// Middleware returns the SAML middleware for HTTP routing
func (p *SAMLProvider) Middleware() *samlsp.Middleware {
	return p.sp
}

// ExtractIdentity extracts SAML attributes from an authenticated session
func (p *SAMLProvider) ExtractIdentity(r *http.Request) (*SAMLIdentity, error) {
	session := samlsp.SessionFromContext(r.Context())
	if session == nil {
		return nil, fmt.Errorf("no SAML session found")
	}

	sa, ok := session.(samlsp.SessionWithAttributes)
	if !ok {
		return nil, fmt.Errorf("session does not contain attributes")
	}

	attrs := sa.GetAttributes()

	identity := &SAMLIdentity{
		Email: firstAttr(attrs, "email", "urn:oid:0.9.2342.19200300.100.1.3"),
		Name:  firstAttr(attrs, "displayName", "urn:oid:2.16.840.1.113730.3.1.241"),
	}

	if g := attrs.Get("groups"); g != "" {
		identity.Groups = []string{g}
	}

	p.log.Info("SAML login successful: %s", identity.Email)
	return identity, nil
}

// firstAttr retrieves the first matching SAML attribute by name
func firstAttr(attrs samlsp.Attributes, names ...string) string {
	for _, name := range names {
		if val := attrs.Get(name); val != "" {
			return val
		}
	}
	return ""
}
