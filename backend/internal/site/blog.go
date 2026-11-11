package site

import (
	"bytes"
	"embed"
	"fmt"
	stdhtml "html"
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
	// Related holds up to maxRelated slugs from the "related" front matter,
	// shown as further reading under the post.
	Related []string
	// FAQ holds the questions under the post's "Frequently asked questions"
	// heading, in plain text, for FAQPage structured data.
	FAQ []QA
}

// Editorial metadata budgets; search-result display width varies.
const (
	maxTitleRunes       = 70
	minDescriptionRunes = 50
	maxDescriptionRunes = 160
	maxRelated          = 3
)

// loadPosts reads every post in fsys, newest first, skipping drafts.
func loadPosts(fsys fs.FS) ([]Post, error) {
	files, err := fs.Glob(fsys, "*.md")
	if err != nil {
		return nil, fmt.Errorf("list blog posts: %w", err)
	}
	posts := make([]Post, 0, len(files))
	for _, file := range files {
		// Editorial files share the blog directory with articles.
		if file == "README.md" || file == "blog_style.md" {
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
	for related := range strings.SplitSeq(fields["related"], ",") {
		related = strings.TrimSpace(related)
		switch {
		case related == "":
			continue
		case !slugPattern.MatchString(related) || related == slug:
			return Post{}, false, fmt.Errorf("related %q must be another post's slug", related)
		case slices.Contains(post.Related, related):
			return Post{}, false, fmt.Errorf("related %q is listed twice", related)
		}
		post.Related = append(post.Related, related)
	}
	if len(post.Related) > maxRelated {
		return Post{}, false, fmt.Errorf("related lists %d posts; keep it to %d", len(post.Related), maxRelated)
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
	if post.FAQ, err = postFAQ(body); err != nil {
		return Post{}, false, fmt.Errorf("faq: %w", err)
	}
	return post, fields["draft"] == "true", nil
}

// relatedPosts resolves each post's related slugs against the published
// posts. A post without a list gets the newest other posts, so every post
// links onward. An unknown slug stops the build, like any broken post.
func relatedPosts(posts []Post) (map[string][]Post, error) {
	bySlug := make(map[string]Post, len(posts))
	for _, post := range posts {
		bySlug[post.Slug] = post
	}
	related := make(map[string][]Post, len(posts))
	for _, post := range posts {
		for _, slug := range post.Related {
			other, ok := bySlug[slug]
			if !ok {
				return nil, fmt.Errorf("blog post %s: related %q is not a published post", post.Slug, slug)
			}
			related[post.Slug] = append(related[post.Slug], other)
		}
		for _, other := range posts {
			if len(post.Related) > 0 || len(related[post.Slug]) == maxRelated {
				break
			}
			if other.Slug != post.Slug {
				related[post.Slug] = append(related[post.Slug], other)
			}
		}
	}
	return related, nil
}

// faqHeading marks the section whose H3 headings are questions.
const faqHeading = "## frequently asked questions"

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// postFAQ reads the H3 questions and their answers under the post's
// "Frequently asked questions" H2, up to the next H2. Answers become plain
// text, because structured data must not carry markup.
func postFAQ(body string) ([]QA, error) {
	var faq []QA
	var answer []string
	inFAQ := false
	flush := func() error {
		if len(faq) == 0 {
			return nil
		}
		var html bytes.Buffer
		if err := blogMarkdown.Convert([]byte(strings.Join(answer, "\n")), &html); err != nil {
			return err
		}
		text := stdhtml.UnescapeString(htmlTag.ReplaceAllString(html.String(), " "))
		text = strings.Join(strings.Fields(text), " ")
		text = strings.NewReplacer(" .", ".", " ,", ",", " ?", "?", " !", "!").Replace(text)
		faq[len(faq)-1].Answer = text
		answer = nil
		return nil
	}
	for line := range strings.SplitSeq(body, "\n") {
		switch trimmed := strings.TrimSpace(line); {
		case strings.HasPrefix(trimmed, "## "):
			if inFAQ {
				return faq, flush()
			}
			inFAQ = strings.EqualFold(trimmed, faqHeading)
		case inFAQ && strings.HasPrefix(trimmed, "### "):
			if err := flush(); err != nil {
				return nil, err
			}
			faq = append(faq, QA{Question: strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))})
		case inFAQ && len(faq) > 0:
			answer = append(answer, line)
		}
	}
	return faq, flush()
}

// postJSONLD describes a post for search engines: the article, its place in
// the site and, when the post has a FAQ section, its questions.
func postJSONLD(origin string, post Post, image string) []any {
	postURL := origin + "/blog/" + post.Slug
	graph := []any{
		blogPostingJSONLD(origin, post, image),
		map[string]any{
			"@context": "https://schema.org",
			"@type":    "BreadcrumbList",
			"itemListElement": []map[string]any{
				{"@type": "ListItem", "position": 1, "name": "Home", "item": origin + "/"},
				{"@type": "ListItem", "position": 2, "name": "Blog", "item": origin + "/blog"},
				{"@type": "ListItem", "position": 3, "name": post.Title, "item": postURL},
			},
		},
	}
	if len(post.FAQ) == 0 {
		return graph
	}
	questions := make([]map[string]any, 0, len(post.FAQ))
	for _, qa := range post.FAQ {
		questions = append(questions, map[string]any{
			"@type":          "Question",
			"name":           qa.Question,
			"acceptedAnswer": map[string]string{"@type": "Answer", "text": qa.Answer},
		})
	}
	return append(graph, map[string]any{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": questions,
	})
}

// blogPostingJSONLD describes a post for search engines.
func blogPostingJSONLD(origin string, post Post, image string) map[string]any {
	author := map[string]string{"@type": authorType(post.Author), "name": post.Author}
	if post.Author == "ChatGPT Plus Content Writer Agent with Skills" {
		// Credit the publisher in structured data when the visible byline is an AI tool.
		author = map[string]string{"@type": "Organization", "name": name}
	}
	return map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         post.Title,
		"description":      post.Description,
		"datePublished":    post.Published.Format(time.DateOnly),
		"dateModified":     post.Updated.Format(time.DateOnly),
		"mainEntityOfPage": origin + "/blog/" + post.Slug,
		"image":            image,
		"author":           author,
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
