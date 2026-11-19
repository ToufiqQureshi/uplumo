# Public site

The server-rendered public pages: landing, pricing, product pages, company and
policy pages, and the blog. Pages are rendered once at start-up from Go
templates and served with ETags. No page makes a paid provider call.

## Code map

| File | Holds |
| --- | --- |
| `site.go` | `New`, the page list (`infoPages`), landing and pricing data, `robots.txt`, `sitemap.xml`, assets, the Organization/WebSite/SoftwareApplication JSON-LD |
| `answers.go` | Product-page questions (`*FAQ`), `plansWith`, BreadcrumbList and FAQPage JSON-LD, `llms.txt` |
| `blog.go` | Markdown posts in `blog/`, their BlogPosting/FAQPage JSON-LD and related guides |
| `templates/` | One HTML template per page; `base.html` holds the head, nav, FAQ block and footer |
| `static/` | Fonts, logo, social card, product screenshots, blog covers |

## Routes (registered in `httpapi`)

`/`, `/pricing`, every `InfoPaths()` page, `/blog`, `/blog/{slug}`,
`/robots.txt`, `/sitemap.xml`, `/llms.txt`, `/site/*` assets, and `/free-seo-audit` with its report fragment (`freeaudit.go`, see `internal/freeaudit`).

## Rules

- A product page lists its questions in `infoPages`; the page shows them and
  its FAQPage JSON-LD repeats exactly them.
- Plan names in answers come from `plansWith` (the plan catalog), never typed
  into the text.
- `llms.txt` shows dollar prices only after `ShowUSD`, the same rule as the
  landing and pricing pages.
