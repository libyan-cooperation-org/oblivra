package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ForbiddenImport defines a rule: package A must NOT import package B.
type ForbiddenImport struct {
	From string // package directory name (e.g. "detection")
	To   string // forbidden import substring (e.g. "vault")
	Why  string // human-readable reason
}

// Domain boundary rules — these MUST be enforced.
var forbiddenImports = []ForbiddenImport{
	{From: "detection", To: "/vault/", Why: "Detection layer must not depend on Vault layer"},
	{From: "detection", To: "/ssh/", Why: "Detection layer must not depend on SSH layer"},
	{From: "enrich", To: "/vault/", Why: "Enrichment must be read-only pipeline stage, no vault access"},
	{From: "enrich", To: "/ssh/", Why: "Enrichment must not depend on SSH layer"},
	{From: "ingest", To: "/vault/", Why: "Ingest pipeline must not access vault directly"},
	{From: "ingest", To: "/ssh/", Why: "Ingest pipeline must not depend on SSH layer"},
	{From: "search", To: "/vault/", Why: "Search engine must not access vault directly"},
	{From: "search", To: "/ssh/", Why: "Search engine must not depend on SSH layer"},
	{From: "integrity", To: "/ssh/", Why: "Integrity layer must not depend on SSH"},
	{From: "eventbus", To: "/vault/", Why: "Event bus must remain infrastructure-agnostic"},
	{From: "eventbus", To: "/database/", Why: "Event bus must remain infrastructure-agnostic"},
	{From: "logger", To: "/vault/", Why: "Logger must remain zero-dependency"},
	{From: "logger", To: "/database/", Why: "Logger must remain zero-dependency"},
	{From: "logger", To: "/ssh/", Why: "Logger must remain zero-dependency"},
	{From: "platform", To: "/vault/", Why: "Platform abstraction must not depend on vault"},
	{From: "platform", To: "/database/", Why: "Platform abstraction must not depend on database"},
}

func TestArchitecturalBoundaries(t *testing.T) {
	internalDir := findInternalDir(t)
	fset := token.NewFileSet()

	for _, rule := range forbiddenImports {
		pkgDir := filepath.Join(internalDir, rule.From)
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			continue // package doesn't exist yet, skip
		}

		violations := findViolations(t, fset, pkgDir, rule.To)
		for _, v := range violations {
			t.Errorf("ARCHITECTURE VIOLATION: %s imports %s\n"+
				"  File: %s\n"+
				"  Rule: %s\n"+
				"  Fix:  Use an interface from interfaces.go instead of a concrete import",
				rule.From, v.importPath, v.file, rule.Why)
		}
	}
}

func TestNoCrossLayerConcreteTypeLeaks(t *testing.T) {
	internalDir := findInternalDir(t)
	fset := token.NewFileSet()

	// app/ is the only package allowed to import concrete types from all layers
	// (it's the DI container). All other packages should use interfaces.
	allowedCrossLayer := map[string]bool{
		"app": true,
		"api": true, // API layer wires services
	}

	entries, err := os.ReadDir(internalDir)
	if err != nil {
		t.Fatalf("Failed to read internal dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || allowedCrossLayer[entry.Name()] {
			continue
		}

		pkgDir := filepath.Join(internalDir, entry.Name())
		imports := collectImports(t, fset, pkgDir)

		// Count how many other internal packages this package imports
		internalImports := 0
		for _, imp := range imports {
			if strings.Contains(imp, "oblivrashell/internal/") &&
				!strings.HasSuffix(imp, "/"+entry.Name()) {
				internalImports++
			}
		}

		// Warn if a leaf package imports too many internal packages (>5 = likely over-coupled)
		if internalImports > 5 {
			t.Logf("WARNING: %s imports %d internal packages — consider reducing coupling",
				entry.Name(), internalImports)
		}
	}
}

type violation struct {
	file       string
	importPath string
}

func findViolations(t *testing.T, fset *token.FileSet, pkgDir string, forbidden string) []violation {
	t.Helper()
	var violations []violation

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if strings.Contains(importPath, forbidden) {
				violations = append(violations, violation{
					file:       filePath,
					importPath: importPath,
				})
			}
		}
	}

	return violations
}

func collectImports(t *testing.T, fset *token.FileSet, pkgDir string) []string {
	t.Helper()
	seen := make(map[string]bool)

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			seen[importPath] = true
		}
	}

	result := make([]string, 0, len(seen))
	for imp := range seen {
		result = append(result, imp)
	}
	return result
}

func findInternalDir(t *testing.T) string {
	t.Helper()
	// Walk up from test file location to find internal/ directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Try current directory first, then walk up
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "internal")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}

	t.Fatal("Could not find internal/ directory")
	return ""
}
