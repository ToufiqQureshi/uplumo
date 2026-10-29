package audit

import (
	"encoding/json"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// AI readiness: can an AI assistant tell who the business is and quote a
// direct answer from the site? These checks read the HTML the crawl already
// fetched, so they cost nothing.

// maxLDJSONBytes caps the JSON-LD kept per page; real blocks are a few KB.
const maxLDJSONBytes = 256 * 1024

// AIPageSignals are the per-page facts the AI checks need. Everything but
// Phones is stored with the page (citation_signals_json) for Cite Fix.
type AIPageSignals struct {
	// Version distinguishes scans which actually measured the current signals.
	Version int `json:"version"`
	// SchemaTypes are the JSON-LD @type values on the page, @graph included.
	SchemaTypes []string `json:"schemaTypes"`
	// QuestionHeadings counts H2/H3 headings phrased as a question.
	QuestionHeadings int `json:"questionHeadings"`
	// Phones are tel: link numbers, reduced to their last 10 digits. Only the
	// site-wide phone check reads them, during the crawl.
	Phones []string `json:"-"`
	// Author is true when the page names who wrote it: a meta or itemprop
	// author, a rel=author link, or a JSON-LD author.
	Author bool `json:"author"`
	// Published and Modified are the first publish and update dates the page
	// states (meta, itemprop or JSON-LD), as written, at most 40 characters.
	Published string `json:"published,omitempty"`
	Modified  string `json:"modified,omitempty"`
	// LeadWords counts the words of the first paragraph of the main content
	// (outside nav, header, footer and aside): is the answer up front?
	LeadWords int `json:"leadWords"`
	// Lists and Tables count ul/ol and table elements in the main content;
	// Quotes counts blockquote elements.
	Lists  int `json:"lists"`
	Tables int `json:"tables"`
	Quotes int `json:"quotes"`
	// Statistics counts percent, currency and lakh/crore figures in the body.
	Statistics             int                `json:"statistics"`
	SnippetBlocked         bool               `json:"snippetBlocked"`
	SnippetTextWords       int                `json:"snippetTextWords"`
	NoSnippetWords         int                `json:"noSnippetWords"`
	AnswerCapsule          bool               `json:"answerCapsule"`
	DefinitionAfterHeading bool               `json:"definitionAfterHeading"`
	Sections               int                `json:"sections"`
	ChunkReadySections     int                `json:"chunkReadySections"`
	AuthorityLinks         int                `json:"authorityLinks"`
	DecayYears             []int              `json:"decayYears"`
	DatedClaims            bool               `json:"datedClaims"`
	InjectionEvidence      []CitationEvidence `json:"injectionEvidence"`
	TopicHeadings          []string           `json:"topicHeadings"`
	TopicLinks             []string           `json:"topicLinks"`
	HreflangTargets        []HreflangTarget   `json:"hreflangTargets"`
	HSTS                   *bool              `json:"hsts"`
}

// CitationSignalsVersion changes whenever stored facts gain new checks.
const CitationSignalsVersion = 2

// CitationEvidence is bounded, untrusted page text for the customer to review.
type CitationEvidence struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// maxDateChars bounds a stored date; real ones are ISO dates or short text.
const maxDateChars = 40

// statisticPattern finds the figures an assistant can quote as a fact.
var statisticPattern = regexp.MustCompile(`(?i)\d[\d,.]*\s?%|[₹$€£]\s?\d|\brs\.?\s?\d|\d[\d,.]*\s?(?:lakh|crore|million|billion)\b`)

// maxCountedStatistics caps the count; a price list is not more evidence.
const maxCountedStatistics = 100

// countStatistics returns how many quotable figures the text holds.
func countStatistics(text string) int {
	return len(statisticPattern.FindAllStringIndex(text, maxCountedStatistics))
}

// businessTypes are schema.org types that say who the business is. Local
// business subtypes are many; these cover the customers Uplumo serves first.
var businessTypes = map[string]bool{
	"organization": true, "corporation": true, "localbusiness": true,
	"professionalservice": true, "store": true, "onlinestore": true,
	"hotel": true, "lodgingbusiness": true, "resort": true, "bedandbreakfast": true,
	"restaurant": true, "foodestablishment": true, "cafeorcoffeeshop": true,
	"medicalbusiness": true, "medicalclinic": true, "dentist": true, "physician": true,
	"hospital": true, "educationalorganization": true, "school": true,
	"travelagency": true, "realestateagent": true, "legalservice": true,
	"accountingservice": true, "financialservice": true, "homeandconstructionbusiness": true,
	"automotivebusiness": true, "healthandbeautybusiness": true, "sportsactivitylocation": true,
}

// questionWords start an English question heading that has no "?".
var questionWords = []string{
	"what ", "how ", "why ", "which ", "when ", "where ", "who ",
	"can ", "is ", "are ", "does ", "do ", "should ",
}

// ldFacts are what one JSON-LD block says about the page.
type ldFacts struct {
	types     []string
	author    bool
	published string
	modified  string
}

// jsonLDFacts reads every @type, any author and the first publish and update
// dates in one JSON-LD block. Invalid JSON yields nothing: a block AI cannot
// parse describes nothing.
func jsonLDFacts(block string) ldFacts {
	var value any
	if err := json.Unmarshal([]byte(block), &value); err != nil {
		return ldFacts{}
	}
	var facts ldFacts
	var walk func(any)
	walk = func(node any) {
		switch v := node.(type) {
		case []any:
			for _, item := range v {
				walk(item)
			}
		case map[string]any:
			switch t := v["@type"].(type) {
			case string:
				facts.types = append(facts.types, t)
			case []any:
				for _, item := range t {
					if s, ok := item.(string); ok {
						facts.types = append(facts.types, s)
					}
				}
			}
			if author, ok := v["author"]; ok && author != nil && author != "" {
				facts.author = true
			}
			if date, ok := v["datePublished"].(string); ok && facts.published == "" {
				facts.published = truncateChars(strings.TrimSpace(date), maxDateChars)
			}
			if date, ok := v["dateModified"].(string); ok && facts.modified == "" {
				facts.modified = truncateChars(strings.TrimSpace(date), maxDateChars)
			}
			// Sorted, so "the first date" is the same on every crawl.
			for _, key := range slices.Sorted(maps.Keys(v)) {
				if key != "@type" {
					walk(v[key])
				}
			}
		}
	}
	walk(value)
	return facts
}

// isQuestionHeading reports whether a heading reads as a question.
func isQuestionHeading(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	if strings.HasSuffix(text, "?") {
		return true
	}
	for _, word := range questionWords {
		if strings.HasPrefix(text, word) {
			return true
		}
	}
	return false
}

// telNumber reduces a tel: href to its last 10 digits ("" if fewer), so
// "+91 22 1234 5678" and "022-12345678" compare equal.
func telNumber(href string) string {
	rest, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(href)), "tel:")
	if !ok {
		return ""
	}
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, rest)
	if len(digits) < 10 {
		return ""
	}
	return digits[len(digits)-10:]
}

// aiSiteSignals collects page signals during one crawl.
type aiSiteSignals struct {
	pages           int
	sawHome         bool
	homeHasBusiness bool
	anyBusiness     bool
	faqPages        int
	phonePages      map[string]string // number -> first page that shows it
	phoneOrder      []string
	probes          []externalProbe
	probeIndex      map[string]int
	probeLinks      int
	probeImages     int
	probeLimited    bool
}

// isHomePage reports whether a crawled URL is the site's root page.
func isHomePage(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == ""
}

func (s *aiSiteSignals) add(page CrawledPageResult, home bool) {
	if !page.IsHTML || page.StatusCode < 200 || page.StatusCode >= 300 {
		return
	}
	s.pages++
	s.collectExternalProbes(page)
	signals := page.AI
	business := slices.ContainsFunc(signals.SchemaTypes, func(t string) bool {
		return businessTypes[strings.ToLower(t)]
	})
	s.anyBusiness = s.anyBusiness || business
	if home {
		s.sawHome = true
		s.homeHasBusiness = s.homeHasBusiness || business
	}
	faq := slices.ContainsFunc(signals.SchemaTypes, func(t string) bool {
		return strings.EqualFold(t, "FAQPage")
	})
	if faq || signals.QuestionHeadings >= 2 {
		s.faqPages++
	}
	for _, phone := range signals.Phones {
		if s.phonePages == nil {
			s.phonePages = map[string]string{}
		}
		if _, seen := s.phonePages[phone]; !seen {
			s.phonePages[phone] = page.URL
			s.phoneOrder = append(s.phoneOrder, phone)
		}
	}
}

// issues turns the crawl's signals into site-wide AI readiness issues. A crawl
// that read no HTML page reports nothing rather than guessing.
func (s *aiSiteSignals) issues(startURL string) []DetectedIssue {
	if s.pages == 0 {
		return nil
	}
	var issues []DetectedIssue
	// Judge the home page when the crawl reached it; otherwise any page.
	if (s.sawHome && !s.homeHasBusiness) || (!s.sawHome && !s.anyBusiness) {
		issues = append(issues, DetectedIssue{
			IssueType: IssueMissingBusinessSchema,
			PageURL:   startURL,
			Details:   map[string]any{"foundOnOtherPages": s.anyBusiness},
		})
	}
	if s.faqPages == 0 {
		issues = append(issues, DetectedIssue{
			IssueType: IssueNoQuestionAnswers,
			PageURL:   startURL,
			Details:   map[string]any{"pagesChecked": s.pages},
		})
	}
	if len(s.phoneOrder) > 1 {
		shown := make([]string, 0, min(len(s.phoneOrder), 5))
		for _, phone := range s.phoneOrder[:min(len(s.phoneOrder), 5)] {
			shown = append(shown, phone+" on "+s.phonePages[phone])
		}
		issues = append(issues, DetectedIssue{
			IssueType: IssueSeveralPhoneNumbers,
			PageURL:   startURL,
			Details:   map[string]any{"numbers": shown},
		})
	}
	return issues
}
