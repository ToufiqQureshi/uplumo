package site

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
)

// front builds a post's front matter from key: value lines.
func front(lines ...string) string {
	return "---\n" + strings.Join(lines, "\n") + "\n---\n"
}

const (
	goodTitle       = "title: What is AEO? Answer engine optimisation explained"
	goodDescription = "description: How to get your pages quoted by ChatGPT and Google AI Overviews, in five clear steps."
	goodDate        = "date: 2026-10-20"
)

func getPost(s *Site, slug string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/blog/"+slug, nil)
	req.SetPathValue("slug", slug)
	rec := httptest.NewRecorder()
	s.ServeBlogPost(rec, req)
	return rec
}

func TestParsePostRejectsWhatWouldHurtThePage(t *testing.T) {
	cases := map[string]struct{ slug, source string }{
		"bad file name":         {"What_Is", front(goodTitle, goodDescription, goodDate) + "Hi"},
		"no front matter":       {"post", "Hi"},
		"unclosed front matter": {"post", "---\n" + goodTitle + "\nHi"},
		"line without colon":    {"post", front("title", goodDescription, goodDate) + "Hi"},
		"no title":              {"post", front(goodDescription, goodDate) + "Hi"},
		"title too long":        {"post", front("title: "+strings.Repeat("x", 71), goodDescription, goodDate) + "Hi"},
		"description too short": {"post", front(goodTitle, "description: Short.", goodDate) + "Hi"},
		"description too long":  {"post", front(goodTitle, "description: "+strings.Repeat("x", 161), goodDate) + "Hi"},
		"bad date":              {"post", front(goodTitle, goodDescription, "date: 20/10/2026") + "Hi"},
		"bad updated":           {"post", front(goodTitle, goodDescription, goodDate, "updated: soon") + "Hi"},
		"empty body":            {"post", front(goodTitle, goodDescription, goodDate) + "  \n"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := parsePost(tc.slug, []byte(tc.source)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParsePost(t *testing.T) {
	body := "Answer first.\n\n## Why it matters\n\n<script>alert(1)</script>\n\n" + strings.Repeat("word ", 450)
	source := strings.ReplaceAll(front(goodTitle, goodDescription, goodDate)+body, "\n", "\r\n")
	post, draft, err := parsePost("what-is-aeo", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if draft || post.Author != "Uplumo team" || post.ReadMinutes != 3 || !post.Updated.Equal(post.Published) {
		t.Fatalf("post = %+v draft %v", post, draft)
	}
	if !strings.Contains(string(post.Body), `<h2 id="why-it-matters">`) {
		t.Error("headings need ids so a section can be linked")
	}
	if strings.Contains(string(post.Body), "<script>") {
		t.Error("raw HTML must be dropped")
	}
}

func TestBlogPages(t *testing.T) {
	files := fstest.MapFS{
		"README.md":      {Data: []byte("# How to write posts")},
		"older.md":       {Data: []byte(front(goodTitle, goodDescription, "date: 2026-10-01") + "Old post.")},
		"what-is-aeo.md": {Data: []byte(front(goodTitle, goodDescription, goodDate, "updated: 2026-10-22", "author: Asha Rao") + "Answer first.")},
		"hidden.md":      {Data: []byte(front(goodTitle, goodDescription, goodDate, "draft: true") + "Not yet.")},
	}
	s, err := build(&url.URL{Scheme: "https", Host: "uplumo.com"}, files)
	if err != nil {
		t.Fatal(err)
	}

	index := get(s.ServeBlog, "/blog", nil).Body.String()
	if strings.Index(index, "/blog/what-is-aeo") > strings.Index(index, "/blog/older") || strings.Contains(index, "/blog/hidden") {
		t.Error("the index lists published posts newest first, without drafts")
	}

	page := getPost(s, "what-is-aeo").Body.String()
	for _, want := range []string{
		`<link rel="canonical" href="https://uplumo.com/blog/what-is-aeo">`,
		`<meta property="og:type" content="article">`,
		`"@type":"BlogPosting"`, `"dateModified":"2026-10-22"`, `"@type":"Person"`,
		"What is AEO? Answer engine optimisation explained | Uplumo",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("post page lacks %s", want)
		}
	}
	for _, slug := range []string{"hidden", "README", "nope"} {
		if code := getPost(s, slug).Code; code != http.StatusNotFound {
			t.Errorf("/blog/%s: status %d, want 404", slug, code)
		}
	}

	var doc struct {
		URLs []struct {
			Loc     string `xml:"loc"`
			LastMod string `xml:"lastmod"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(get(s.ServeSitemap, "/sitemap.xml", nil).Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	lastmod := map[string]string{}
	for _, u := range doc.URLs {
		lastmod[u.Loc] = u.LastMod
	}
	if lastmod["https://uplumo.com/blog/what-is-aeo"] != "2026-10-22" || lastmod["https://uplumo.com/blog/older"] != "2026-10-01" {
		t.Errorf("sitemap posts = %v", lastmod)
	}
	if _, ok := lastmod["https://uplumo.com/blog/hidden"]; ok {
		t.Error("a draft must not be in the sitemap")
	}
}

func TestABrokenPostStopsTheSite(t *testing.T) {
	files := fstest.MapFS{"bad.md": {Data: []byte(front("title: x") + "Hi")}}
	if _, err := build(&url.URL{Scheme: "https", Host: "uplumo.com"}, files); err == nil || !strings.Contains(err.Error(), "bad.md") {
		t.Fatalf("err = %v, want the broken file named", err)
	}
}

func TestBlogCoverValidation(t *testing.T) {
	for _, image := range []string{"https://example.com/image.svg", "/site/blog/../private.svg", `/site/blog/evil.svg" onerror="alert(1)`, "/site/theme.js"} {
		if _, _, err := parsePost("test", []byte(front(goodTitle, goodDescription, goodDate, "image: "+image)+"Body")); err == nil {
			t.Errorf("accepted unsafe image %q", image)
		}
	}
	files := fstest.MapFS{"post.md": {Data: []byte(front(goodTitle, goodDescription, goodDate, "image: /site/blog/missing-cover.svg") + "Body")}}
	if _, err := build(&url.URL{Scheme: "https", Host: "uplumo.com"}, files); err == nil {
		t.Fatal("missing cover must fail before a page with broken metadata is served")
	}
}

func TestPublishedBlogContent(t *testing.T) {
	s := newSite(t, &url.URL{Scheme: "https", Host: "uplumo.com"})
	posts := []struct{ slug, published, updated string }{
		{"best-seo-aeo-geo-tools", "2026-11-09", "2026-11-09"},
		{"why-website-not-ranking-google", "2026-11-08", "2026-11-08"},
		{"chatgpt-recommends-competitors-not-my-business", "2026-11-08", "2026-11-08"},
		{"chatgpt-traffic-google-analytics", "2026-10-15", "2026-11-08"},
	}
	if len(s.posts) != len(posts) {
		t.Fatalf("published %d articles, want %d approved posts", len(s.posts), len(posts))
	}
	index := get(s.ServeBlog, "/blog", nil)
	if index.Code != http.StatusOK {
		t.Fatalf("blog index status = %d, want 200", index.Code)
	}
	sitemap := get(s.ServeSitemap, "/sitemap.xml", nil).Body.String()
	for _, post := range posts {
		slug := post.slug
		if !strings.Contains(index.Body.String(), "/blog/"+slug) {
			t.Errorf("blog index lacks %s", slug)
		}
		if !strings.Contains(sitemap, "<loc>https://uplumo.com/blog/"+slug+"</loc><lastmod>"+post.updated+"</lastmod>") {
			t.Errorf("sitemap lacks dated post %s", slug)
		}
		res := getPost(s, slug)
		body := res.Body.String()
		if res.Code != http.StatusOK || strings.Count(body, "<h1>") != 1 {
			t.Errorf("%s: status %d or duplicate heading", slug, res.Code)
		}
		for _, want := range []string{
			`<link rel="canonical" href="https://uplumo.com/blog/` + slug + `">`,
			`"@type":"BlogPosting"`, `"datePublished":"` + post.published + `"`,
			`href="https://uplumo.com/pricing"`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s lacks %s", slug, want)
			}
		}
		if strings.Contains(body, "meta_description:") || strings.Contains(body, "\n---") {
			t.Errorf("%s leaked editorial metadata", slug)
		}
	}
	for _, serve := range []http.HandlerFunc{s.ServeLanding, s.ServePricing} {
		if !strings.Contains(get(serve, "/", nil).Body.String(), `href="/blog"`) {
			t.Error("public navigation lacks the approved blog")
		}
	}
}

func TestSEOGuideNavigationAndImages(t *testing.T) {
	s := newSite(t, &url.URL{Scheme: "https", Host: "uplumo.com"})
	body := getPost(s, "best-seo-aeo-geo-tools").Body.String()
	for _, id := range strings.Fields(`choose-the-right-tool get-your-pages-indexed-on-google
give-ai-search-crawlers-access write-content-for-search-intent check-your-ai-visibility
track-rankings-and-leads improve-your-local-seo your-first-30-days frequently-asked-questions`) {
		if !strings.Contains(body, `href="#`+id+`"`) || !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("guide lacks a table-of-contents link or heading for %q", id)
		}
	}
	for _, file := range []string{
		"blog/best-seo-aeo-geo-tools-cover.webp",
		"blog/best-seo-aeo-geo-tools-google-indexing.webp",
		"blog/best-seo-aeo-geo-tools-content-brief.webp",
		"blog/best-seo-aeo-geo-tools-lead-tracking.webp",
		"product/prompt-explorer.webp",
	} {
		path := AssetsPrefix + file
		if !strings.Contains(body, `src="`+path+`"`) {
			t.Errorf("guide lacks image %q", path)
		}
		if res := get(s.ServeAsset, path, nil); res.Code != http.StatusOK {
			t.Errorf("image %q returned %d, want 200", path, res.Code)
		}
	}
}

func TestEmptyBlogStillReturnsNotFound(t *testing.T) {
	s, err := build(&url.URL{Scheme: "https", Host: "uplumo.com"}, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if res := get(s.ServeBlog, "/blog", nil); res.Code != http.StatusNotFound {
		t.Errorf("empty blog index status = %d, want 404", res.Code)
	}
	if strings.Contains(get(s.ServeSitemap, "/sitemap.xml", nil).Body.String(), "/blog") {
		t.Error("empty blog remains in sitemap")
	}
}

func TestRetiredBlogContentRemainsRemoved(t *testing.T) {
	s := newSite(t, &url.URL{Scheme: "https", Host: "uplumo.com"})
	for _, slug := range strings.Fields(`
choose-the-best-seo-aeo-tool
core-web-vitals-explained
daily-keyword-rank-fluctuations-how-to-report
ecommerce-faceted-navigation-crawl-traps
ecommerce-product-pages-not-ranking
ethical-backlinks-for-small-business
fix-redirect-chains-and-loops
ga4-vs-search-console-organic-clicks-mismatch
get-cited-in-ai-search-results
google-chose-different-canonical-how-to-fix
google-core-update-organic-traffic-drop
google-manual-action-unnatural-links-recovery
google-search-console-for-small-businesses
google-search-console-sitemap-couldnt-fetch
googlebot-403-5xx-cloudflare-firewall-block
gptbot-vs-oai-searchbot-robots-txt
hotel-seo-direct-bookings
hreflang-wrong-language-page-in-google
images-not-indexed-in-google-images
internal-linking-for-small-business-seo
japanese-seo-hack-spam-urls-recovery
javascript-content-not-visible-to-google
keyword-cannibalization-how-to-diagnose-and-fix
on-page-seo-checklist-for-service-pages
out-of-stock-product-pages-seo
pdf-files-indexed-outdated-or-private-seo
product-rich-results-price-stars-not-showing
rank-tracker-results-differ-from-google
refresh-old-blog-posts-that-lost-traffic
robots-txt-vs-noindex-pages-still-indexed
search-console-impressions-clicks-down
seo-30-day-plan-small-business
seo-audit-that-finds-real-problems
seo-client-report-template-that-proves-progress
seo-client-seo-roi-long-sales-cycle
seo-content-brief-for-small-business
seo-site-migration-traffic-drop-checklist
seo-tools-show-different-search-volume
seo-website-traffic-no-leads
service-pages-that-rank-and-convert
soft-404-errors-on-live-pages
staging-site-indexed-in-google-how-to-remove
track-ai-overview-visibility-search-console
uplumo-advanced-site-audit-crawler
uplumo-ai-content-writing-search-briefs
uplumo-ai-keyword-research-content-workflow
uplumo-ai-traffic-analytics-search-performance
uplumo-ai-visibility-citation-tracking
uplumo-cite-fix-guide
uplumo-seo-aeo-content-scoring
uplumo-seo-aeo-platform-smb-enterprise
uplumo-vs-ahrefs-content-ai-visibility
uplumo-vs-semrush-seo-aeo-workflow
uplumo-vs-ubersuggest-intent-content
valid-page-returns-404-to-google
video-indexing-report-no-thumbnail
website-not-showing-on-google
what-is-answer-engine-optimisation
why-backlink-counts-differ-between-seo-tools-and-gsc
write-search-titles-and-meta-descriptions
`) {
		t.Run(slug, func(t *testing.T) {
			res := getPost(s, slug)
			if res.Code != http.StatusNotFound || res.Header().Get("Location") != "" {
				t.Errorf("removed article status/location = %d %q, want 404 without redirect", res.Code, res.Header().Get("Location"))
			}
		})
	}
	for file := range s.versions {
		if !strings.HasPrefix(file, "blog/") {
			continue
		}
		used := false
		for _, post := range s.posts {
			body := string(post.body)
			if strings.Contains(body, `src="`+AssetsPrefix+file+`"`) ||
				strings.Contains(body, AssetsPrefix+file+`?v=`+s.versions[file]) {
				used = true
				break
			}
		}
		if !used {
			t.Errorf("unused blog asset %q remains embedded", file)
		}
	}
}
