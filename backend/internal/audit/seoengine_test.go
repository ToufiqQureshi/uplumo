package audit

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// googlebotBlockedSite blocks Googlebot from the whole site and nothing else.
const googlebotBlockedSite = "User-agent: Googlebot\nDisallow: /\n"

func TestGooglebotBlockedCapsSEOAndLeavesAEOAlone(t *testing.T) {
	allowed, _ := checkerFor(t, standardSite(goodPage))
	_, seoOpen, aeoOpen := checkBoth(t, allowed, "http://shop.test/")

	routes := standardSite(goodPage)
	routes["/robots.txt"] = plainText(googlebotBlockedSite)
	blocked, _ := checkerFor(t, routes)
	_, seoBlocked, aeoBlocked := checkBoth(t, blocked, "http://shop.test/")

	if got := statuses(seoBlocked)["googlebot"]; got != CheckFail {
		t.Fatalf("googlebot = %q, want fail", got)
	}
	if seoBlocked.Score > 49 {
		t.Fatalf("SEO score = %d, want at most 49 when Googlebot is blocked", seoBlocked.Score)
	}
	if aeoBlocked.Score != aeoOpen.Score {
		t.Fatalf("AEO score changed from %d to %d: a Googlebot rule must not touch AEO", aeoOpen.Score, aeoBlocked.Score)
	}
	if _, ok := statuses(aeoBlocked)["googlebot"]; ok {
		t.Fatalf("AEO report has a googlebot check")
	}
	// The fixture is plain HTTP, which also caps SEO. Without that failure,
	// the Googlebot rule alone must be what holds the blocked page to 49.
	if got := withoutHTTPSFailure(seoBlocked, seoCategories, 49); got > 49 {
		t.Fatalf("SEO without the HTTP failure = %d with Googlebot blocked, want at most 49", got)
	}
	if got := withoutHTTPSFailure(seoOpen, seoCategories, 49); got <= 49 {
		t.Fatalf("SEO without the HTTP failure = %d with Googlebot allowed, want above the cap", got)
	}
}

func TestGooglebotRulesAreReadForGooglebotOnly(t *testing.T) {
	cases := map[string]struct {
		robots string
		want   CheckStatus
	}{
		"no robots":          {"", CheckPass},
		"all allowed":        {"User-agent: *\nAllow: /\n", CheckPass},
		"disallow everyone":  {"User-agent: *\nDisallow: /\n", CheckFail},
		"disallow googlebot": {googlebotBlockedSite, CheckFail},
		"disallow a path":    {"User-agent: Googlebot\nDisallow: /private/\n", CheckPass},
		"disallow bingbot":   {"User-agent: bingbot\nDisallow: /\n", CheckPass},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var robots *string
			if tc.robots != "" {
				robots = &tc.robots
			}
			got := googlebotCheck("http://shop.test", robots, "http://shop.test/")
			if got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

func TestRedirectsCountHops(t *testing.T) {
	cases := map[int]CheckStatus{0: CheckPass, 1: CheckPass, 2: CheckWarn, 4: CheckWarn}
	for hops, want := range cases {
		if got := redirectsCheck(hops); got.Status != want {
			t.Errorf("%d redirects = %q, want %q", hops, got.Status, want)
		}
	}
}

func TestHreflangNeedsSelfAndXDefaultWhenUsed(t *testing.T) {
	const page = "https://shop.example/"
	cases := map[string]struct {
		targets []HreflangTarget
		want    CheckStatus
	}{
		"none":              {nil, CheckPass},
		"self and default":  {[]HreflangTarget{{"en", page}, {"x-default", page}}, CheckPass},
		"no self":           {[]HreflangTarget{{"en", "https://shop.example/en/"}, {"x-default", "https://shop.example/en/"}}, CheckWarn},
		"no x-default":      {[]HreflangTarget{{"en", page}, {"fr", "https://shop.example/fr/"}}, CheckWarn},
		"self with slash":   {[]HreflangTarget{{"en", "https://shop.example"}, {"x-default", page}}, CheckPass},
		"nothing but other": {[]HreflangTarget{{"fr", "https://shop.example/fr/"}}, CheckWarn},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := hreflangCheck(tc.targets, page); got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

func TestGenericAnchorsWarnWhenMostInternalLinksSayNothing(t *testing.T) {
	link := func(anchor string) PageLink {
		return PageLink{Anchor: anchor, IsInternal: true, TargetURL: "http://shop.test/x"}
	}
	cases := map[string]struct {
		links []PageLink
		want  CheckStatus
	}{
		"none internal":     {nil, CheckInfo},
		"descriptive":       {[]PageLink{link("Wedding cakes"), link("Prices"), link("Contact us")}, CheckPass},
		"one in three":      {[]PageLink{link("Click here"), link("Prices"), link("Contact us")}, CheckWarn},
		"half generic":      {[]PageLink{link("Read more"), link("Click here"), link("Prices"), link("Contact us")}, CheckWarn},
		"external ignored":  {[]PageLink{{Anchor: "click here", TargetURL: "https://other.test/"}, link("Prices")}, CheckPass},
		"spaces and case":   {[]PageLink{link("  LEARN   More  "), link("Prices")}, CheckWarn},
		"read more on page": {[]PageLink{link("read more"), link("read more"), link("Menu")}, CheckWarn},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := genericAnchorsCheck(tc.links); got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

// headScripts returns a page with n blocking scripts in the head and the
// given attribute on each.
func headScripts(n int, attrs string) string {
	var b strings.Builder
	b.WriteString("<html><head>")
	for i := range n {
		fmt.Fprintf(&b, `<script src="/s%d.js"%s></script>`, i, attrs)
	}
	b.WriteString("</head><body><p>x</p></body></html>")
	return b.String()
}

func TestRenderBlockingScriptsAreCountedFromTheHead(t *testing.T) {
	cases := map[string]struct {
		html string
		want CheckStatus
		n    int
	}{
		"none":            {headScripts(0, ""), CheckPass, 0},
		"three":           {headScripts(3, ""), CheckPass, 3},
		"four":            {headScripts(4, ""), CheckWarn, 4},
		"nine":            {headScripts(9, ""), CheckFail, 9},
		"all deferred":    {headScripts(9, " defer"), CheckPass, 0},
		"async":           {headScripts(5, " async"), CheckPass, 0},
		"module":          {headScripts(9, ` type="module"`), CheckPass, 0},
		"inline ignored":  {`<html><head><script>var a = 1;</script></head><body></body></html>`, CheckPass, 0},
		"body not counts": {`<html><head></head><body>` + strings.Repeat(`<script src="/x.js"></script>`, 9) + `</body></html>`, CheckPass, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tech := AnalyzeHTML(tc.html, "http://shop.test/", 200, 0, "").Technical
			if tech.RenderBlockingScripts != tc.n {
				t.Fatalf("counted %d blocking scripts, want %d", tech.RenderBlockingScripts, tc.n)
			}
			if got := renderBlockingCheck(tech); got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

func TestLazyImagesAfterTheFirstThreeAreChecked(t *testing.T) {
	image := func(attrs string) string {
		return fmt.Sprintf(`<img src="/i.jpg" alt="x" width="1" height="1"%s>`, attrs)
	}
	cases := map[string]struct {
		images string
		want   CheckStatus
		n      int
	}{
		"few eager":        {strings.Repeat(image(""), 3), CheckPass, 0},
		"ten eager extra":  {strings.Repeat(image(""), 13), CheckPass, 10},
		"eleven eager":     {strings.Repeat(image(""), 14), CheckWarn, 11},
		"all lazy":         {strings.Repeat(image(` loading="lazy"`), 20), CheckPass, 0},
		"lazy upper case":  {strings.Repeat(image(` loading="LAZY"`), 20), CheckPass, 0},
		"first three free": {strings.Repeat(image(""), 3) + strings.Repeat(image(` loading="lazy"`), 12), CheckPass, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tech := AnalyzeHTML("<html><body>"+tc.images+"</body></html>", "http://shop.test/", 200, 0, "").Technical
			if tech.EagerImagesBelowFold != tc.n {
				t.Fatalf("eager images = %d, want %d", tech.EagerImagesBelowFold, tc.n)
			}
			if got := lazyImagesCheck(tech); got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

func TestBrokenJSONLDIsCountedAndFlagged(t *testing.T) {
	good := `<script type="application/ld+json">{"@type":"Organization","name":"Demo"}</script>`
	bad := `<script type="application/ld+json">{"@type":"Organization",}</script>`
	cases := map[string]struct {
		html   string
		blocks int
		broken int
		want   CheckStatus
	}{
		"none":        {"<p>x</p>", 0, 0, CheckInfo},
		"all valid":   {good + good, 2, 0, CheckPass},
		"one broken":  {good + bad, 2, 1, CheckWarn},
		"two broken":  {bad + bad + good, 3, 2, CheckWarn},
		"three broke": {bad + bad + bad, 3, 3, CheckFail},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ai := AnalyzeHTML("<html><head>"+tc.html+"</head><body><p>x</p></body></html>", "http://shop.test/", 200, 0, "").AI
			if ai.LDJSONBlocks != tc.blocks || ai.LDJSONBroken != tc.broken {
				t.Fatalf("blocks = %d broken = %d, want %d and %d", ai.LDJSONBlocks, ai.LDJSONBroken, tc.blocks, tc.broken)
			}
			if got := structuredDataValidCheck(ai); got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

func TestSchemaTypesNeedAtLeastOne(t *testing.T) {
	if got := schemaTypesCheck(AIPageSignals{}); got.Status != CheckWarn {
		t.Fatalf("no schema = %q, want warn", got.Status)
	}
	got := schemaTypesCheck(AIPageSignals{SchemaTypes: []string{"WebSite", "Organization", "WebSite"}})
	if got.Status != CheckPass || got.Found != "Structured data: Organization, WebSite." {
		t.Fatalf("schema = %q (%s), want pass listing each type once", got.Status, got.Found)
	}
}

func TestSEOFreshnessIsEighteenMonths(t *testing.T) {
	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	cases := map[string]struct {
		date string
		want CheckStatus
	}{
		"recent iso":  {"2026-10-01", CheckPass},
		"seventeen":   {"2025-06-01", CheckPass},
		"nineteen":    {"2025-04-01", CheckWarn},
		"unreadable":  {"last spring", CheckWarn},
		"none":        {"", CheckWarn},
		"modified 1y": {"2025-11-01", CheckPass},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := seoFreshnessCheck(AIPageSignals{Modified: tc.date}, now); got.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", got.Status, got.Found, tc.want)
			}
		})
	}
}

func TestSEOChecksReadUnicodeTitles(t *testing.T) {
	// Devanagari title and heading share a word; the length counts characters.
	title := "कैफ़े पुणे: " + strings.Repeat("ताज़ा केक ", 3) + "रोज़"
	if n := utf8.RuneCountInString(title); n < freeTitleMinChars || n > freeTitleMaxChars {
		t.Fatalf("fixture title has %d characters, want 30-60", n)
	}
	heading := "ताज़ा केक पुणे में"
	if got := titleCheck(title, heading); got.Status != CheckPass {
		t.Fatalf("unicode title = %q (%s), want pass", got.Status, got.Found)
	}
	if got := titleCheck(title, "Welcome"); got.Status != CheckWarn {
		t.Fatalf("unicode title without a shared word = %q, want warn", got.Status)
	}
}

func TestSEOEngineScoresOnlyItsOwnCategories(t *testing.T) {
	checker, _ := checkerFor(t, standardSite(goodPage))
	_, seo, _ := checkBoth(t, checker, "http://shop.test/")
	ids := map[string]bool{}
	for _, category := range seo.Categories {
		ids[category.ID] = true
	}
	for _, id := range []string{seoCrawl, seoOnPage, seoPerformance, seoContent} {
		if !ids[id] {
			t.Errorf("SEO categories = %v, missing %s", ids, id)
		}
	}
	for _, check := range seo.Checks {
		if !ids[check.Category] {
			t.Errorf("SEO check %s in unknown category %q", check.ID, check.Category)
		}
	}
	if seo.Kind != KindSEO {
		t.Fatalf("kind = %q", seo.Kind)
	}
}
