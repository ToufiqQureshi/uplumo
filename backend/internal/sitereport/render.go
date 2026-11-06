package sitereport

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strings"
	"time"
)

//go:embed report.html.tmpl
var reportTemplateText string

// defaultAccent is Uplumo's brand green, used when the workspace has no
// white-label accent.
const defaultAccent = "#65a30d"

// maxRankRows caps the keyword table so a large tracking list still fits the
// 500 KB report limit.
const maxRankRows = 200

var reportTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"num":           Number,
	"compact":       Compact,
	"pos":           Position,
	"rate":          func(v float64) string { return Percent(v * 100) },
	"delta":         delta,
	"posDelta":      positionDelta,
	"ptDelta":       pointDelta,
	"mdelta":        metricDelta,
	"mval":          metricValue,
	"date":          func(t time.Time) string { return t.Format("2 Jan 2006") },
	"short":         shortURL,
	"href":          safeHref,
	"label":         issueLabel,
	"width":         widthStyle,
	"ring":          scoreRing,
	"rankMove":      rankMove,
	"deref":         func(v *int) int { return *v },
	"deref64":       func(v *int64) int64 { return *v },
	"lineChart":     lineChart,
	"searchSeries":  searchSeries,
	"trafficSeries": trafficSeries,
	"rankRows":      func(rows []RankRow) []RankRow { return firstN(rows, maxRankRows) },
	"rankClass":     rankClass,
	"share":         func(part, total int) float64 { return float64(part) / math.Max(float64(total), 1) * 100 },
	"sub":           func(a, b int) int { return a - b },
	"ms":            func(v float64) string { return trimZero(fmt.Sprintf("%.1f", v/1000)) + " s" },
	"cls":           func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"inp":           func(v float64) string { return Number(v) + " ms" },
	"paragraphs":    paragraphs,
	"inum":          func(n int) string { return Number(float64(n)) },
	"float":         func(n int64) float64 { return float64(n) },
	"str":           func(u template.URL) string { return string(u) },
	"dict":          dict,
}).Parse(reportTemplateText))

// dict passes named values to a nested template.
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict needs key and value pairs")
	}
	out := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict key %v is not a string", pairs[i])
		}
		out[key] = pairs[i+1]
	}
	return out, nil
}

type view struct {
	Snapshot
	AccentCSS  template.CSS
	Where      string
	Highlights []Highlight
	Sections   []navItem
}

type navItem struct{ ID, Label string }

// Render turns a snapshot into one self-contained HTML document. It has no
// scripts and loads nothing, so it renders the same inside the report
// sandbox, on the public share link and in print.
func Render(s Snapshot) (string, error) {
	accent := defaultAccent
	if hexColor.MatchString(s.Accent) {
		accent = s.Accent
	}
	v := view{Snapshot: s, Highlights: firstN(Highlights(s), 6), Where: s.Project.Domain}
	// #nosec G203 -- accent is a validated six-digit hex color.
	v.AccentCSS = template.CSS(accent)
	if v.Where == "" {
		v.Where = s.Project.Name
	}
	v.Sections = append(v.Sections, navItem{"summary", "Summary"})
	if s.Search != nil {
		v.Sections = append(v.Sections, navItem{"search", "Google Search"})
	}
	if s.Traffic != nil {
		v.Sections = append(v.Sections, navItem{"traffic", "Organic traffic"})
	}
	if s.Rankings != nil {
		v.Sections = append(v.Sections, navItem{"rankings", "Rankings"})
	}
	if s.Audit != nil {
		v.Sections = append(v.Sections, navItem{"audit", "Site health"})
	}
	if s.Backlinks != nil {
		v.Sections = append(v.Sections, navItem{"backlinks", "Backlinks"})
	}
	if s.NextSteps != "" {
		v.Sections = append(v.Sections, navItem{"next", "Next steps"})
	}
	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, v); err != nil {
		return "", fmt.Errorf("render site report: %w", err)
	}
	return buf.String(), nil
}

// Delta is a change badge: its text and whether the change is good.
type Delta struct{ Text, Tone string }

func delta(current, previous float64) Delta {
	if previous == 0 {
		return Delta{}
	}
	change := (current - previous) / previous * 100
	return toned(change, Percent(math.Abs(change)), false)
}

func positionDelta(current, previous float64) Delta {
	if current == 0 || previous == 0 {
		return Delta{}
	}
	diff := current - previous
	return toned(diff, Position(math.Abs(diff)), true)
}

func pointDelta(current, previous float64) Delta {
	diff := (current - previous) * 100
	return toned(diff, trimZero(fmt.Sprintf("%.1f", math.Abs(diff)))+" pts", false)
}

func metricDelta(m Metric) Delta {
	if m.Value == nil || m.Previous == nil {
		return Delta{}
	}
	return delta(*m.Value, *m.Previous)
}

func metricValue(m Metric) float64 {
	if m.Value == nil {
		return 0
	}
	return *m.Value
}

// toned marks a change up or down; lowerIsBetter flips the tone for
// positions, where a smaller number is a better ranking.
func toned(change float64, magnitude string, lowerIsBetter bool) Delta {
	if math.Abs(change) < 0.05 {
		return Delta{Text: "No change", Tone: "flat"}
	}
	good := change > 0
	if lowerIsBetter {
		good = !good
	}
	arrow := "▲ "
	if change < 0 {
		arrow = "▼ "
	}
	tone := "down"
	if good {
		tone = "up"
	}
	return Delta{Text: arrow + magnitude, Tone: tone}
}

func rankMove(r RankRow) Delta {
	switch {
	case r.Position == nil && r.Previous != nil:
		return Delta{Text: "Lost", Tone: "down"}
	case r.Position != nil && r.Previous == nil:
		return Delta{Text: "New", Tone: "up"}
	}
	change := r.Change()
	if change == 0 {
		return Delta{Text: "—", Tone: "flat"}
	}
	return toned(float64(change), fmt.Sprintf("%d", abs(change)), false)
}

func rankClass(r RankRow) string {
	classes := []string{}
	if r.Position != nil && *r.Position <= 10 {
		classes = append(classes, "t10")
	}
	if r.Position == nil {
		classes = append(classes, "nr")
	}
	switch move := rankMove(r); move.Tone {
	case "up":
		classes = append(classes, "mu")
	case "down":
		classes = append(classes, "md")
	}
	return strings.Join(classes, " ")
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// shortURL shows a page as its path, which is what a reader recognizes.
func shortURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return path
}

// safeHref keeps only http(s) links; anything else renders as plain text.
func safeHref(raw string) template.URL {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return ""
	}
	// #nosec G203 -- scheme and host are validated above.
	return template.URL(u.String())
}

// issueLabel turns an audit issue code into words: "missing-h1" -> "Missing h1".
func issueLabel(code string) string {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(code))
	if len(words) == 0 {
		return code
	}
	for i, w := range words {
		switch strings.ToLower(w) {
		case "h1", "html", "http", "https", "hsts", "url", "urls", "js", "css":
			words[i] = strings.ToUpper(w)
		}
	}
	first := []rune(words[0])
	if len(first) > 0 && first[0] >= 'a' && first[0] <= 'z' {
		first[0] -= 'a' - 'A'
	}
	words[0] = string(first)
	return strings.Join(words, " ")
}

func widthStyle(percent float64) template.CSS {
	percent = math.Max(0, math.Min(100, percent))
	// #nosec G203 -- a formatted number, not user input.
	return template.CSS(fmt.Sprintf("width:%.2f%%", percent))
}

func paragraphs(text string) []string {
	out := []string{}
	for _, part := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

type series struct {
	Values []float64
	Dates  []string
}

func searchSeries(days []SearchDay, impressions bool) series {
	out := series{}
	for _, d := range days {
		v := d.Clicks
		if impressions {
			v = d.Impressions
		}
		out.Values = append(out.Values, v)
		out.Dates = append(out.Dates, d.Date)
	}
	return out
}

func trafficSeries(days []TrafficDay) series {
	out := series{}
	for _, d := range days {
		out.Values = append(out.Values, d.Sessions)
		out.Dates = append(out.Dates, d.Date)
	}
	return out
}

// lineChart draws an area chart as inline SVG. Each day is a hover target
// with a native tooltip, so the chart is readable without scripts.
func lineChart(id string, values []float64, dates []string, unit string) template.HTML {
	if len(values) < 2 {
		return ""
	}
	const w, h, left, right, top, bottom = 720.0, 220.0, 48.0, 12.0, 12.0, 28.0
	maxV := 0.0
	for _, v := range values {
		maxV = math.Max(maxV, v)
	}
	maxV = niceCeil(maxV)
	x := func(i int) float64 { return left + float64(i)*(w-left-right)/float64(len(values)-1) }
	y := func(v float64) float64 { return top + (h-top-bottom)*(1-v/maxV) }
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="chart" viewBox="0 0 %.0f %.0f" role="img" aria-label="%s per day">`, w, h, template.HTMLEscapeString(unit))
	fmt.Fprintf(&b, `<defs><linearGradient id="g-%s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" class="s0"/><stop offset="1" class="s1"/></linearGradient></defs>`, id)
	for i := 0; i <= 4; i++ {
		v := maxV * float64(i) / 4
		fmt.Fprintf(&b, `<line class="grid" x1="%.1f" x2="%.1f" y1="%.1f" y2="%.1f"/><text class="axis" x="%.1f" y="%.1f" text-anchor="end">%s</text>`, left, w-right, y(v), y(v), left-8, y(v)+4, Compact(v))
	}
	var line strings.Builder
	for i, v := range values {
		cmd := "L"
		if i == 0 {
			cmd = "M"
		}
		fmt.Fprintf(&line, "%s%.1f %.1f ", cmd, x(i), y(v))
	}
	fmt.Fprintf(&b, `<path d="%sL%.1f %.1f L%.1f %.1f Z" fill="url(#g-%s)"/>`, line.String(), x(len(values)-1), y(0), x(0), y(0), id)
	fmt.Fprintf(&b, `<path class="line" d="%s"/>`, line.String())
	step := (w - left - right) / float64(len(values)-1)
	for i, v := range values {
		label := dates[i]
		if t, err := time.Parse(day, label); err == nil {
			label = t.Format("Mon 2 Jan")
		}
		fmt.Fprintf(&b, `<g class="pt"><rect x="%.1f" y="%.1f" width="%.1f" height="%.1f"/><line class="cursor" x1="%.1f" x2="%.1f" y1="%.1f" y2="%.1f"/><circle cx="%.1f" cy="%.1f" r="4"/><title>%s: %s %s</title></g>`,
			x(i)-step/2, top, step, h-top-bottom, x(i), x(i), top, h-bottom, x(i), y(v), template.HTMLEscapeString(label), Number(v), template.HTMLEscapeString(unit))
	}
	for _, i := range []int{0, len(values) / 2, len(values) - 1} {
		label := dates[i]
		if t, err := time.Parse(day, label); err == nil {
			label = t.Format("2 Jan")
		}
		anchor := "middle"
		if i == 0 {
			anchor = "start"
		} else if i == len(values)-1 {
			anchor = "end"
		}
		fmt.Fprintf(&b, `<text class="axis" x="%.1f" y="%.1f" text-anchor="%s">%s</text>`, x(i), h-8, anchor, template.HTMLEscapeString(label))
	}
	b.WriteString(`</svg>`)
	// #nosec G203 -- built from numbers and escaped date labels only.
	return template.HTML(b.String())
}

// niceCeil rounds an axis maximum up to 1, 2, 2.5 or 5 times a power of ten.
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if v <= m*exp {
			return m * exp
		}
	}
	return 10 * exp
}

// scoreRing draws a Lighthouse score dial. Colors follow Lighthouse's own
// bands: 90+ good, 50-89 needs work, under 50 poor.
func scoreRing(score *int) template.HTML {
	if score == nil {
		return `<div class="ring na">–</div>`
	}
	s := max(0, min(100, *score))
	band := "poor"
	switch {
	case s >= 90:
		band = "good"
	case s >= 50:
		band = "fair"
	}
	const r = 34.0
	circ := 2 * math.Pi * r
	// #nosec G203 -- built from a bounded integer only.
	return template.HTML(fmt.Sprintf(`<svg class="ring %s" viewBox="0 0 80 80" role="img" aria-label="Score %d of 100"><circle class="track" cx="40" cy="40" r="%.0f"/><circle class="value" cx="40" cy="40" r="%.0f" stroke-dasharray="%.2f %.2f" transform="rotate(-90 40 40)"/><text x="40" y="46" text-anchor="middle">%d</text></svg>`,
		band, s, r, r, circ*float64(s)/100, circ, s))
}
