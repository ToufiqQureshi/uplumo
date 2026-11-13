package contenteditor

import (
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// importedPage is the editable text of a published page.
type importedPage struct {
	Title           string
	MetaDescription string
	Markdown        string
}

// skipped elements never hold the article: navigation, chrome and code.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true,
	atom.Nav: true, atom.Header: true, atom.Footer: true, atom.Aside: true,
	atom.Form: true, atom.Button: true, atom.Svg: true, atom.Iframe: true,
	atom.Select: true, atom.Img: true, atom.Picture: true, atom.Video: true,
	atom.Audio: true, atom.Canvas: true, atom.Figure: true,
}

// pageToMarkdown keeps the page's title, headings, paragraphs, lists, quotes
// and links, in the Markdown the editor stores. Images are left out on
// purpose. It reads the <main> or <article> element when the page has one.
func pageToMarkdown(document, pageURL string) importedPage {
	root, err := html.Parse(strings.NewReader(document))
	if err != nil {
		return importedPage{}
	}
	base, _ := url.Parse(pageURL)
	var page importedPage
	var h1 string
	walk(root, func(n *html.Node) bool {
		switch n.DataAtom {
		case atom.Title:
			if page.Title == "" {
				page.Title = text(n)
			}
		case atom.Meta:
			if strings.EqualFold(attr(n, "name"), "description") && page.MetaDescription == "" {
				page.MetaDescription = strings.TrimSpace(attr(n, "content"))
			}
		case atom.H1:
			if h1 == "" {
				h1 = text(n)
			}
		}
		return true
	})
	if h1 != "" {
		page.Title = h1
	}
	body := find(root, atom.Main)
	if body == nil {
		body = find(root, atom.Article)
	}
	if body == nil {
		body = find(root, atom.Body)
	}
	if body == nil {
		return page
	}
	w := markdownWriter{base: base, title: page.Title}
	w.block(body)
	page.Markdown = strings.TrimSpace(w.out.String())
	return page
}

type markdownWriter struct {
	out   strings.Builder
	base  *url.URL
	title string
	// titleSkipped drops the first H1 when it repeats the title field.
	titleSkipped bool
}

func (w *markdownWriter) paragraph(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	w.out.WriteString(s)
	w.out.WriteString("\n\n")
}

func (w *markdownWriter) block(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			if c.Type == html.TextNode {
				w.paragraph(collapse(c.Data))
			}
			continue
		}
		if skipped[c.DataAtom] {
			continue
		}
		switch c.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			heading := w.inline(c)
			if c.DataAtom == atom.H1 && !w.titleSkipped && heading == w.title {
				w.titleSkipped = true
				continue
			}
			level := int(c.DataAtom.String()[1] - '0')
			// The draft's title is its H1, so body headings start at H2.
			w.paragraph(strings.Repeat("#", max(level, 2)) + " " + heading)
		case atom.P:
			w.paragraph(w.inline(c))
		case atom.Ul, atom.Ol:
			w.list(c, c.DataAtom == atom.Ol)
		case atom.Blockquote:
			if quote := w.inline(c); quote != "" {
				w.paragraph("> " + quote)
			}
		case atom.Br, atom.Hr:
		default:
			w.block(c)
		}
	}
}

func (w *markdownWriter) list(n *html.Node, ordered bool) {
	i := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom != atom.Li {
			continue
		}
		item := w.inline(c)
		if item == "" {
			continue
		}
		i++
		marker := "- "
		if ordered {
			marker = strconv.Itoa(i) + ". "
		}
		w.out.WriteString(marker + item + "\n")
	}
	if i > 0 {
		w.out.WriteString("\n")
	}
}

// inline renders a block's text with bold, italics and links kept.
func (w *markdownWriter) inline(n *html.Node) string {
	var b strings.Builder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c.Type == html.TextNode:
				b.WriteString(c.Data)
			case c.Type != html.ElementNode || skipped[c.DataAtom]:
			case c.DataAtom == atom.Br:
				b.WriteString(" ")
			case c.DataAtom == atom.Strong || c.DataAtom == atom.B:
				if t := strings.TrimSpace(w.inline(c)); t != "" {
					b.WriteString(" **" + t + "** ")
				}
			case c.DataAtom == atom.Em || c.DataAtom == atom.I:
				if t := strings.TrimSpace(w.inline(c)); t != "" {
					b.WriteString(" *" + t + "* ")
				}
			case c.DataAtom == atom.A:
				t := strings.TrimSpace(w.inline(c))
				href := w.link(attr(c, "href"))
				if t != "" && href != "" {
					b.WriteString(" [" + t + "](" + href + ") ")
				} else {
					b.WriteString(" " + t + " ")
				}
			case c.DataAtom == atom.Ul || c.DataAtom == atom.Ol:
				// A nested list reads as part of its item.
				b.WriteString(" " + w.inline(c) + " ")
			default:
				visit(c)
			}
		}
	}
	visit(n)
	return collapse(b.String())
}

// link makes an href absolute; non-web links are dropped.
func (w *markdownWriter) link(href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if w.base != nil {
		u = w.base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return strings.NewReplacer("(", "%28", ")", "%29", " ", "%20").Replace(u.String())
}

func walk(n *html.Node, visit func(*html.Node) bool) {
	if n.Type == html.ElementNode && !visit(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}

func find(n *html.Node, a atom.Atom) *html.Node {
	var found *html.Node
	walk(n, func(n *html.Node) bool {
		if found == nil && n.DataAtom == a {
			found = n
		}
		return found == nil
	})
	return found
}

func text(n *html.Node) string {
	var b strings.Builder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				b.WriteString(c.Data)
				b.WriteString(" ")
			} else if c.Type == html.ElementNode && !skipped[c.DataAtom] {
				visit(c)
			}
		}
	}
	visit(n)
	return collapse(b.String())
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

// collapse joins runs of whitespace and tidies spaces before punctuation.
func collapse(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.NewReplacer(" .", ".", " ,", ",", " :", ":", " ;", ";", " !", "!", " ?", "?", " )", ")", "( ", "(").Replace(s)
}
