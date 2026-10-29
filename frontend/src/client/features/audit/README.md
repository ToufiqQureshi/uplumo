# Site audit

Starts an asynchronous crawl and presents technical SEO findings, crawl progress and optional performance data.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `launch/useLaunchController.ts`
- `auditApi.ts`
- `AuditDetail.tsx`
- `backend/internal/audit/handler.go`
- `backend/internal/audit/runner.go`

- `launch/AuditChangeGuard.tsx` and `compare.ts` (read-only saved-audit comparison)
- `orphanLinkCandidates.ts` (local, review-only link suggestions)

Audits inspect initial HTTP HTML, then can optionally render only pages the
crawler flagged as app shells through DataForSEO On-Page JavaScript. The launch
control appears only when backend capabilities confirm the renderer is wired.
It is opt-in, capped at 20 pages, and shows the 39-credit maximum (actual
delivered cost is charged). See [flow.md](flow.md) for the paid-call boundary.

Results show the added technical and external-link findings. Queued or partial
external checks are identified; old audits without current signals ask for a
new scan. Training-bot blocks are informational and do not reduce readiness.
