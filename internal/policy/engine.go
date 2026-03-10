package policy

import (
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Config represents the dynamic configuration for the policy engine
type Config struct {
	ActiveTier  Tier            `json:"active_tier"`
	CustomFlags map[string]bool `json:"custom_flags"` // Overrides for specific features
}

// Engine evaluates capabilities and features based on the active tier and configuration
type Engine struct {
	config Config
	log    *logger.Logger
}

// NewEngine initializes a new Policy Engine
func NewEngine(cfg Config, log *logger.Logger) *Engine {
	if cfg.ActiveTier == "" {
		cfg.ActiveTier = TierFree
	}
	if cfg.CustomFlags == nil {
		cfg.CustomFlags = make(map[string]bool)
	}

	return &Engine{
		config: cfg,
		log:    log,
	}
}

// Evaluate determines if a feature is accessible given the current constraints
func (e *Engine) Evaluate(feature string) bool {
	// 1. Check if there's an explicit override (beta flags, custom grants)
	if override, exists := e.config.CustomFlags[feature]; exists {
		return override
	}

	// 2. Look up the tier requirement for the feature
	requiredTier, exists := featureRequirements[feature]
	if !exists {
		// If it's not a known tiered feature, default to false safety
		e.log.Warn("[POLICY] Attempted to evaluate unknown feature: %s", feature)
		return false
	}

	// 3. Evaluate against active tier
	allowed := IsAtLeast(e.config.ActiveTier, requiredTier)
	if !allowed {
		e.log.Debug("[POLICY] Access Denied: %s requires %s (have %s)", feature, requiredTier, e.config.ActiveTier)
	}

	return allowed
}

// GetActiveTier returns the currently configured platform tier
func (e *Engine) GetActiveTier() string {
	return string(e.config.ActiveTier)
}

// GetCapabilitiesMatrix returns a map of all known features and whether they are currently enabled
func (e *Engine) GetCapabilitiesMatrix() map[string]bool {
	matrix := make(map[string]bool)
	for feature := range featureRequirements {
		matrix[feature] = e.Evaluate(feature)
	}

	// Add any custom flags that aren't in the standard requirements map
	for feature, enabled := range e.config.CustomFlags {
		if _, exists := matrix[feature]; !exists {
			matrix[feature] = enabled
		}
	}

	return matrix
}

// GetRawConfig returns the engine configuration
func (e *Engine) GetRawConfig() Config {
	return e.config
}

// SetTier updates the active tier at runtime (e.g. license activation)
func (e *Engine) SetTier(tier string) error {
	t := Tier(strings.ToLower(tier))
	if _, valid := tierHierarchy[t]; !valid {
		return fmt.Errorf("invalid tier: %s", tier)
	}

	e.config.ActiveTier = t
	e.log.Info("[POLICY] Platform tier updated to: %s", t)
	return nil
}

// ApplyOfflineBundle overwrites the active configuration with a verified offline policy bundle
func (e *Engine) ApplyOfflineBundle(config *Config) {
	if config == nil {
		return
	}
	e.config = *config
	e.log.Info("[POLICY] Applied verified offline policy bundle. Active Tier: %s", config.ActiveTier)
}
