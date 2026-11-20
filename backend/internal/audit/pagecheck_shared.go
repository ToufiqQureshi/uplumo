package audit

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Helpers both free-audit engines use. The checks themselves live in
// seoengine.go and aeoengine.go.

// newCheck starts a check. Its weight comes from its severity.
func newCheck(id, category, severity, title, why string) PageCheck {
	return PageCheck{ID: id, Category: category, Severity: severity, Weight: severityWeights[severity],
		Title: title, Why: why, Status: CheckPass}
}

// set records the outcome of a check.
func (c *PageCheck) set(status CheckStatus, found, fix string) {
	c.Status, c.Found, c.Fix = status, found, fix
}

// lengthCheck checks a text for presence and length in characters.
func lengthCheck(id, category, severity, title, value string, minChars, maxChars int, why, fix string) PageCheck {
	c := newCheck(id, category, severity, title, why)
	length := utf8.RuneCountInString(value)
	switch {
	case value == "":
		c.set(CheckFail, "Missing.", fix)
	case length > maxChars:
		c.set(CheckWarn, fmt.Sprintf("%q (%d characters, longer than %d; Google may cut it).", value, length, maxChars), fix)
	case length < minChars:
		c.set(CheckWarn, fmt.Sprintf("%q (%d characters, too short to describe the page).", value, length), fix)
	default:
		c.set(CheckPass, fmt.Sprintf("%q (%d characters).", value, length), "")
	}
	return c
}

// titleStopWords are words that do not make a title and a heading related.
var titleStopWords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true, "your": true,
	"our": true, "you": true, "are": true, "this": true, "that": true, "all": true,
	"new": true, "home": true, "page": true,
}

// meaningfulWords returns the distinct words of text that can tie a title to
// a heading: at least three characters, not a stop word. Letters, digits and
// combining marks (Devanagari vowel signs) make up a word.
func meaningfulWords(text string) map[string]bool {
	words := map[string]bool{}
	spaced := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, text)
	for word := range strings.FieldsSeq(spaced) {
		if utf8.RuneCountInString(word) >= 3 && !titleStopWords[word] {
			words[word] = true
		}
	}
	return words
}

// sharesWord reports whether a and b have a meaningful word in common.
func sharesWord(a, b string) bool {
	words := meaningfulWords(a)
	for word := range meaningfulWords(b) {
		if words[word] {
			return true
		}
	}
	return false
}

// sameName reports whether text is only the site's name: the host without
// "www." or its first label, compared letters and digits only.
func sameName(text, host string) bool {
	name := strings.TrimPrefix(host, "www.")
	label, _, _ := strings.Cut(name, ".")
	key := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, s)
	}
	normalized := key(text)
	return normalized != "" && (normalized == key(label) || normalized == key(name))
}

// pageHost returns the host of pageURL without a port, or "".
func pageHost(pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// firstNonEmpty returns the first non-empty value.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// prefixRunes returns at most n runes of s.
func prefixRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// parseLooseDate reads the date formats pages state (ISO 8601 and the common
// written forms); ok is false for anything else.
func parseLooseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02", "2 January 2006", "January 2, 2006", "2 Jan 2006", "Jan 2, 2006"}
	for _, candidate := range []string{value, prefixRunes(value, 10)} {
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, candidate); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

// countOf writes a count with the right noun: "1 list", "2 lists".
func countOf(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// pageDate is the date the page states as its freshest: modified, else
// published. "" when the page states neither.
func pageDate(ai AIPageSignals) string {
	return firstNonEmpty(ai.Modified, ai.Published)
}

// isGenericAnchor reports whether a link text says nothing about its target.
func isGenericAnchor(text string) bool {
	switch strings.ToLower(strings.Join(strings.Fields(text), " ")) {
	case "click here", "here", "read more", "learn more", "more", "this", "link", "more info", "find out more":
		return true
	}
	return false
}
