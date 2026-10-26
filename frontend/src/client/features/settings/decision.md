# Workspace and account settings: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Groups appearance, API-key, report-branding and crawler-access settings under a single settings experience.

## Decisions to preserve

- Settings is a navigation shell, not a shared authorization boundary; each backend feature enforces its own permissions. _Why:_ Explicit bounds prevent a large or abusive request from consuming shared memory, latency or service capacity.
- Plaintext API keys and crawler credentials must be shown only when created, then never returned by list endpoints. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- Do not combine unrelated settings saves into a mutation that can partially corrupt workspace configuration. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Visible header appearance control

- **Decision:** Put a labelled, keyboard-accessible System/Light/Dark menu in
  the authenticated app header on desktop and mobile. Reuse the existing
  theme hook and radio menu items rather than introducing another theme state.
- **Why:** Both palettes and persisted preferences already existed, but the
  choice was hidden in settings/account menus and the app appeared dark-only.
- **Preserve:** Existing storage key, startup initialization, system preference,
  cross-tab updates and storage-failure fallback. The always-mounted header
  consumer keeps root colors synchronized even when menus are closed. This is
  a browser appearance preference with no server mutation or provider call.
- **Trace:** `layout/AppShell.tsx`, `components/ThemePreferenceMenuItems.tsx`,
  `lib/theme.ts`; `tests/theme-preference.spec.ts` checks switching, persistence,
  keyboard/mobile access, shared settings state and system-color changes.

## Verification pointers

- Verify the listed source files and the feature route manually when behavior changes.
