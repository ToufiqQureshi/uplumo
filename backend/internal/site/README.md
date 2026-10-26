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
The link test validates known local targets and approved external hosts, not
live external responses or factual support. Follow `blog/README.md` for intent,
primary-source review, labelled examples and substantive-update rules.

`static/analytics.js` loads the owner's GA4 and first-party Uplumo tags on
production public pages and the React shell. It respects DNT/GPC, excludes
authentication routes, and records named signup/pricing/plan-selection clicks.
See `decision.md` for the measurement boundary and verification.
