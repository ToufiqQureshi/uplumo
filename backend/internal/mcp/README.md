# MCP server

The Go MCP handler owns the `/mcp` route for API-key and OAuth callers. It
authenticates existing `oseo_` API keys and Go-issued OAuth access tokens, and
serves its registered tools locally. OAuth client registration, authorization,
PKCE code exchange, refresh-token rotation, and resource discovery use the
Go-owned `go_mcp_oauth_*` tables. OAuth client registrations are valid for one
year, access tokens for one day, and refresh tokens for 30 days. Raw client
secrets, authorization codes, and tokens are never stored.

The Go MCP lane includes the account `whoami` tool and the `run_site_audit` tool.
Audit starts reuse Go's project authorization, plan/capacity checks, SSRF guard,
background job queue, and optional Lighthouse/rendering controls. `whoami`
reads the active organization's credit balance without charging credits. If
the balance cannot be read, the response reports it as unknown.

`get_cite_fix_report` also returns saved AI Visibility mentions, free bounded
rival-page comparisons and the latest critical/warning audit issues. The tool
does not start JavaScript rendering, refresh mentions or submit Google AI
Overview tasks; paid actions remain in their confirmed product flows.

Keep tool names and JSON schemas compatible with MCP clients. Tools without a
Go registration fail locally; there is no other backend to forward them to.
Tests: `oauth_test.go`, `registry_test.go` (plan gating for every tool),
`runner_test.go` and per-group tool tests (`project_context_tools_test.go`,
`rank_tracking_tools_test.go`, `report_tools_test.go`). Most data tools
(keywords, domain, backlinks, AI search) have no handler-level contract test.


The Go MCP registry also owns `get_backlinks_overview`. It reads the
organization-scoped Go backlinks service, applies the default spam filter to
the top referring-domain rows, and preserves the scope notes and provider
limitations from the legacy tool.


The Go MCP registry owns `get_backlinks_profile`, including bounded
pagination, sort mapping, spam filtering, grouping, and source/target filters.
The Go service applies the same target validation and filter budget as its HTTP
API.


The Go MCP registry owns `get_domain_overview`. It uses the Go
Domain service for organic estimates and the Go backlinks service for backlink
totals, while resolving country/language from the project when omitted.


The Go MCP registry owns `get_domain_keyword_suggestions`, using the
Go Domain service, project market defaults, and supported Labs location and
language validation.


The Go MCP registry also owns `get_ranked_keywords`. It uses the Go Domain service for scoped, filtered, market-specific ranked keyword rows and preserves the legacy sorting and pagination contract.


The Go MCP registry also owns `get_serp_results`. Each query resolves
its own market and optional canonical local location, while individual provider
errors are returned alongside successful query rows.


The Go MCP registry also owns `create_rank_tracker`. It creates an empty
configuration with the project's market/domain defaults, validates optional
canonical local locations, defaults to mobile/depth 10/manual, and starts no
provider spend at creation. Depth 10 is allowed on every paid tier; deeper
requests still pass the plan depth limit.


The Go MCP registry also owns `estimate_rank_tracker_cost`. It reads
the project-scoped tracker and calculates live and scheduled estimates without
starting a check or spending provider credits.


The Go MCP registry also owns `run_rank_tracker`. It requires an
explicit positive credit ceiling, rechecks the live estimate in the Go check
service, and queues the run through the shared background jobs path. Active runs
return their blocking run ID without creating another run.


The Go MCP registry also owns `get_keyword_metrics`. It resolves the project market, selects Labs or Google Ads, supports optional clickstream refinement and monthly trend omission, and preserves nullable metric fields.


The Go MCP registry also owns `research_keywords`. It runs each seed independently through the shared Go research service and preserves per-seed failures and result metadata.


The Go MCP registry also owns `inspect_urls`. It checks project authorization, uses the saved Search Console property, and returns per-URL inspection outcomes without running a live crawl.


The Go MCP registry also owns `find_serp_competitors`, using the Go domain provider's Labs lookup with market validation, domain exclusions, and the legacy sort choices.









`get_search_growth_report` authorizes the active project and calls GA4's shared
GSC/Google-organic growth analysis. Classified as `GoogleData`, it uses no paid
SEO provider or AI completion and preserves coverage warnings and nullable
metrics. SAM binds its project server-side and uses it in read-only analyst mode.

Search growth accepts `goalEvent`, `comparison` (`previous`/`year`) and explicit
`expanded` coverage. The shared service validates registered goal metrics before
reads and keys its cache by all three. Standard uses at most 5 report reads plus
1 goal metadata read; expanded at most 25 + metadata. SAM's draft includes the
selected goal/comparison so the analyst does not silently switch outcome scope.

Registry dispatch contains handler panics and missing results as explicit tool errors for both external MCP clients and SAM. Unexpected internal errors are logged server-side and returned with a generic message; intentional app/spend errors retain their safe guidance. Removed unused telemetry bookkeeping never wrote any usage; provider spending remains recorded by the shared spend guard.

Successful external calls persist the first connection/tool-call onboarding milestone. Internal SAM and failed tools do not advance it. Writes are bounded to two seconds, preserve the first timestamp and cannot make a successful tool fail.
