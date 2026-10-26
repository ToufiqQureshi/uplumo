# Workspace and account settings

Groups appearance, API-key, report-branding and crawler-access settings under a single settings experience.

Appearance is available in the authenticated app header as well as personal
settings and the account menu. `components/ThemePreferenceMenuItems.tsx` shares
the System/Light/Dark choices; `lib/theme.ts` persists the browser preference
and applies the existing app palette through the root `data-theme` attribute.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `ApiKeySettings.tsx`
- `BrandingSettings.tsx`
- `CrawlerAccessSettings.tsx`
- `backend/internal/apikeys`
- `backend/internal/branding`
- `backend/internal/crawleraccess`
