# Operator Profiles

> Phase 32. Replaces the legacy "one UI fits everyone" model with a
> small, opinionated set of role-aligned configurations.

OBLIVRA serves four distinct operator personas — SOC Tier-1/2 analysts,
threat hunters / red-teamers, incident commanders, and MSP / platform
admins. Trying to optimise one UI for all four leaves every persona
slightly unhappy. Operator Profiles are the platform's answer:

> A bundled rule-set the operator picks once. Picking a profile flips
> nine settings together (home route, density, palette behaviour, vim
> leader keys, tenant chrome, crisis affordance, alert noise floor,
> layout mode, primary metric). Power users can fork into `custom` and
> override individual rules.

The first-run wizard asks one question. Settings → Operator Profile
shows the bundle and the override grid. That's the entire surface.

## The four shipped profiles

### `soc-analyst` — Maya
The triager. 8-hour shift, 40 open alerts, two monitors. Wants to
close as many alerts as possible without missing the one real one.

```
Home route          → /alert-management   (queue is the front door)
Density             → comfortable          (12 px body, 10 px micro)
Primary metric      → MTTR
Layout              → single screen
Palette front-door  → no                   (mouse-friendly)
Vim leader (g+letter) → no
Tenant chrome       → breadcrumb           (single tenant for the shift)
Crisis affordance   → banner               (compact)
Alert noise floor   → medium+              (suppress info / low)
```

### `threat-hunter` — Daniel
The proactive operator. Lives in `/siem-search` and `/shell`. Speed of
pivot matters more than visual polish. Runs tmux at home.

```
Home route          → /siem-search         (search box is the front door)
Density             → compact              (max information density)
Primary metric      → hunt yield
Layout              → single screen
Palette front-door  → yes                  (⌘K stays focused)
Vim leader          → yes                  (g+a → /alerts, g+s → /siem)
Tenant chrome       → breadcrumb
Crisis affordance   → banner
Alert noise floor   → all                  (every signal, including info)
```

### `incident-commander` — Rita
The crisis lead. Joins the war-room when something is on fire. Reads
the executive dash on her phone in the morning. Owns DSR turnaround.

```
Home route          → /war-mode            (crisis-led)
Density             → comfortable
Primary metric      → MTTR + evidence-latency
Layout              → war-room             (multi-monitor bias)
Palette front-door  → no
Vim leader          → no
Tenant chrome       → badge                (always-visible)
Crisis affordance   → fullscreen-takeover  (the whole screen turns red)
Alert noise floor   → critical-only        (no chatter during incidents)
```

### `msp-admin` — Multi-customer ops
MSP / platform admin operating across many customer tenants per shift.

```
Home route          → /admin
Density             → compact
Primary metric      → FP-rate
Layout              → single screen
Palette front-door  → yes
Vim leader          → yes
Tenant chrome       → switcher-bar         (Cmd+T fast-switcher)
Crisis affordance   → banner
Alert noise floor   → high+
```

### `custom`
Auto-selected the moment an operator overrides a single rule from a
preset. Their bundle is preserved in localStorage at
`oblivra:profileRules`; resetting via the "Reset to SOC Analyst preset"
button copies the SOC Analyst bundle and leaves the operator on
`custom` until they switch back to a named preset explicitly.

## How the rules wire through

The mechanism is intentionally small — one store, one DOM-attribute
push, a handful of consumers.

| Rule                | Reader                          |
|---------------------|---------------------------------|
| `homeRoute`         | `App.svelte` `onMount`          |
| `defaultDensity`    | `appStore.setDensity` → `body[data-density=*]` |
| `primaryMetric`     | dashboards (KPI tile emphasis)  |
| `layoutMode`        | `body[data-layout-mode=*]` (CSS) |
| `paletteFront`      | `App.svelte` keymap (future: focus pull) |
| `vimLeader`         | `App.svelte` `g+letter` handler  |
| `tenantChrome`      | `body[data-tenant-chrome=*]` + TenantFastSwitcher gate |
| `crisisAffordance`  | `CrisisDecisionPanel` self-gate |
| `alertNoiseFloor`   | `AlertManagement` filter pipe   |

Persistence: `localStorage` keys `oblivra:profile`,
`oblivra:profileRules`, `oblivra:profileChosen`.

## When to add a new profile

Don't, unless one of these is true:

1. A persona we haven't shipped for is being onboarded (e.g. a new
   "DLP analyst" workflow that shares no defaults with the four above).
2. The override grid is the wrong front door — i.e. there's a 9-rule
   combination that 30%+ of new users would pick by hand. Promote it
   to a preset.

Profiles are "smart defaults," not feature gates. Never use a profile
to *deny* an operator a feature they could otherwise reach. Locking
features behind tier sits in `internal/licensing/`, not here.

## Adding a new rule to the bundle

If you find yourself adding a 10th rule:

1. Add the field to `OperatorProfileRules` in
   `frontend/src/lib/stores/app.svelte.ts`.
2. Add the default to **all four** preset bundles in `PROFILE_PRESETS`.
3. Add the override widget to Settings → Operator Profile.
4. Wire one (and only one) reader. If two pages need the same rule,
   you've probably picked the wrong abstraction — extract a shared
   helper instead.
5. Document the rule in this file's table above.

## Anti-patterns

- **Profile-as-feature-flag.** If feature X needs to be off for SOC
  Analyst, off for Threat Hunter, on for everyone else, that's not a
  profile concern — it's a feature flag. Use the existing
  `licensing.Feature` / `appStore.featureEnabled` pattern.
- **Letting the wizard ask multi-question funnels.** One question. If
  your new profile needs four questions to pick correctly, the
  profile is too narrow.
- **Persisting profile rules on the server.** No. They live in
  localStorage. Cross-device profile sync is a separate concern that
  goes through user-prefs (Phase 33+) once we have a user-prefs store.

## Related code

- `frontend/src/lib/stores/app.svelte.ts` — types, presets, reducer
- `frontend/src/components/onboarding/ProfileWizard.svelte` — first-run
- `frontend/src/pages/Settings.svelte` — the override grid
- `frontend/src/components/ui/TenantFastSwitcher.svelte` — `tenantChrome=switcher-bar` gate
- `frontend/src/pages/AlertManagement.svelte` — `alertNoiseFloor` filter
- `frontend/src/components/layout/CrisisDecisionPanel.svelte` —
  `crisisAffordance` self-gate; auto-lifts noise floor on `crisis.arm()`
