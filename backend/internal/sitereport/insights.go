package sitereport

import (
	"fmt"
	"math"
	"strings"
)

// Highlight is one data-backed takeaway shown at the top of the report. Every
// highlight comes from a number in the snapshot; none is generic advice.
type Highlight struct {
	Tone   string // "up", "down" or "flat"
	Title  string
	Detail string
	Anchor string
}

// Highlights lists the period's main movements, strongest signals first.
func Highlights(s Snapshot) []Highlight {
	var out []Highlight
	period := fmt.Sprintf("the previous %d days", s.Period.Days)
	if a := s.Analytics; a != nil && len(a.LeadActions) > 0 {
		action := a.LeadActions[0]
		out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("%s recorded %d times", action.Name, action.Count), Detail: "Configured lead action in Uplumo Analytics.", Anchor: "analytics"})
	}
	if t := s.Traffic; t != nil && t.KeyEvents.Value != nil {
		switch {
		case positive(t.KeyEvents.Previous):
			out = append(out, changeHighlight("Organic key events", *t.KeyEvents.Value, *t.KeyEvents.Previous, period, "traffic"))
		case positive(t.KeyEvents.Value):
			out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("%s key events from organic visits", Number(*t.KeyEvents.Value)), Anchor: "traffic"})
		}
	}
	if q := s.Search; q != nil {
		if q.HasPrevious && q.Previous.Clicks > 0 {
			out = append(out, changeHighlight("Search clicks", q.Current.Clicks, q.Previous.Clicks, period, "search"))
		} else if q.Current.Clicks > 0 {
			out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("%s clicks from Google Search", Number(q.Current.Clicks)), Detail: fmt.Sprintf("From %s impressions.", Number(q.Current.Impressions)), Anchor: "search"})
		}
		if q.HasPrevious && q.Previous.Position > 0 && q.Current.Position > 0 && math.Abs(q.Current.Position-q.Previous.Position) >= 0.5 {
			tone, verb := "up", "improved"
			if q.Current.Position > q.Previous.Position {
				tone, verb = "down", "dropped"
			}
			out = append(out, Highlight{Tone: tone, Title: "Average position " + verb, Detail: fmt.Sprintf("From %s to %s in Google Search.", Position(q.Previous.Position), Position(q.Current.Position)), Anchor: "search"})
		}
		if n := len(q.Opportunities); n > 0 {
			out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("%d %s close to the top results", n, plural(n, "query", "queries")), Detail: fmt.Sprintf("Ranking 5 to 20 with %s impressions between them.", Number(sumImpressions(q.Opportunities))), Anchor: "opportunities"})
		}
	}
	if t := s.Traffic; t != nil && t.Sessions.Value != nil {
		if t.Sessions.Previous != nil && *t.Sessions.Previous > 0 {
			out = append(out, changeHighlight("Organic visits", *t.Sessions.Value, *t.Sessions.Previous, period, "traffic"))
		}
	}
	if a := s.AI; a != nil {
		out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("Cited in %d of %d tracked AI answers", a.Cited, a.Answered), Detail: "Latest saved AI visibility check.", Anchor: "ai-visibility"})
	}
	if l := s.Local; l != nil && l.Check != nil && l.Check.MapsAvailable() && l.Check.YourRank > 0 {
		out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("Google Maps position %d", l.Check.YourRank), Detail: l.Check.SearchTerm, Anchor: "local-visibility"})
	}
	if r := s.Rankings; r != nil {
		out = append(out, Highlight{Tone: "flat", Title: fmt.Sprintf("%d of %d keywords in the top 10", r.Top10, r.Total), Detail: fmt.Sprintf("%d in the top 3.", r.Top3), Anchor: "rankings"})
		if r.Improved+r.Declined > 0 {
			tone := "flat"
			if r.Improved > r.Declined {
				tone = "up"
			} else if r.Declined > r.Improved {
				tone = "down"
			}
			out = append(out, Highlight{Tone: tone, Title: fmt.Sprintf("%d %s moved up, %d moved down", r.Improved, plural(r.Improved, "keyword", "keywords"), r.Declined), Anchor: "rankings"})
		}
	}
	if a := s.Audit; a != nil {
		critical := 0
		for _, issue := range a.Issues {
			if issue.Severity == "critical" {
				critical++
			}
		}
		if critical > 0 {
			out = append(out, Highlight{Tone: "down", Title: fmt.Sprintf("%d critical %s to fix", critical, plural(critical, "issue", "issues")), Detail: fmt.Sprintf("Found on %s of %s crawled pages.", Number(float64(a.Critical)), Number(float64(a.PagesCrawled))), Anchor: "audit"})
		} else {
			out = append(out, Highlight{Tone: "up", Title: "No critical site issues", Detail: fmt.Sprintf("Across %s crawled pages.", Number(float64(a.PagesCrawled))), Anchor: "audit"})
		}
	}
	if b := s.Backlinks; b != nil && b.NewReferringDomains != nil && b.LostReferringDomains != nil && *b.NewReferringDomains+*b.LostReferringDomains > 0 {
		tone := "flat"
		switch {
		case *b.NewReferringDomains > *b.LostReferringDomains:
			tone = "up"
		case *b.NewReferringDomains < *b.LostReferringDomains:
			tone = "down"
		}
		out = append(out, Highlight{Tone: tone, Title: fmt.Sprintf("%d new linking %s", *b.NewReferringDomains, plural(int(*b.NewReferringDomains), "site", "sites")), Detail: fmt.Sprintf("%d lost.", *b.LostReferringDomains), Anchor: "backlinks"})
	}
	return out
}

func priorities(s Snapshot) []Highlight {
	var out []Highlight
	if s.Audit != nil {
		for _, issue := range s.Audit.Issues {
			if issue.Severity == "critical" && len(issue.Examples) > 0 {
				out = append(out, Highlight{Title: "Fix " + strings.ToLower(issueLabel(issue.Type)), Detail: fmt.Sprintf("%d affected pages, including %s", issue.Pages, issue.Examples[0]), Anchor: "audit"})
				break
			}
		}
	}
	if s.Search != nil && len(s.Search.Opportunities) > 0 {
		row := s.Search.Opportunities[0]
		out = append(out, Highlight{Title: fmt.Sprintf("Improve the page targeting %q", row.Query), Detail: fmt.Sprintf("%s · position %s · %s impressions", row.Page, Position(row.Position), Number(row.Impressions)), Anchor: "opportunities"})
	}
	if s.Local != nil && s.Local.Check != nil {
		for _, fix := range s.Local.Check.Fixes {
			if fix.Status == "missing" || fix.Status == "behind" {
				out = append(out, Highlight{Title: fix.Title, Detail: fix.Detail, Anchor: "local-visibility"})
				break
			}
		}
	}
	if s.AI != nil {
		for _, answer := range s.AI.Evidence {
			if answerPriority(answer) != 1 {
				continue
			}
			for _, source := range answer.Sources {
				if source.Brand != "" {
					out = append(out, Highlight{Title: "Review the citation gap for " + answer.Engine, Detail: fmt.Sprintf("%q cited %s.", answer.Prompt, source.Brand), Anchor: "ai-visibility"})
					return firstN(out, 3)
				}
			}
		}
	}
	return firstN(out, 3)
}

func changeHighlight(label string, current, previous float64, period, anchor string) Highlight {
	change := (current - previous) / previous * 100
	h := Highlight{Tone: "flat", Title: label + " held steady", Anchor: anchor,
		Detail: fmt.Sprintf("%s vs %s in %s.", Number(current), Number(previous), period)}
	switch {
	case change >= 1:
		h.Tone, h.Title = "up", fmt.Sprintf("%s up %s", label, Percent(change))
	case change <= -1:
		h.Tone, h.Title = "down", fmt.Sprintf("%s down %s", label, Percent(-change))
	}
	return h
}

// summary is the report list's preview text: the client message when there
// is one, else the top highlights.
func summary(s Snapshot) string {
	if s.Note != "" {
		return truncate(s.Note, 2_000)
	}
	lines := []string{}
	for _, h := range firstN(Highlights(s), 4) {
		lines = append(lines, strings.TrimSpace(h.Title+". "+h.Detail))
	}
	if len(lines) == 0 {
		where := s.Project.Domain
		if where == "" {
			where = s.Project.Name
		}
		return "Site report for " + where + "."
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

func sumImpressions(rows []Opportunity) float64 {
	total := 0.0
	for _, r := range rows {
		total += r.Impressions
	}
	return total
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Number formats a count with thousands separators: 12,345.
func Number(v float64) string {
	n := int64(math.Round(v))
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}

// Compact formats large counts for tiles: 12.3K, 4.5M.
func Compact(v float64) string {
	abs := math.Abs(v)
	switch {
	case abs >= 1_000_000:
		return trimZero(fmt.Sprintf("%.1f", v/1_000_000)) + "M"
	case abs >= 10_000:
		return trimZero(fmt.Sprintf("%.1f", v/1_000)) + "K"
	}
	return Number(v)
}

// Percent formats a percentage with one decimal under 10: 4.5%, 23%.
func Percent(v float64) string {
	if math.Abs(v) < 10 {
		return trimZero(fmt.Sprintf("%.1f", v)) + "%"
	}
	return Number(v) + "%"
}

// Position formats an average ranking position: 12.3.
func Position(v float64) string { return trimZero(fmt.Sprintf("%.1f", v)) }

func trimZero(s string) string { return strings.TrimSuffix(s, ".0") }
