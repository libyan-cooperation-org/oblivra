package policy

// Tier represents a product tier for capability isolation
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
	TierSovereign  Tier = "sovereign"
)

// Feature flag identifiers
const (
	FeatureSIEM       = "feature.siem"
	FeatureGovernance = "feature.governance"
	FeatureAirGap     = "feature.airgap"
	FeatureMAC        = "feature.mac" // Mandatory Access Control
	FeatureAdvancedML = "feature.ml"
)

// tierHierarchy defines the implicit hierarchy of tiers
var tierHierarchy = map[Tier]int{
	TierFree:       0,
	TierPro:        1,
	TierEnterprise: 2,
	TierSovereign:  3,
}

// featureRequirements maps each feature to its minimum required tier
var featureRequirements = map[string]Tier{
	FeatureSIEM:       TierPro,
	FeatureGovernance: TierEnterprise,
	FeatureAdvancedML: TierEnterprise,
	FeatureAirGap:     TierSovereign,
	FeatureMAC:        TierSovereign,
}

// IsAtLeast returns true if 'current' is greater than or equal to 'required'
func IsAtLeast(current Tier, required Tier) bool {
	return tierHierarchy[current] >= tierHierarchy[required]
}
