# Public site

`site.go` embeds and renders public templates at startup. `InfoPaths()` registers information routes with the HTTP router and includes them in the sitemap. `blog.go` loads published Markdown posts.

Careers: `/careers` uses `templates/careers.html`; the shared company footer links to it. Three owner-requested roles have native details panels and role-specific email application links. Tests in `site_test.go` cover routing, metadata, links, sitemap and application subjects.
