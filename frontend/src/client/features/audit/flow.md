# Site audit: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. User chooses a project and audit scope in `LaunchView` / `useLaunchController`.
2. `auditApi.ts` submits the start request and then the UI reads status/progress/results while the job runs.
3. Go `audit` validates URLs and limits, queues a job, discovers pages from robots/sitemaps, crawls bounded pages and persists checks.
4. The results UI filters issues and pages; Lighthouse panels read saved PageSpeed/CrUX results when available.

## Implementation map

- `launch/useLaunchController.ts`
- `auditApi.ts`
- `AuditDetail.tsx`
- `backend/internal/audit/handler.go`
- `backend/internal/audit/runner.go`

## API and service dependencies

Go package: `backend/internal/audit`; asynchronous execution uses the jobs queue; optional PSI/CrUX calls are bounded and quota-limited.

## SEO change guard

On the audit history screen, the user explicitly starts a comparison. `launch/AuditChangeGuard.tsx` chooses the two latest completed audits with the same start URL and then fetches their saved result payloads through `auditApi.ts`; no comparison request is made before the click. `compare.ts` compares title, meta description, canonical URL, indexability and HTTP status for URLs present in both crawls, and counts newly seen URLs, URLs missing from the latest crawl and issue-row delta. The view caps displayed field changes. A URL missing from a crawl is only “missing,” not proof the customer deleted it; the user should review crawl coverage.

## Orphan-page link suggestions

For orphan-page issues, `results/IssuesView.tsx` displays up to 100 orphan rows. Suggestions are computed locally from the saved audit pages only after the user expands an orphan. `orphanLinkCandidates.ts` excludes the orphan itself, known orphans, non-indexable pages and pages whose fetch failed; it ranks remaining pages by shared title/path terms and internal-link count and returns at most three. The copy action escapes the target title and URL into an HTML link snippet. No provider call or customer-site change happens automatically.

## Cost, security and failure boundaries

- Audit start can consume shared crawl capacity and Google quota; keep per-plan limits, per-organization quotas and timeouts.
- Every crawl URL must pass the SSRF guard; never let redirect targets bypass it.
- Lighthouse/CrUX unavailability is a partial result, not proof that a page has no field data.
- At-least-once jobs must remain idempotent so retries do not duplicate audit rows.
- JavaScript rendering is an explicit launch opt-in, available only when the
  server advertises a wired DataForSEO renderer. The UI shows “Render JavaScript
  pages (at most 39 credits)” and passes the flag only after the user checks it.
  The backend renders at most 20 app-shell URLs, charges actual successful
  provider cost within that ceiling and charges zero for an unusable render.
  No automatic retry occurs; normal audit reads/polls do not buy rendering.

Rendered HTML runs through the regular page analyzer; PSI runs separately on
Google. Without the renderer, the opt-in stays disabled and no price is shown.
