package licensing

import "fmt"

// FeatureGateError is returned by RequireFeature when a feature is not available
// under the current license tier. The frontend uses RequiredTier to render the
// correct upgrade call-to-action.
type FeatureGateError struct {
	Feature      Feature
	CurrentTier  Tier
	RequiredTier Tier
}

func (e *FeatureGateError) Error() string {
	return fmt.Sprintf(
		"feature %q requires %s tier (current: %s) — upgrade your license to access this capability",
		e.Feature, e.RequiredTier, e.CurrentTier,
	)
}

// IsFeatureGateError reports whether err is a *FeatureGateError.
func IsFeatureGateError(err error) bool {
	_, ok := err.(*FeatureGateError)
	return ok
}
