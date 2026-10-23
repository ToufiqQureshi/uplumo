# Database schema

Goose migrations live under `migrations/`. Version 1 holds the tables the
product started with (users, sessions, organizations, projects, audits,
reports and the rest) plus the first analytics table, so a new database is
built by `Migrate` alone. Versions 2 onward add tables prefixed `go_`.

Never edit a migration that has run anywhere, and production has run all of
them: add a new file numbered after the highest one. Two files with the same
number stop the server from starting, so check `main` before picking a number
when several branches are open.

`testdata/legacy_schema.sql` is a trimmed copy of the legacy tables for Go
tests that run in a private schema. `database_test.go` checks that a fresh
database migrates end to end; keep every column in `legacy_schema.sql` present
in the migrations too.

Search growth uses migration 31: `go_search_growth_reports` holds one expiring
report cache per project, while `go_search_growth_actions` holds at most 100
tracked page baselines per project. Both cascade on project/account deletion;
reads/writes carry organization and project, and page uniqueness prevents
repeated saves from replacing baseline evidence.

Migrations 29/30 add campaign fields, session/event hashes, engagement storage
and indexes with defaults compatible with existing analytics writes. The schema
alone does not enable session tracking or change analytics ingestion. Preserve
applied migration contents and keep all prerequisite versions in order so startup
never encounters a missing lower version.

Migration 32 adds `go_search_growth_monitors`: one scoped opt-in configuration
and last in-app digest per project. A partial due-time index supports bounded
worker claims; a revision token fences concurrent/disabled updates. RLS is enabled
with no public-client policies; the trusted server/migration owner operates it.

Migration 34 applies the same RLS boundary to report caches and saved actions.
All three growth tables have no browser/Data API policies. The server database
role must own these tables or have BYPASSRLS; API queries still require explicit
organization/project predicates. `TestSearchGrowthStorageDeniesPublicRole`
verifies that table grants alone cannot expose or alter stored evidence.

Migration 35 adds optional analytics browser recognition and sanitized click destinations. It enables RLS and revokes public/client privileges on the browser registry and pageview/custom-event tables. The Go server must own these tables or have BYPASSRLS. See analytics `decision.md` and `AUDIENCE_DESTINATIONS.txt`; migration tests verify fresh startup, restricted-role denial and legacy-write compatibility.

Migration 41 adds pending-outcome and settlement metadata to credit reservations. It protects already-bought work during credit-transaction recovery and marks existing open feature holds conservatively; no plan price or grant changes.
