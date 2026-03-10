package architecture_test

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// AllowedDependencies defines the acceptable import paths for key domain packages.
// A package on the left map key is ONLY allowed to import the packages listed in the slice.
// If the slice contains "*", all internal imports are allowed (e.g., the `app` package).
var AllowedDependencies = map[string][]string{
	"github.com/kingknull/oblivrashell/internal/detection": {
		"github.com/kingknull/oblivrashell/internal/logger",
		"github.com/kingknull/oblivrashell/internal/eventbus", // Core infrastructural primitives typically allowed
		// Note: 'vault' is explicitly excluded. 'database' might also be excluded depending on strictness.
	},
}

// BannedDependencies defines strictly forbidden relationships.
// A package on the left cannot import any package listed on the right.
var BannedDependencies = map[string][]string{
	"github.com/kingknull/oblivrashell/internal/detection": {
		"github.com/kingknull/oblivrashell/internal/vault",
		"github.com/kingknull/oblivrashell/internal/database",
		"github.com/kingknull/oblivrashell/internal/app",
	},
	// Repositories shouldn't know about UI/App layer
	"github.com/kingknull/oblivrashell/internal/database": {
		"github.com/kingknull/oblivrashell/internal/app",
	},
	// Core primitives shouldn't know about domains
	"github.com/kingknull/oblivrashell/internal/logger": {
		"github.com/kingknull/oblivrashell/internal/app",
		"github.com/kingknull/oblivrashell/internal/detection",
		"github.com/kingknull/oblivrashell/internal/database",
	},
	"github.com/kingknull/oblivrashell/internal/eventbus": {
		"github.com/kingknull/oblivrashell/internal/app",
		"github.com/kingknull/oblivrashell/internal/detection",
		"github.com/kingknull/oblivrashell/internal/database",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports,
	}

	// Load all internal packages
	pkgs, err := packages.Load(cfg, "github.com/kingknull/oblivrashell/internal/...")
	if err != nil {
		t.Fatalf("Failed to load packages: %v", err)
	}

	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			// Only evaluate internal boundaries
			if !strings.HasPrefix(importPath, "github.com/kingknull/oblivrashell/internal") {
				continue
			}

			// 1. Check Banned Dependencies
			if bannedList, exists := BannedDependencies[pkg.PkgPath]; exists {
				for _, banned := range bannedList {
					if strings.HasPrefix(importPath, banned) {
						t.Errorf("Architecture Violation: '%s' is strictly forbidden from importing '%s'", pkg.PkgPath, importPath)
					}
				}
			}

			// 2. Check Explicit Allowed List (if defined for the package)
			if allowedList, exists := AllowedDependencies[pkg.PkgPath]; exists {
				allowed := false
				for _, a := range allowedList {
					if a == "*" || strings.HasPrefix(importPath, a) {
						allowed = true
						break
					}
				}
				if !allowed {
					t.Errorf("Architecture Violation: '%s' is restricted and not allowed to import '%s' without explicit permission.", pkg.PkgPath, importPath)
				}
			}
		}
	}
}
