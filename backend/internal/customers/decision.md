# Customers (staff): decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Gives the owner one place to see who signed up and what they told onboarding, and to export it for marketing.

## Decisions to preserve

- Access is the verified-email `STAFF_EMAILS` allowlist, the same rule billing uses for the unlimited staff workspace. _Why:_ this is every customer's email and business answers; workspace role alone must not unlock it, and an unverified sign-up with a staff address must not either.
- Read-only, one query, no pagination yet. _Why:_ a few thousand users fit one response; add paging when the table is slow, not before.
- CSV cells starting with `= + - @`, tab or CR get a leading `'`. _Why:_ names and "Other" onboarding answers are user-typed and would otherwise run as formulas in Excel/Sheets (CSV injection).
- The `Default` placeholder is not counted as a website, matching plan limits in `../projects/decision.md`.
- Marketing use of this export must respect consent and unsubscribes in the email tool (Loops); this page does not track marketing consent.

## Dependencies

Postgres `user`, `member`, `organization`, `projects`, `gsc_connections`, `ga4_connections`, `go_billing_subscriptions`, `user_onboarding_answers`.

## Verification

`customers_test.go` against `TEST_DATABASE_URL`: non-staff and unverified staff get 403, staff see onboarding, plan and sites, CSV neutralises formulas.
