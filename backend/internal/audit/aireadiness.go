package audit

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"unicode"
)

// AI readiness: can an AI assistant tell who the business is and quote a
// direct answer from the site? These checks read the HTML the crawl already
// fetched, so they cost nothing.

// maxLDJSONBytes caps the JSON-LD kept per page; real blocks are a few KB.
const maxLDJSONBytes = 256 * 1024

// AIPageSignals are the per-page facts the site-wide AI checks need.
type AIPageSignals struct {
	// SchemaTypes are the JSON-LD @type values on the page, @graph included.
	SchemaTypes []string
	// QuestionHeadings counts H2/H3 headings phrased as a question.
	QuestionHeadings int
	// Phones are tel: link numbers, reduced to their last 10 digits.
	Phones []string
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

// schemaTypes returns every @type in one JSON-LD block. Invalid JSON yields
// none: a block AI cannot parse does not describe the business.
func schemaTypes(block string) []string {
	var value any
	if err := json.Unmarshal([]byte(block), &value); err != nil {
		return nil
	}
	var types []string
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
				types = append(types, t)
			case []any:
				for _, item := range t {
					if s, ok := item.(string); ok {
						types = append(types, s)
					}
				}
			}
			for key, child := range v {
				if key != "@type" {
					walk(child)
				}
			}
		}
	}
	walk(value)
	return types
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
