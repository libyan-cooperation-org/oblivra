# OBLIVRA Frontend — Changelog

> Phase-tagged design-system + chrome changes. Living doc; new phases
> append at the top.

## Phase 32 — Operator Profiles + Decision Support (current)

### Design system
- New `--cr2 / --ok2 / --md2 / --hi2 / --pu2 / --ac2 / --s0` short
  aliases defined in `:root`. Previously referenced ~57 times across
  the app but never defined; severity colours silently fell back.
- 4-step type ramp via `--fs-micro / --fs-label / --fs-body / --fs-heading`.
  Body bumped from 11 px to 12 px; compact density mode preserved at 11/9.
- `--banner-h: 28px` standardises crisis banner / SystemBanner heights so
  routes don't shift 6 px.
- Crown-jewel asset toggle (operator-side localStorage; backend tag
  persistence in Phase 33).

### Chrome
- Sidebar collapsed from 8 groups → 5 (SIEM / Operations / Investigate
  / Govern / Admin). Persisted-state migration via `migrateLegacyGroup`.
- Sovereignty badge in TitleBar — single-glance posture indicator
  (on-prem · TPM · air-gap · KMS).
- Pivot crumb strip — operator's recent navigation, click-to-jump-back.
- Operator Profile wizard — first-run, 4 presets + custom.
- Tenant fast-switcher (⌘T) — gated on `tenantChrome=switcher-bar`.
- Crisis Decision Panel — sliding panel with 3 actions (seal /
  war-room / stand-down). Auto-lifts noise floor to critical-only on
  arm; restores on stand-down.
- Toast container split into two `aria-live` regions (polite +
  assertive); cap at 4 visible per region.
- Scrollbar grew from 4 px to 6 px (10 px on hover).
- Mobile breakpoint bumped from 640 → 768; new tablet band 768–1024
  shows icon-only sidebar.

### Functional additions
- Next-Best-Action recommender (rule-based, server-side at
  `/api/v1/alerts/recommend`).
- Inline IOC underlining in xterm output (1 px red bar; 6 s auto-dispose).
- Alert→shell reverse direction (Ctrl+Click in shell rail injects
  command + comment block).
- Suppress-as-FP one-keystroke (`x`) with 5 s undo + cross-device
  persistence via `/api/v1/alerts/{id}/suppress`.
- Bulk evidence seal (`/api/v1/evidence/seal`).
- Crisis-lifecycle audit endpoint (`/api/v1/crisis/state`).
- Daily queue digest (yesterday's still-open high+ alerts).
- Delta tile on dashboards (vs-yesterday counts).
- Single-source keyboard shortcuts registry (`lib/shortcuts.ts`);
  `KeyboardMap.svelte` auto-generates from it.

## Phase 31 — SOC redesign + investigation-first IA
- Tenant scope persisted to localStorage; API requests carry
  `X-Tenant-Id`.
- Time-range scope in TitleBar (`timeRangeStore`).
- Density toggle in Settings — comfortable (12 px) vs compact (11 px).
- `body { overflow: hidden }` scoped to `body.app-shell` so the web
  build can scroll the document naturally.

## Phase 30.4 — Multi-tenant + severity palette
- `--color-sev-{debug,info,warn,error,critical}` unified palette so
  every source's severity maps to one visual language.
- Severity-tinted backgrounds (`--color-sev-*-bg`) for chips/rails.

## Phase 24.2 — Arabic / RTL support
- Logical-property migration for sidebar borders + chips.
- `dir="rtl"` overrides for hard-coded `left/right` margins.
- xterm forced LTR inside RTL chrome (Arabic-locale ops sessions
  would otherwise render shell output mirrored).

## v1.1 — Tactical design baseline
- 5-level surface stack (`--color-surface-{0..4}`).
- `--font-ui` / `--font-mono` (IBM Plex).
- Ergonomic short aliases (`--bg`, `--tx`, `--ac`, etc.) — frozen,
  do not introduce new ones; new code uses `--color-*`.

---

For non-frontend changes (backend services, security, detection
engine), see the root `docs/` directory and `task.md`.
