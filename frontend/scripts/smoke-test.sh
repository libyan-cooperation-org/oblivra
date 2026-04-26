#!/usr/bin/env bash
#
# smoke-test.sh — frontend-bundle smoke test (Phase 29.7 follow-up).
#
# Validates that a freshly-built `dist/` is a) functional and b)
# contains the components the Wails exe will load. This is the
# closest we can get to a "boot the compiled exe and check it
# renders" test without adding a Playwright/headless-browser
# dependency that the air-gap commitment doesn't allow.
#
# What it catches:
#   - Build regressions (vite build fails) — primary defence.
#   - Tree-shake regressions (e.g. the v1.5.0 lucide-svelte blank
#     screen) — the bundle MUST contain known icon names.
#   - Component drop-outs — every Phase 30/31 component must show
#     up in the JS bundle by name.
#   - Severity token regressions — the CSS bundle must include
#     the unified `--color-sev-*` palette so HostDetail timeline
#     etc. still renders correctly.
#   - index.html sanity — must reference the built JS + CSS.
#
# What it does NOT catch (deliberately, to keep this lightweight):
#   - Live rendering errors (caught by lint-guards.sh + svelte-check)
#   - Wails-specific runtime errors (would need `task wails:smoke`)
#
# Run: bash frontend/scripts/smoke-test.sh
# Exit: 0 = all green, 1 = at least one assertion failed.

set -e

FRONTEND_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$FRONTEND_DIR/dist"
FAIL=0

# ── 1. Build ─────────────────────────────────────────────────────────
echo "▶ smoke: vite build"
cd "$FRONTEND_DIR"
if ! npx vite build > /tmp/oblivra-vite-build.log 2>&1; then
  echo "  ❌ FAIL: vite build returned non-zero"
  tail -20 /tmp/oblivra-vite-build.log | sed 's/^/    /'
  exit 1
fi
echo "  ✓ build succeeded"

# ── 2. dist/ structure ───────────────────────────────────────────────
echo "▶ smoke: dist/ structure"
if [ ! -f "$DIST/index.html" ]; then
  echo "  ❌ FAIL: dist/index.html missing after build"
  exit 1
fi
ASSETS_DIR="$DIST/assets"
if [ ! -d "$ASSETS_DIR" ]; then
  echo "  ❌ FAIL: dist/assets/ missing"
  exit 1
fi

# Locate the main bundle. Vite hash-suffixes filenames so we glob.
INDEX_JS=$(find "$ASSETS_DIR" -name 'index-*.js' | head -1)
INDEX_CSS=$(find "$ASSETS_DIR" -name 'index-*.css' | head -1)
if [ -z "$INDEX_JS" ]; then
  echo "  ❌ FAIL: no index-*.js bundle in dist/assets/"
  exit 1
fi
if [ -z "$INDEX_CSS" ]; then
  echo "  ❌ FAIL: no index-*.css bundle in dist/assets/"
  exit 1
fi
echo "  ✓ index.html + bundles present"

# index.html must reference the bundle — catches the case where
# the build outputs the bundle but doesn't link it.
INDEX_JS_BASE=$(basename "$INDEX_JS")
INDEX_CSS_BASE=$(basename "$INDEX_CSS")
if ! grep -q "$INDEX_JS_BASE" "$DIST/index.html"; then
  echo "  ❌ FAIL: index.html does not reference $INDEX_JS_BASE"
  FAIL=1
fi
if ! grep -q "$INDEX_CSS_BASE" "$DIST/index.html"; then
  echo "  ❌ FAIL: index.html does not reference $INDEX_CSS_BASE"
  FAIL=1
fi

# ── 3. Required components shipped in JS bundle ──────────────────────
# Vite minification mangles symbol names, but string literals (UI
# labels, CSS class names, error messages) survive intact. We probe
# for a stable, unique-per-component string from each component's
# template / script — if the component is fully tree-shaken, its
# strings are gone too.
echo "▶ smoke: required component fingerprints in JS bundle"
required_components=(
  # Phase 30 SOC chrome — unique CSS class names + UI labels
  "cr-rail"               # CommandRail (legacy nav still shipped)
  "dock-items"            # BottomDock items strip
  "ip-panel"              # InvestigationPanel root
  "entity-link"           # EntityLink primitive
  "trp-btn"               # TimeRangePicker button class
  "ts-trigger"            # TenantSwitcher trigger button
  "feed-entry"            # ActivityFeed row class
  # Phase 30/31 pages
  "what happened on this machine"   # HostDetail timeline subtitle
  "Mission Control"                  # Overview page title
  "PLATFORM RISK"                    # Overview risk-card label
  # Phase 30 store keys (localStorage)
  "oblivra:nav"           # navigationStore persistence key
  "oblivra:savedQueries"  # savedQueriesStore persistence key
)
missing_components=()
for c in "${required_components[@]}"; do
  # `grep -F -e` lets the pattern start with `-` and treats it as a
  # literal string (no regex special characters).
  if ! grep -F -q -e "$c" "$INDEX_JS"; then
    missing_components+=("$c")
  fi
done
if [ ${#missing_components[@]} -gt 0 ]; then
  echo "  ❌ FAIL: ${#missing_components[@]} required component(s) missing from bundle:"
  for c in "${missing_components[@]}"; do echo "      $c"; done
  FAIL=1
else
  echo "  ✓ all ${#required_components[@]} required components present"
fi

# ── 4. Lucide icons (defends against the v1.5.0 tree-shake regression) ─
# The original blank-screen was caused by `import * as LucideIcons` +
# string lookup; we now use explicit named imports + a static map.
# These icon class names MUST appear in the bundle.
echo "▶ smoke: critical lucide icons in bundle"
required_icons=(
  "LayoutDashboard"
  "Shield"
  "Network"
  "UserCog"
  "Server"
  "FileText"
  "Settings"
  "AlertTriangle"
  "ShieldAlert"
  "Activity"
)
missing_icons=()
for i in "${required_icons[@]}"; do
  if ! grep -F -q -e "$i" "$INDEX_JS"; then
    missing_icons+=("$i")
  fi
done
if [ ${#missing_icons[@]} -gt 0 ]; then
  echo "  ❌ FAIL: ${#missing_icons[@]} required icon(s) missing — possible tree-shake regression:"
  for i in "${missing_icons[@]}"; do echo "      $i"; done
  FAIL=1
else
  echo "  ✓ all ${#required_icons[@]} critical icons present"
fi

# ── 5. CSS tokens (severity palette must ship) ───────────────────────
echo "▶ smoke: severity color tokens in CSS bundle"
required_tokens=(
  "--color-sev-debug"
  "--color-sev-info"
  "--color-sev-warn"
  "--color-sev-error"
  "--color-sev-critical"
)
missing_tokens=()
for t in "${required_tokens[@]}"; do
  if ! grep -F -q -e "$t" "$INDEX_CSS"; then
    missing_tokens+=("$t")
  fi
done
if [ ${#missing_tokens[@]} -gt 0 ]; then
  echo "  ❌ FAIL: ${#missing_tokens[@]} severity token(s) missing from CSS:"
  for t in "${missing_tokens[@]}"; do echo "      $t"; done
  FAIL=1
else
  echo "  ✓ all ${#required_tokens[@]} severity tokens present"
fi

# ── 6. Bundle size sanity ────────────────────────────────────────────
# A wildly-undersized bundle indicates Vite tree-shook everything (the
# v1.5.0 failure mode would have produced a ~50KB bundle if every
# component was stripped). Floor + ceiling checks.
echo "▶ smoke: bundle size sanity"
JS_BYTES=$(wc -c < "$INDEX_JS")
JS_KB=$((JS_BYTES / 1024))
if [ "$JS_KB" -lt 200 ]; then
  echo "  ❌ FAIL: index.js is suspiciously small ($JS_KB KB) — possible tree-shake disaster"
  FAIL=1
elif [ "$JS_KB" -gt 2000 ]; then
  echo "  ⚠️  WARN: index.js is large ($JS_KB KB). No fail, but watch for regressions."
else
  echo "  ✓ index.js $JS_KB KB (within sane bounds)"
fi

# ── Done ─────────────────────────────────────────────────────────────
echo ""
if [ "$FAIL" -eq 0 ]; then
  echo "✅ smoke: all assertions passed — bundle is shippable"
  exit 0
else
  echo "💥 smoke: at least one assertion failed (see above)"
  exit 1
fi
