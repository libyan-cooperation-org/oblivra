package enrich

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/database"
)

// AssetEnricher cross-references IPs against Sovereign Terminal's own internal managed inventory
type AssetEnricher struct {
	hostRepo database.HostStore
	sessRepo database.SessionStore
}

func NewAssetEnricher(hostRepo database.HostStore, sessRepo database.SessionStore) *AssetEnricher {
	return &AssetEnricher{
		hostRepo: hostRepo,
		sessRepo: sessRepo,
	}
}

func (a *AssetEnricher) Name() string {
	return "AssetTagging"
}

func (a *AssetEnricher) Enrich(event *database.HostEvent) error {
	// If the event came from an internal managed host, we can enrich the user or labels implicitly
	if event.HostID != "" {
		host, err := a.hostRepo.GetByID(context.Background(), event.HostID)
		if err == nil && host != nil {
			// If location is blank, maybe we know where the server actually is from its labels
			for _, tag := range host.Tags {
				if tag == "prod" {
					event.User = "Production Server" // Co-opting UI fields until SIEM schema is finalized
				}
			}
		}
	}

	// Maybe the Source IP is another Sovereign Terminal user jumping box-to-box?
	if event.SourceIP != "" && event.SourceIP != "127.0.0.1" {
		hosts, _ := a.hostRepo.GetAll(context.Background())
		for _, h := range hosts {
			if h.Hostname == event.SourceIP {
				event.User = "Internal Server (" + h.Label + ")"
				event.Location = "Sovereign Managed Asset"
				break
			}
		}
	}

	return nil
}
