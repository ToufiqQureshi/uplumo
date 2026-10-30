# Public website, pricing and blog

Serves Uplumo marketing pages, pricing, blog posts, robots.txt, sitemap.xml and static assets from Go.

This is an internal developer/AI-agent guide, not customer-facing product copy. Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for cost, data and security invariants.

## Source map

- `site.go`
- `blog.go`

## API/trigger

GET /, /pricing, /blog, /blog/{slug}, /robots.txt, /sitemap.xml, assets

Blog sources must use reviewed HTTPS documentation hosts accepted by
`TestPagesHaveNoDeadLinks`; Google crawling/indexing explanations link to the
Search Central fundamentals documentation.
Product comparisons may also cite the reviewed official Semrush, Ahrefs and
Ubersuggest hosts in that exact allowlist. Comparison claims must follow the
scope and evidence rules in `decision.md`; a host check alone does not verify
the claim or live destination.

`static/analytics.js` loads the owner's GA4 and first-party Uplumo tags on
production public pages and the React shell. It respects DNT/GPC, excludes
authentication routes, and records named signup/pricing/plan-selection clicks.
See `decision.md` for the measurement boundary and verification.

## Public appearance

Every public page uses the shared header Color theme selector: System, Light or
Dark. `static/theme.js` loads before CSS and shares `theme-preference` with the
React app. The light palette is in `templates/base.html`; saved picks survive
page navigation and reload, and System follows OS changes. No API calls or
account are needed. A blocked storage write keeps the pick for this document.

## Buyer-facing content

`templates/landing.html` explains evidence, Cite Fix, editing and measurement;
`about.html` explains the owner's supplied experience without a personal name.
`pricing.html` keeps all numeric offers catalog-driven and explains shared
credits. Legal templates describe research inputs, shared reports, publishing
and activation prerequisites without changing billing rules. Keep sampled AI
answers separate from consumer-app coverage and recognised referral visits.
See `decision.md` for capability evidence and the public anonymity boundary.

## Blog publication state

The owner removed every article on 30 October 2026. There are no published
sources or blog-only assets. Former article URLs and the empty `/blog` index
return 404, without redirects. Navigation and sitemap contain no blog links.
The generic Markdown/cover renderer remains covered by fixture-based tests;
shared product screenshots are retained. See `blog/README.md` and the removal
regression in `blog_test.go` before considering new publication.
