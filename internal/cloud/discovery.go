package cloud

import (
	"context"


	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// CloudDiscoveryManager orchestrates the fetching of cloud assets from multiple providers.
type CloudDiscoveryManager struct {
	store     database.CloudAssetStore
	log       *logger.Logger
	providers []CloudProvider
}

// CloudProvider defines the interface for cloud service metadata ingestion.
type CloudProvider interface {
	Name() string
	FetchAssets(ctx context.Context) ([]database.CloudAsset, error)
}

func NewCloudDiscoveryManager(store database.CloudAssetStore, log *logger.Logger) *CloudDiscoveryManager {
	manager := &CloudDiscoveryManager{
		store: store,
		log:   log.WithPrefix("cloud_discovery"),
	}
	
	// Register Mock AWS Provider by default for Phase 16 demonstration
	manager.providers = append(manager.providers, &AWSMockProvider{})
	
	return manager
}

// DiscoverAll triggers asset fetching from all registered providers and persists them.
func (m *CloudDiscoveryManager) DiscoverAll(ctx context.Context) error {
	m.log.Info("Starting cloud asset discovery across %d providers...", len(m.providers))
	
	for _, p := range m.providers {
		assets, err := p.FetchAssets(ctx)
		if err != nil {
			m.log.Error("Provider %s failed: %v", p.Name(), err)
			continue
		}
		
		for _, asset := range assets {
			if err := m.store.Upsert(ctx, &asset); err != nil {
				m.log.Error("Failed to persist asset %s: %v", asset.ID, err)
			}
		}
		m.log.Info("Provider %s: discovered %d assets", p.Name(), len(assets))
	}
	
	return nil
}

// AWSMockProvider simulates AWS API responses for testing the CSPM framework.
type AWSMockProvider struct{}

func (p *AWSMockProvider) Name() string { return "aws" }

func (p *AWSMockProvider) FetchAssets(ctx context.Context) ([]database.CloudAsset, error) {
	// Simulate some EC2 instances and S3 buckets
	return []database.CloudAsset{
		{
			ID:        "i-0abcdef1234567890",
			Provider:  "aws",
			Region:    "us-east-1",
			AccountID: "123456789012",
			Type:      "ec2",
			Name:      "prod-api-server",
			Status:    "running",
			Metadata: map[string]string{
				"InstanceType": "t3.medium",
				"PublicIp":     "3.23.45.67",
				"PrivateIp":    "10.0.1.5",
			},
			Tags: map[string]string{
				"Environment": "Production",
				"CostCenter":  "SIEM-Operations",
			},
		},
		{
			ID:        "oblivra-siem-logs-2026",
			Provider:  "aws",
			Region:    "us-west-2",
			AccountID: "123456789012",
			Type:      "s3",
			Name:      "oblivra-siem-logs",
			Status:    "encrypted",
			Metadata: map[string]string{
				"Versioning": "Enabled",
				"Logging":    "Target(audit-bucket)",
			},
			Tags: map[string]string{
				"Sensitivity": "High",
				"DataOwner":    "Compliance",
			},
		},
	}, nil
}
