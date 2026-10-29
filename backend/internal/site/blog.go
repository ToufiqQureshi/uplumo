package site

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Blog posts are Markdown files in blog/, one per post, named <slug>.md. They
// are parsed and rendered once at startup like every other public page, so a
// broken post stops the deploy instead of serving a broken page.

//go:embed blog
var blogFS embed.FS

// blogMarkdown renders GitHub-flavoured Markdown. Raw HTML in a post is
// dropped, not passed through, so a post cannot carry a script. Headings get
// ids, so a section can be linked to.
var blogMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// Retired articles resolve to maintained guides covering their original task.
// Keep these URLs out of the index and sitemap; avoid chains and home redirects.
var blogRedirects = map[string]string{
	"daily-keyword-rank-fluctuations-how-to-report":        "search-console-impressions-clicks-down",
	"ecommerce-faceted-navigation-crawl-traps":             "ecommerce-product-pages-not-ranking",
	"fix-redirect-chains-and-loops":                        "seo-audit-that-finds-real-problems",
	"ga4-vs-search-console-organic-clicks-mismatch":        "uplumo-ai-traffic-analytics-search-performance",
	"google-chose-different-canonical-how-to-fix":          "website-not-showing-on-google",
	"google-core-update-organic-traffic-drop":              "refresh-old-blog-posts-that-lost-traffic",
	"google-manual-action-unnatural-links-recovery":        "ethical-backlinks-for-small-business",
	"google-search-console-sitemap-couldnt-fetch":          "website-not-showing-on-google",
	"googlebot-403-5xx-cloudflare-firewall-block":          "seo-audit-that-finds-real-problems",
	"gptbot-vs-oai-searchbot-robots-txt":                   "get-cited-in-ai-search-results",
	"hreflang-wrong-language-page-in-google":               "seo-audit-that-finds-real-problems",
	"images-not-indexed-in-google-images":                  "ecommerce-product-pages-not-ranking",
	"japanese-seo-hack-spam-urls-recovery":                 "website-not-showing-on-google",
	"javascript-content-not-visible-to-google":             "uplumo-advanced-site-audit-crawler",
	"keyword-cannibalization-how-to-diagnose-and-fix":      "refresh-old-blog-posts-that-lost-traffic",
	"out-of-stock-product-pages-seo":                       "ecommerce-product-pages-not-ranking",
	"pdf-files-indexed-outdated-or-private-seo":            "website-not-showing-on-google",
	"product-rich-results-price-stars-not-showing":         "ecommerce-product-pages-not-ranking",
	"rank-tracker-results-differ-from-google":              "search-console-impressions-clicks-down",
	"robots-txt-vs-noindex-pages-still-indexed":            "website-not-showing-on-google",
	"seo-client-report-template-that-proves-progress":      "uplumo-seo-aeo-platform-smb-enterprise",
	"seo-client-seo-roi-long-sales-cycle":                  "seo-website-traffic-no-leads",
	"seo-site-migration-traffic-drop-checklist":            "seo-audit-that-finds-real-problems",
	"seo-tools-show-different-search-volume":               "uplumo-ai-keyword-research-content-workflow",
	"soft-404-errors-on-live-pages":                        "website-not-showing-on-google",
	"staging-site-indexed-in-google-how-to-remove":         "website-not-showing-on-google",
	"valid-page-returns-404-to-google":                     "seo-audit-that-finds-real-problems",
	"video-indexing-report-no-thumbnail":                   "seo-audit-that-finds-real-problems",
	"why-backlink-counts-differ-between-seo-tools-and-gsc": "ethical-backlinks-for-small-business",
}

var blogImagePattern = regexp.MustCompile(`^/site/(blog|product)/[a-z0-9-]+\.(svg|webp|png)$`)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Post is one published blog post.
type Post struct {
	Slug        string
	Title       string
	Description string
	Image       string
	Published   time.Time
	Updated     time.Time
	Author      string
	Body        template.HTML
	// ReadMinutes is the reading time at 200 words a minute, at least 1.
	ReadMinutes int
}

// Editorial metadata budgets; search-result display width varies.
const (
	maxTitleRunes       = 70
	minDescriptionRunes = 50
	maxDescriptionRunes = 160
)

// loadPosts reads every post in fsys, newest first, skipping drafts.
func loadPosts(fsys fs.FS) ([]Post, error) {
	files, err := fs.Glob(fsys, "*.md")
	if err != nil {
		return nil, fmt.Errorf("list blog posts: %w", err)
	}
	posts := make([]Post, 0, len(files))
	for _, file := range files {
		// README.md explains how to write a post; it is not one.
		if file == "README.md" {
			continue
		}
		source, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("read blog post %s: %w", file, err)
		}
		post, draft, err := parsePost(strings.TrimSuffix(path.Base(file), ".md"), source)
		if err != nil {
			return nil, fmt.Errorf("blog post %s: %w", file, err)
		}
		if !draft {
			posts = append(posts, post)
		}
	}
	slices.SortFunc(posts, func(a, b Post) int {
		if order := b.Published.Compare(a.Published); order != 0 {
			return order
		}
		return strings.Compare(a.Slug, b.Slug)
	})
	return posts, nil
}

// parsePost reads the front matter (between two "---" lines, one "key: value"
// per line) and renders the body.
func parsePost(slug string, source []byte) (Post, bool, error) {
	if !slugPattern.MatchString(slug) {
		return Post{}, false, fmt.Errorf("file name must be lower-case words joined by hyphens, like how-to-rank.md")
	}
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return Post{}, false, fmt.Errorf("must start with a --- front matter block")
	}
	header, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return Post{}, false, fmt.Errorf("front matter must end with a --- line")
	}
	fields := map[string]string{}
	for line := range strings.SplitSeq(header, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Post{}, false, fmt.Errorf("front matter line %q is not key: value", line)
		}
		fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}

	post := Post{Slug: slug, Title: fields["title"], Description: fields["description"], Image: fields["image"], Author: fields["author"]}
	if post.Image != "" && !blogImagePattern.MatchString(post.Image) {
		return Post{}, false, fmt.Errorf("image must be a local /site/blog/ or /site/product/ asset")
	}
	if post.Author == "" {
		post.Author = "Uplumo team"
	}
	switch n := utf8.RuneCountInString(post.Title); {
	case n == 0:
		return Post{}, false, fmt.Errorf("title is required")
	case n > maxTitleRunes:
		return Post{}, false, fmt.Errorf("title is %d characters; keep it to the editorial limit of %d", n, maxTitleRunes)
	}
	if n := utf8.RuneCountInString(post.Description); n < minDescriptionRunes || n > maxDescriptionRunes {
		return Post{}, false, fmt.Errorf("description is %d characters; it must be %d to %d", n, minDescriptionRunes, maxDescriptionRunes)
	}
	var err error
	if post.Published, err = time.Parse(time.DateOnly, fields["date"]); err != nil {
		return Post{}, false, fmt.Errorf("date must be YYYY-MM-DD: %w", err)
	}
	post.Updated = post.Published
	if updated := fields["updated"]; updated != "" {
		if post.Updated, err = time.Parse(time.DateOnly, updated); err != nil {
			return Post{}, false, fmt.Errorf("updated must be YYYY-MM-DD: %w", err)
		}
	}
	if strings.TrimSpace(body) == "" {
		return Post{}, false, fmt.Errorf("the post has no text")
	}
	var html bytes.Buffer
	if err := blogMarkdown.Convert([]byte(body), &html); err != nil {
		return Post{}, false, fmt.Errorf("render: %w", err)
	}
	post.Body = template.HTML(html.String()) // #nosec G203 -- goldmark drops raw HTML by default.
	post.ReadMinutes = max(1, (len(strings.Fields(body))+199)/200)
	return post, fields["draft"] == "true", nil
}

// blogPostingJSONLD describes a post for search engines.
func blogPostingJSONLD(origin string, post Post, image string) map[string]any {
	return map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         post.Title,
		"description":      post.Description,
		"datePublished":    post.Published.Format(time.DateOnly),
		"dateModified":     post.Updated.Format(time.DateOnly),
		"mainEntityOfPage": origin + "/blog/" + post.Slug,
		"image":            image,
		"author":           map[string]string{"@type": authorType(post.Author), "name": post.Author},
		"publisher": map[string]any{
			"@type": "Organization", "name": name,
			"logo": map[string]string{"@type": "ImageObject", "url": origin + AssetsPrefix + "logo.svg"},
		},
	}
}

// authorType is Organization for the default byline and Person for a named one.
func authorType(author string) string {
	if author == "Uplumo team" {
		return "Organization"
	}
	return "Person"
}
