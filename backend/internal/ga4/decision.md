# Google Analytics and Search Growth: decisions

Read this before changing report calculations, goal selection, diagnosis,
baselines or weekly checks. `README.md` owns the current read bounds, deadlines,
thresholds and storage limits; `flow.md` traces execution.

## Measurement and reasoning

| Decision | Why | Implementation / verification |
| --- | --- | --- |
| Join final GSC pages to Google-organic GA4 landing pages, preserving source dates/time zones. | GSC measures Google search clicks; GA4 includes other acquisition sources unless filtered. Clicks and sessions remain different measurements even after matching. | `growth.go`, `search_opportunity.go`, `growth_test.go` |
| Discover a selected registered goal through property metadata; use its event-specific session rate and keep raw `keyEvents` separate. | Every business defines success differently. Filtering the entire report by event name would distort the session denominator; unrelated events must not inflate the chosen outcome. | `growth_goal.go`, `growth_business_test.go` |
| Label sessions × goal rate as an estimate used for minimum-sample checks. | The rate is directly reported by Google; the derived quantity is not a measured event count or CRM-qualified lead total. | `growth.go`, `growth_goal.go` |
| Validate finite metrics and minimum samples before findings; suppress business recommendations/alerts when reports are limited. Leave absent or ambiguous joins null. | Fabricated zeroes, combined incompatible URL variants and restricted rates can lead to harmful content/business decisions. | `growth.go`, `normalization.go`; limited-data mutation regression |
| Diagnose low visibility, ranking movement and CTR changes as investigation candidates. Read exact-page query windows and keep absent queries unknown. | Query mix, demand, devices and search-result changes can explain averages; the API cannot establish the reason an individual searcher did not click. | `growth_clicks.go`, `growth_business_test.go` |
| Use actual ordered tracked-event user funnels; keep alpha API and instrumentation limitations visible. | Independent aggregate event counts cannot reconstruct a journey or prove abandonment. User sequences also differ from landing-page session conversion. | `growth_funnel.go`, `growth_business_test.go` |
| Expose observed referral evidence and an instrumentation checklist rather than a false tracking-health guarantee. | Server-side report reads cannot verify browser tags, consent or cross-domain linker behavior. Unknown Direct traffic is not automatically AI traffic. | `growth_tracking.go` |

## Comparisons, scope and cost

| Decision | Why | Implementation / related decision |
| --- | --- | --- |
| Freeze page/goal/source baselines from an authorized cached server report; preserve the first applied timestamp. Scope every repository operation by organization and project. | Client metrics are untrusted; retries or later settings must not move the baseline or expose another customer's evidence. | `growth_repository.go`, `growth_test.go`, database `decision.md` |
| Wait for a complete post-change window and reject changed properties/time zones/currencies. Normalize differing durations with explicit daily averages. | Partial, differently scoped windows can imply an improvement that was never measured. | `growth_handler.go` |
| Offer a 364-day weekday-aligned comparison and an optional user-selected unchanged page. Keep adjusted movement descriptive. | Weekday alignment and comparison pages provide context; holidays, audience changes and a poor control choice still prevent causal claims. | `growth_goal.go`, `growth_handler.go`, `growth_business_test.go` |
| Default to bounded standard coverage; expanded coverage is explicit. Include goal/comparison/coverage and selected connections in cache identity. | More Google data must not create unbounded quota/memory use or reuse evidence from a different analysis. A full API page still does not establish complete GSC query coverage. | `growth.go`; `README.md` for current bounds |
| Cap per-page queries and competing-URL examples, retaining the returned-page count. | A shared query across many pages previously allowed quadratic evidence growth; expanded coverage must remain practical. | `growth.go`, competing-page bound regression |
| Deduplicate identical reads through the scoped cache and singleflight. Do not call a paid provider from these reports. | Duplicate clicks and parallel requests must not buy research or consume redundant reporting quota. Existing SAM replies retain their own metering. | `growth.go`, `service.go`, concurrent call-count regressions |

## Weekly checks and AI boundary

- **Choice:** weekly checks are explicitly enabled per project for a registered
  goal; recheck entitlement and property identity before reading. **Reason:** a
  background task must retain the customer's intended scope. **Preserve:** bounded
  standard reads, in-app digest only, no scheduled paid AI/provider or email.
- **Choice:** atomically claim due work, fence results by revision and back off on
  failure. Identical enable replays preserve the schedule. **Reason:** replicas,
  retries or a disable/change during work must not duplicate checks or publish a
  stale digest. **Code/tests:** `growth_monitor.go`, `growth_business_test.go`;
  current cadence and alert thresholds are defined in `README.md`.
- **Choice:** retain the last successful digest with its own dates when a later
  check fails. **Reason:** past evidence remains useful only if it cannot be
  mistaken for a new successful check.
- **Choice:** AI explanations use the existing explicit SAM Send and server-side
  read-only tool allowlist. **Reason:** opening a finding cannot authorize paid
  research or a site mutation. See SAM/MCP decision files for execution controls.

## Future changes

Update this file with the changed decision and its reason in the same commit.
Preserve tenant isolation, goal denominators, missing-data semantics, cost bounds
and failure/disable regressions unless the change explicitly revises and verifies
them. Automated fixtures do not replace real-account Google reconciliation.
