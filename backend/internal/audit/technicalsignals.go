package audit

import (
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

// HreflangTarget keeps both the language and the resolved URL for cross-page
// checks. A target is checked only when it was actually crawled.
type HreflangTarget struct {
	Language string `json:"language"`
	URL      string `json:"url"`
}

// TechnicalSignals are extracted from the response already fetched by the
// audit. Header observations are kept separately because rendered HTML does
// not contain response headers.
type TechnicalSignals struct {
	Measured          bool
	HTMLLanguage      string
	Viewport          bool
	Titles            int
	Descriptions      int
	Canonicals        int
	InsecureForms     []string
	ImagesWithoutSize int
	Hreflangs         []HreflangTarget
}

func hstsEnabled(value string) bool {
	for _, directive := range strings.Split(value, ";") {
		key, raw, ok := strings.Cut(strings.TrimSpace(directive), "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "max-age") {
			seconds, err := strconv.ParseUint(strings.Trim(strings.TrimSpace(raw), `"`), 10, 64)
			return err == nil && seconds > 0
		}
	}
	return false
}

type technicalAccumulator struct {
	pageURL   string
	headDepth int
	signals   TechnicalSignals
}

func (a *technicalAccumulator) open(name string, attrs map[string]string) {
	if name == "head" {
		a.headDepth++
	}
	if name == "html" {
		a.signals.HTMLLanguage = strings.TrimSpace(attrs["lang"])
	}
	if a.headDepth > 0 {
		switch {
		case name == "title":
			a.signals.Titles++
		case name == "meta" && strings.EqualFold(strings.TrimSpace(attrs["name"]), "description"):
			a.signals.Descriptions++
		case name == "meta" && strings.EqualFold(strings.TrimSpace(attrs["name"]), "viewport") && strings.TrimSpace(attrs["content"]) != "":
			a.signals.Viewport = true
		case name == "link":
			rel := strings.ToLower(attrs["rel"])
			if hasSpaceSeparatedToken(rel, "canonical") {
				a.signals.Canonicals++
			}
			if hasSpaceSeparatedToken(rel, "alternate") && strings.TrimSpace(attrs["hreflang"]) != "" && len(a.signals.Hreflangs) < 100 {
				if target, ok := normalizeURL(attrs["href"], a.pageURL); ok {
					a.signals.Hreflangs = append(a.signals.Hreflangs, HreflangTarget{Language: strings.ToLower(strings.TrimSpace(attrs["hreflang"])), URL: target})
				}
			}
		}
	}
	if name == "img" && (strings.TrimSpace(attrs["width"]) == "" || strings.TrimSpace(attrs["height"]) == "") {
		a.signals.ImagesWithoutSize++
	}
	if name == "form" && attrs["action"] != "" && len(a.signals.InsecureForms) < 20 {
		base, baseErr := url.Parse(a.pageURL)
		action, actionErr := url.Parse(strings.TrimSpace(attrs["action"]))
		if baseErr == nil && actionErr == nil {
			resolved := base.ResolveReference(action)
			if strings.EqualFold(resolved.Scheme, "http") {
				a.signals.InsecureForms = append(a.signals.InsecureForms, resolved.String())
			}
		}
	}
}

func (a *technicalAccumulator) close(name string) {
	if name == "head" && a.headDepth > 0 {
		a.headDepth--
	}
}

func validHTMLLanguage(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "_") {
		return false
	}
	tag, err := language.Parse(value)
	return err == nil && tag != language.Und
}

func canonicalIsRelative(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && !u.IsAbs()
}

func normalizedCanonical(raw, pageURL string) string {
	if raw == "" {
		return ""
	}
	value, ok := normalizeURL(raw, pageURL)
	if !ok {
		return ""
	}
	return value
}
