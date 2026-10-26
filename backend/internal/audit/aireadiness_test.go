package audit

import (
	"slices"
	"testing"
)

func TestAnalyzeHTMLCollectsAISignals(t *testing.T) {
	page := `<!doctype html><html><head>
<script type="application/ld+json">{"@context":"https://schema.org","@graph":[{"@type":"Hotel","name":"Lagoon"},{"@type":["WebPage","FAQPage"]}]}</script>
<script type="application/ld+json">{ not json</script>
</head><body>
<h2>How far is Lonavala from Mumbai?</h2><p>83 km.</p>
<h3>What time is check-in</h3><p>12 noon.</p>
<h2>Our rooms</h2>
<a href="tel:+91 22 1234 5678">Call</a> <a href="tel:022-12345678">Call again</a> <a href="tel:98765 43210">Mobile</a>
<a href="tel:100">Too short</a>
</body></html>`
	got := AnalyzeHTML(page, "https://lagoon.example/", 200, 10, "").AI
	if !slices.Equal(got.SchemaTypes, []string{"Hotel", "WebPage", "FAQPage"}) {
		t.Fatalf("schema types = %q", got.SchemaTypes)
	}
	if got.QuestionHeadings != 2 {
		t.Fatalf("question headings = %d, want 2", got.QuestionHeadings)
	}
	if !slices.Equal(got.Phones, []string{"2212345678", "9876543210"}) {
		t.Fatalf("phones = %q", got.Phones)
	}
}

func TestAnalyzeHTMLKeepsJSONLDOutOfBodyText(t *testing.T) {
	got := AnalyzeHTML(`<html><body><script type="application/ld+json">{"@type":"Organization","name":"Secret words"}</script><p>Visible</p></body></html>`, "https://x.example/", 200, 1, "")
	if got.BodyText != "Visible" || got.WordCount != 1 {
		t.Fatalf("body text = %q (%d words)", got.BodyText, got.WordCount)
	}
	if !slices.Equal(got.AI.SchemaTypes, []string{"Organization"}) {
		t.Fatalf("schema types = %q", got.AI.SchemaTypes)
	}
}

func TestIsQuestionHeading(t *testing.T) {
	for text, want := range map[string]bool{
		"Is breakfast included?":   true,
		"Breakfast included?":      true,
		"how to reach tiger point": true,
		"  Which room is best ":    true,
		"Our rooms":                false,
		"Island tours":             false, // "is" only as a whole word
		"":                         false,
	} {
		if got := isQuestionHeading(text); got != want {
			t.Errorf("isQuestionHeading(%q) = %v, want %v", text, got, want)
		}
	}
}

func aiPage(url string, signals AIPageSignals) CrawledPageResult {
	return CrawledPageResult{URL: url, StatusCode: 200, IsHTML: true, AI: signals}
}

func aiIssueTypes(issues []DetectedIssue) []IssueType {
	types := make([]IssueType, 0, len(issues))
	for _, issue := range issues {
		types = append(types, issue.IssueType)
	}
	return types
}

func TestAISiteSignalsIssues(t *testing.T) {
	const start = "https://shop.example/"
	tests := []struct {
		name  string
		pages []CrawledPageResult
		home  string
		want  []IssueType
	}{
		{name: "no HTML page read: no guesses", pages: []CrawledPageResult{{URL: start, StatusCode: 500, IsHTML: true}}},
		{
			name: "ready site",
			pages: []CrawledPageResult{
				aiPage(start, AIPageSignals{SchemaTypes: []string{"LocalBusiness"}, Phones: []string{"9876543210"}}),
				aiPage(start+"faq", AIPageSignals{SchemaTypes: []string{"FAQPage"}, Phones: []string{"9876543210"}}),
			},
			home: start,
		},
		{
			name: "everything missing",
			pages: []CrawledPageResult{
				aiPage(start, AIPageSignals{QuestionHeadings: 1, Phones: []string{"1111111111"}}),
				aiPage(start+"contact", AIPageSignals{SchemaTypes: []string{"WebPage"}, Phones: []string{"2222222222"}}),
			},
			home: start,
			want: []IssueType{IssueMissingBusinessSchema, IssueNoQuestionAnswers, IssueSeveralPhoneNumbers},
		},
		{
			name: "business schema only on an inner page still flags the home page",
			pages: []CrawledPageResult{
				aiPage(start, AIPageSignals{QuestionHeadings: 2}),
				aiPage(start+"about", AIPageSignals{SchemaTypes: []string{"Organization"}}),
			},
			home: start,
			want: []IssueType{IssueMissingBusinessSchema},
		},
		{
			name: "home page never reached: any page with business schema is enough",
			pages: []CrawledPageResult{
				aiPage(start+"about", AIPageSignals{SchemaTypes: []string{"dentist"}, QuestionHeadings: 3}),
			},
		},
		{
			name: "redirected home page is skipped, the real one is judged",
			pages: []CrawledPageResult{
				{URL: "http://shop.example/", StatusCode: 301},
				aiPage(start, AIPageSignals{SchemaTypes: []string{"Restaurant"}, QuestionHeadings: 2}),
			},
			home: start,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signals := &aiSiteSignals{}
			for _, page := range tt.pages {
				signals.add(page, page.URL == tt.home && tt.home != "")
			}
			if got := aiIssueTypes(signals.issues(start)); !slices.Equal(got, tt.want) {
				t.Fatalf("issues = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIReadinessIssuesHaveCopy(t *testing.T) {
	for _, issueType := range []IssueType{IssueMissingBusinessSchema, IssueNoQuestionAnswers, IssueSeveralPhoneNumbers} {
		descriptor, ok := DescribeIssueType(string(issueType))
		if !ok || descriptor.Title == "" || descriptor.HowToFix == "" {
			t.Fatalf("%s: %+v", issueType, descriptor)
		}
	}
}

func TestIsHomePage(t *testing.T) {
	for raw, want := range map[string]bool{
		"https://a.example":        true,
		"https://a.example/":       true,
		"https://a.example/?utm=x": false,
		"https://a.example/rooms":  false,
		"://bad":                   false,
	} {
		if got := isHomePage(raw); got != want {
			t.Errorf("isHomePage(%q) = %v, want %v", raw, got, want)
		}
	}
}
