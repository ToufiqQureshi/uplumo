package contenteditor

import "testing"

func TestPageToMarkdownKeepsTheTextExactly(t *testing.T) {
	for _, tc := range []struct{ name, html, want string }{
		{"escaped html example", `<p>Use &lt;strong&gt; for bold.</p>`, `Use \<strong\> for bold.`},
		{"literal markdown punctuation", `<p>Fields marked *required* or _new_ use [brackets] and a \ slash.</p>`,
			`Fields marked \*required\* or \_new\_ use \[brackets\] and a \\ slash.`},
		{"code ticks", "<p>Run `make`</p>", "Run \\`make\\`"},
		{"line that looks like a heading", `<p># 1 tip</p>`, `\# 1 tip`},
		{"line that looks like a list", `<p>- not a list</p><p>2026. A year</p><p>+ plus</p>`, "\\- not a list\n\n2026\\. A year\n\n\\+ plus"},
		{"line that looks like a quote", `<p>&gt; not a quote</p>`, `\> not a quote`},
		{"real formatting stays", `<p>A <strong>bold *star*</strong> and <a href="/x">a [link]</a>.</p>`,
			`A **bold \*star\*** and [a \[link\]](http://example.test/x).`},
		{"text in a div keeps its link", `<div>Read <a href="/guide">our guide</a> now.</div>`,
			`Read [our guide](http://example.test/guide) now.`},
		{"div text keeps bold and italics", `<div><span>Fast</span> <b>setup</b>, <em>no</em> fees</div>`, `Fast **setup**, *no* fees`},
		{"div mixes text runs and blocks", `<div>Intro <a href="/a">link</a><h2>Next</h2>Outro <i>here</i></div>`,
			"Intro [link](http://example.test/a)\n\n## Next\n\nOutro *here*"},
		{"title with markdown characters is still dropped once", `<h1>A * B</h1><p>x</p>`, `x`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := pageToMarkdown("<html><body><main>"+tc.html+"</main></body></html>", "http://example.test/page").Markdown
			if got != tc.want {
				t.Fatalf("got\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}
