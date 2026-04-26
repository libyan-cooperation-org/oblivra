#!/usr/bin/env bash
#
# lint-guards.sh — frontend regression-prevention guards (Phase 29.x)
#
# Runs three grep-based checks that would have caught the four
# blank-screen regressions encountered on 2026-04-25:
#
#   1. Runes ($state / $derived / $effect) outside .svelte.ts files
#      → triggered the v1.5.0 i18n/index.ts blank-screen regression.
#
#   2. `import * as` from tree-shakeable icon libraries
#      → triggered the v1.5.0 BottomDock lucide-svelte regression.
#
#   3. Components that call t(...) in their template without importing
#      it from @lib/i18n → triggered the v1.4.0 PopOutButton blank-screen.
#
# Why bash + grep instead of ESLint:
#   - Zero new npm dependencies (the project commits to a small
#     air-gap-deployable surface area; ESLint pulls in 100+ packages).
#   - Zero false-positive risk on auto-fix (ESLint's auto-fix on
#     unused-import warnings was the original Phase 29 disaster).
#   - Runs in <1 second on the whole frontend tree.
#
# Run via: bash frontend/scripts/lint-guards.sh
# Add to CI: pre-build step in .github/workflows/ci.yml.
#
# Exit code: 0 if clean, 1 if any check fails.

set -e

SRC_DIR="$(cd "$(dirname "$0")/.." && pwd)/src"
FAIL=0

echo "▶ lint-guards: scanning ${SRC_DIR}"
echo ""

# ── Guard 1: runes in plain .ts files ────────────────────────────
# Svelte 5 runtime hard-rejects $state / $derived / $effect in any
# file that isn't .svelte, .svelte.ts, or .svelte.js. Catches both
# top-level declarations and class-field initializers.
echo "  [1/3] Runes outside .svelte / .svelte.ts ..."
RUNE_HITS=$(
  # Match actual rune INVOCATIONS, not just word mentions in comments.
  # Real rune uses always have one of: $state(, $state<, $derived(,
  # $derived., $effect(. A comment that says "we use $state" won't match.
  grep -rln -E '(\$state[(<]|\$derived[(.]|\$effect\()' \
    --include='*.ts' --include='*.js' \
    "$SRC_DIR" 2>/dev/null \
  | grep -v '\.svelte\.ts$' \
  | grep -v '\.svelte\.js$' \
  | grep -v '\.d\.ts$' \
  || true
)
if [ -n "$RUNE_HITS" ]; then
  echo "  ❌ FAIL: rune used in non-.svelte.ts files:"
  echo "$RUNE_HITS" | sed 's/^/      /'
  echo ""
  echo "      Fix: rename the file to *.svelte.ts OR move the rune"
  echo "      declaration into a .svelte.ts module and re-export."
  FAIL=1
else
  echo "      ✓ no runes in plain .ts files"
fi

# ── Guard 2: import * as from tree-shaken icon libs ───────────────
# Vite's prod tree-shaker strips namespace exports that are only
# accessed via property lookup. `import * as LucideIcons` + runtime
# string indexing means the icons get stripped → blank screen.
echo "  [2/3] import * as from lucide-svelte / @radix-ui ..."
NS_HITS=$(
  grep -rEn 'import \* as [A-Z][A-Za-z0-9_]* from ['"'"'"](lucide-svelte|@radix-ui/)' \
    --include='*.ts' --include='*.svelte' \
    "$SRC_DIR" 2>/dev/null \
  | grep -v '\.d\.ts:' \
  || true
)
if [ -n "$NS_HITS" ]; then
  echo "  ❌ FAIL: namespace import from tree-shakeable icon lib:"
  echo "$NS_HITS" | sed 's/^/      /'
  echo ""
  echo "      Fix: replace with explicit named imports + a static map."
  echo "      See BottomDock.svelte for the canonical pattern."
  FAIL=1
else
  echo "      ✓ no import * as from icon libs"
fi

# ── Guard 3: t() called in template without import ───────────────
# Catches the v1.4.0 PopOutButton failure mode. Heuristic: a Svelte
# file that uses `t('...')` or `t(\`...\`)` should also have
# `from '@lib/i18n'` somewhere in the import block. False-positive
# rate on this is acceptable because it's strictly subtractive — if
# you call t(...) in a template, you need to import t.
echo "  [3/3] Missing t() import where t() is used ..."
T_FAIL_FILES=()
while IFS= read -r f; do
  # Skip if file imports from @lib/i18n at all.
  if grep -q "from '@lib/i18n'" "$f" 2>/dev/null; then
    continue
  fi
  # Otherwise, did this file actually call t(...) ?
  # We anchor on whitespace/punctuation + t( to avoid matching e.g.
  # `text(...)`, `setTimeout(...)`, etc.
  if grep -qE '(^|[ \t({}\\<\\>=,;\"]|\$\{)t\(' "$f" 2>/dev/null; then
    T_FAIL_FILES+=("$f")
  fi
done < <(find "$SRC_DIR" -name '*.svelte' -type f)

if [ ${#T_FAIL_FILES[@]} -gt 0 ]; then
  echo "  ❌ FAIL: t() called without importing from @lib/i18n:"
  for f in "${T_FAIL_FILES[@]}"; do
    echo "      $f"
  done
  echo ""
  echo "      Fix: add 'import { t } from \"@lib/i18n\";' to each."
  FAIL=1
else
  echo "      ✓ no missing t() imports"
fi

echo ""
if [ $FAIL -eq 0 ]; then
  echo "✅ lint-guards: all checks passed"
  exit 0
else
  echo "💥 lint-guards: at least one check failed (see above)"
  exit 1
fi
