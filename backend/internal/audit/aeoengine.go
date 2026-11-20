package audit

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// The AEO engine of the free audit: how ready one page is for AI assistants
// (ChatGPT, Perplexity, Claude, Copilot, Google AI Overviews) to read, trust
// and quote. It reads the snapshot only; no provider or model is called.
//
// It follows the answer-engine checklists and citation studies of 2026:
// crawler access first, then answers a model can lift (question headings,
// 40-60 word answer capsules, clear definitions, lists and tables), then
// evidence (figures, authoritative sources, quotes, author and date) and
// entity clarity (who the business is, schema). llms.txt is a weak signal and
// weighs little.

// Category IDs of the AEO engine.
const (
	aeoAccess = "access"
	aeoAnswer = "answer"
	aeoTrust  = "trust"
	aeoEntity = "entity"
)

// aeoCategories are the AEO engine's categories, in the order the report
// shows them, with their weights in the score.
var aeoCategories = []categoryDef{
	{ID: aeoAccess, Name: "AI access", Weight: 25},
	{ID: aeoAnswer, Name: "Answers AI can quote", Weight: 30},
	{ID: aeoTrust, Name: "Evidence and trust", Weight: 25},
	{ID: aeoEntity, Name: "Entity clarity", Weight: 20},
}

// aeoSevereCap caps the AEO score when a critical check fails: an AI search
// crawler is blocked, the text is missing without JavaScript, or the page
// hides instructions aimed at AI.
const aeoSevereCap = 39

// Thresholds of the AEO checks.
const (
	aeoQuestionGood  = 3
	aeoCapsuleGood   = 2
	aeoListsGood     = 2
	aeoFactsGood     = 3
	aeoAuthorityGood = 2
	// freeSameAsGood is how many profile links (sameAs) the business schema needs.
	freeSameAsGood = 2
	// aeoFreshMonths and aeoWarnMonths bound the stated date: within the first
	// passes, within the second warns, older (or no date) fails.
	aeoFreshMonths = 12
	aeoWarnMonths  = 24
	// aeoCapsuleMinWords and aeoCapsuleMaxWords bound an answer capsule: the
	// paragraph right after a question heading that answers it.
	aeoCapsuleMinWords = 25
	aeoCapsuleMaxWords = 80
	// freeChunkReadyRatio is the share of sections that must stand alone.
	freeChunkReadyRatio = 0.3
	// freeMaxSnippet bounds the evidence text shown in a check.
	freeMaxSnippet = 80
)

// aeoChecks builds the AEO engine's checks in the order the report shows them.
func aeoChecks(snap PageSnapshot) []PageCheck {
	page := snap.Page
	ai := page.AI
	var checks []PageCheck
	add := func(check PageCheck) { checks = append(checks, check) }

	// AI access.
	add(aiCrawlerCheck(snap.Origin, snap.Robots))
	add(noJavaScriptCheck(page))
	add(snippetCheck(ai))
	add(hiddenInstructionsCheck(ai))
	add(llmsCheck(snap.Origin, snap.LLMs))

	// Answers AI can quote.
	add(questionHeadingsCheck(ai))
	add(answerCapsulesCheck(ai))
	add(definitionCheck(ai))
	add(chunkReadyCheck(ai, page.WordCount))
	add(listsTablesCheck(ai))

	// Evidence and trust.
	add(factsCheck(ai))
	add(authorityCheck(ai))
	add(quotesCheck(ai))
	add(authorDateCheck(ai))
	add(aiFreshnessCheck(ai, snap.CheckedAt))
	add(aboutContactCheck(page.Links))

	// Entity clarity.
	add(businessSchemaCheck(ai))
	add(brandNameCheck(page.Title, ai, snap.Page.Technical))
	add(answerSchemaCheck(ai))
	return checks
}

// ScoreAEO scores the snapshot with the AEO engine.
func ScoreAEO(snap PageSnapshot) PageReport {
	return scoreEngine(snap, KindAEO, aeoCategories, aeoSevereCap, aeoChecks)
}

// aiCrawlerCheck is critical when an AI search crawler is blocked. Blocking
// training-only crawlers is a choice and is shown as information.
func aiCrawlerCheck(origin string, robots *string) PageCheck {
	c := newCheck("ai-crawlers", aeoAccess, sevCritical, "AI crawlers allowed",
		"ChatGPT, Perplexity, Claude and Copilot can only cite pages their crawlers may read.")
	var search, training []string
	for _, crawler := range blockedAICrawlers(origin, robots) {
		if IsAISearchBot(crawler) {
			search = append(search, crawler)
		} else {
			training = append(training, crawler)
		}
	}
	switch {
	case len(search) > 0:
		c.set(CheckFail, "robots.txt blocks: "+strings.Join(search, ", ")+".",
			"Remove the Disallow rules for these user agents in robots.txt, and check your firewall does not block them.")
	case len(training) > 0:
		c.set(CheckInfo, "AI search crawlers may read the site. Training-only crawlers are blocked ("+strings.Join(training, ", ")+"); that is a choice, not an error.", "")
	default:
		c.set(CheckPass, "robots.txt lets the AI search crawlers in.", "")
	}
	return c
}

// noJavaScriptCheck is critical: most AI crawlers read only the HTML.
func noJavaScriptCheck(page CrawledPageResult) PageCheck {
	c := newCheck("no-javascript", aeoAccess, sevCritical, "Readable without JavaScript",
		"Most AI crawlers do not run JavaScript; they only see the HTML your server sends.")
	if page.JavaScriptShell {
		c.set(CheckFail, "The page is an empty shell until JavaScript runs.",
			"Use server-side rendering or prerendering so the text is in the first HTML response.")
		return c
	}
	c.set(CheckPass, fmt.Sprintf("%d words are in the HTML the server sends.", page.WordCount), "")
	return c
}

func snippetCheck(ai AIPageSignals) PageCheck {
	c := newCheck("snippet", aeoAccess, sevHigh, "Allowed to be quoted",
		"nosnippet or max-snippet:0 stops Google and AI answers from quoting your text.")
	if ai.SnippetBlocked {
		c.set(CheckFail, "A nosnippet rule blocks quoting.", "Remove nosnippet / max-snippet:0 unless you mean to block it.")
		return c
	}
	c.set(CheckPass, "No rule blocks quoting.", "")
	return c
}

// hiddenInstructionsCheck is critical: text meant only for AI is manipulation.
func hiddenInstructionsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("hidden-instructions", aeoAccess, sevCritical, "No hidden instructions to AI",
		"Text visitors cannot see that tells AI assistants what to say is manipulation. Search engines and AI can distrust or drop such pages.")
	if len(ai.InjectionEvidence) == 0 {
		c.set(CheckPass, "No hidden text aimed at AI assistants.", "")
		return c
	}
	sample := []rune(ai.InjectionEvidence[0].Text)
	if len(sample) > freeMaxSnippet {
		sample = sample[:freeMaxSnippet]
	}
	c.set(CheckFail, fmt.Sprintf("%d hidden instruction(s) aimed at AI assistants, for example: %q.", len(ai.InjectionEvidence), string(sample)),
		"Remove the hidden text. Write only what visitors can read.")
	return c
}

// llmsCheck is low: llms.txt is optional and few assistants read it yet.
func llmsCheck(origin string, present bool) PageCheck {
	c := newCheck("llms-txt", aeoAccess, sevLow, "llms.txt file",
		"llms.txt is an optional, plain-text map of your key pages for AI tools. Few read it yet.")
	if present {
		c.set(CheckPass, origin+"/llms.txt exists.", "")
		return c
	}
	c.set(CheckWarn, "No /llms.txt.", "Optional: add /llms.txt listing your main pages, one line each on what the page answers.")
	return c
}

// questionHeadingsCheck counts H2 and H3 headings phrased as a question.
func questionHeadingsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("question-headings", aeoAnswer, sevMedium, "Questions as headings",
		"AI assistants match a question in the user's prompt to a heading that asks the same question, then quote the answer below it.")
	const fix = "Turn the questions your buyers ask into H2 or H3 headings, and answer each one in the first lines below it."
	switch n := ai.QuestionHeadings; {
	case n >= aeoQuestionGood:
		c.set(CheckPass, fmt.Sprintf("%d question headings.", n), "")
	case n > 0:
		c.set(CheckWarn, fmt.Sprintf("%d question heading(s); %d or more match the questions buyers ask.", n, aeoQuestionGood), fix)
	default:
		c.set(CheckFail, "No headings phrased as a question.", fix)
	}
	return c
}

// answerCapsulesCheck counts paragraphs of 25-80 words that answer a question
// heading with a statement: the unit AI assistants lift as an answer.
func answerCapsulesCheck(ai AIPageSignals) PageCheck {
	c := newCheck("answer-capsules", aeoAnswer, sevMedium, "Direct answers under questions",
		"A short paragraph that answers the question in its heading, in one statement, is the piece AI assistants quote most.")
	const fix = "Under each question heading, write a 40-60 word paragraph that starts with the answer and includes one fact (a price, a time, a number)."
	switch n := ai.AnswerCapsules; {
	case n >= aeoCapsuleGood:
		c.set(CheckPass, fmt.Sprintf("%d questions have a direct answer under the heading.", n), "")
	case n == 1:
		c.set(CheckWarn, "1 question has a direct answer under the heading; two or more is better.", fix)
	default:
		c.set(CheckFail, "No question heading is followed by a direct answer paragraph.", fix)
	}
	return c
}

// definitionCheck passes when a heading is followed by a "X is ..." definition.
func definitionCheck(ai AIPageSignals) PageCheck {
	c := newCheck("definition", aeoAnswer, sevMedium, "Clear definitions",
		"Definitions in the form \"X is ...\" right under a heading are cited about twice as often as vague openings in GEO research.")
	if ai.DefinitionAfterHeading {
		c.set(CheckPass, "A heading is followed by a clear definition (\"X is ...\").", "")
		return c
	}
	c.set(CheckWarn, "No heading is followed by a clear \"X is ...\" definition.",
		"Under the main heading, define what you offer in one sentence: \"Uplumo is ...\", \"A content brief is ...\".")
	return c
}

// chunkReadyCheck: sections that stand alone are the ones assistants quote.
func chunkReadyCheck(ai AIPageSignals, words int) PageCheck {
	c := newCheck("chunk-ready", aeoAnswer, sevMedium, "Sections that stand alone",
		"AI assistants lift one section at a time. Short sections that start with a clear statement are the ones they quote.")
	const fix = "Give each H2 section a short, self-contained first paragraph that answers its heading."
	ratio := 0.0
	if ai.Sections > 0 {
		ratio = float64(ai.ChunkReadySections) / float64(ai.Sections)
	}
	switch {
	case ai.Sections == 0 && words < freeThinWords:
		c.set(CheckInfo, "Too short to split into sections.", "")
	case ai.Sections == 0 || ai.ChunkReadySections == 0:
		c.set(CheckFail, fmt.Sprintf("None of the %d sections is short and self-contained enough to quote.", ai.Sections), fix)
	case ratio >= freeChunkReadyRatio:
		c.set(CheckPass, fmt.Sprintf("%d of %d sections stand alone.", ai.ChunkReadySections, ai.Sections), "")
	default:
		c.set(CheckWarn, fmt.Sprintf("Only %d of %d sections stand alone.", ai.ChunkReadySections, ai.Sections), fix)
	}
	return c
}

// listsTablesCheck: lists and tables are easy for assistants to turn into answers.
func listsTablesCheck(ai AIPageSignals) PageCheck {
	c := newCheck("lists-tables", aeoAnswer, sevLow, "Lists and tables",
		"Lists and tables are easy to skim and easy for AI to turn into answers.")
	n := ai.Lists + ai.Tables
	found := countOf(ai.Lists, "list", "lists") + " and " + countOf(ai.Tables, "table", "tables") + "."
	switch {
	case n >= aeoListsGood:
		c.set(CheckPass, found, "")
	case n == 1:
		c.set(CheckWarn, found, "Add one more list or table, for features, prices or steps.")
	default:
		c.set(CheckFail, found, "Use a bulleted list or table for features, prices or steps.")
	}
	return c
}

// factsCheck counts prices, percentages and figures an assistant can quote.
func factsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("facts", aeoTrust, sevMedium, "Facts and numbers",
		"Prices, percentages and figures give AI something specific to quote.")
	const fix = "Add concrete facts: prices, years in business, customers served, delivery times."
	switch {
	case ai.Statistics >= aeoFactsGood:
		c.set(CheckPass, fmt.Sprintf("%d figures (prices, percentages, amounts).", ai.Statistics), "")
	case ai.Statistics > 0:
		c.set(CheckWarn, fmt.Sprintf("Only %d figure(s); three or more give AI something specific to quote.", ai.Statistics), fix)
	default:
		c.set(CheckFail, "No prices, percentages or figures.", fix)
	}
	return c
}

// authorityCheck counts links to government, academic and standards sources.
func authorityCheck(ai AIPageSignals) PageCheck {
	c := newCheck("authority-links", aeoTrust, sevMedium, "Links to sources",
		"Pages that link to the sources behind their facts look more trustworthy to AI and readers.")
	switch n := ai.AuthorityLinks; {
	case n >= aeoAuthorityGood:
		c.set(CheckPass, fmt.Sprintf("%d links to authoritative sources.", n), "")
	case n == 1:
		c.set(CheckWarn, "1 link to an authoritative source; two or more is better.",
			"Link a second fact to its source, for example a government page, a standard or a study.")
	default:
		c.set(CheckFail, "No links to government, academic or other authoritative sources.",
			"Link the facts you state to their source, for example a government page, a standard or a study.")
	}
	return c
}

// quotesCheck looks for quotations (blockquote or q): a named statement is
// something an assistant can attribute.
func quotesCheck(ai AIPageSignals) PageCheck {
	c := newCheck("quotes", aeoTrust, sevLow, "Quotations",
		"A quotation from a named person or source gives AI a statement it can attribute.")
	if ai.Quotes >= 1 {
		c.set(CheckPass, fmt.Sprintf("%d quotation(s) on the page.", ai.Quotes), "")
		return c
	}
	c.set(CheckWarn, "No quotations.", "Quote a customer, expert or source in the words they used, with their name.")
	return c
}

// authorDateCheck: a named author and a date tell AI and readers the content is real and current.
func authorDateCheck(ai AIPageSignals) PageCheck {
	c := newCheck("author-date", aeoTrust, sevMedium, "Author and date",
		"A named author and a date tell AI and readers that the content is real and current.")
	const fix = "Show who wrote the page and when it was published or last updated, and add both to your schema."
	date := pageDate(ai) != ""
	switch {
	case ai.Author && date:
		c.set(CheckPass, "Author and date are stated.", "")
	case ai.Author:
		c.set(CheckWarn, "Author is stated, but no date.", fix)
	case date:
		c.set(CheckWarn, "Date is stated, but no author.", fix)
	default:
		c.set(CheckFail, "No author and no date.", fix)
	}
	return c
}

// aiFreshnessCheck: assistants weigh recent content. Modified within 12 months
// passes, within 24 warns, older or undated fails.
func aiFreshnessCheck(ai AIPageSignals, now time.Time) PageCheck {
	c := newCheck("freshness", aeoTrust, sevMedium, "Content is recent",
		"AI assistants prefer answers that are current. A visible, recent date shows the facts were checked lately.")
	const fix = "Show the last updated date on the page and update it when the facts change."
	date := pageDate(ai)
	if date == "" {
		c.set(CheckFail, "No date on the page.", fix)
		return c
	}
	parsed, ok := parseLooseDate(date)
	switch {
	case !ok:
		c.set(CheckFail, fmt.Sprintf("The date %q could not be read.", date), "Write the date as YYYY-MM-DD.")
	case !parsed.Before(now.AddDate(0, -aeoFreshMonths, 0)):
		c.set(CheckPass, "Last dated "+date+".", "")
	case !parsed.Before(now.AddDate(0, -aeoWarnMonths, 0)):
		c.set(CheckWarn, fmt.Sprintf("Last dated %s, more than %d months ago.", date, aeoFreshMonths), fix)
	default:
		c.set(CheckFail, fmt.Sprintf("Last dated %s, more than %d months ago.", date, aeoWarnMonths), fix)
	}
	return c
}

// aboutContactCheck looks for links to the about and contact pages: an
// assistant that cannot tell who stands behind the page trusts it less.
func aboutContactCheck(links []PageLink) PageCheck {
	c := newCheck("about-contact", aeoTrust, sevMedium, "About and contact pages linked",
		"Links to who you are and how to reach you show AI and readers a real business stands behind the page.")
	const fix = "Link to your About and Contact pages from this page, for example in the header or footer."
	about, contact := false, false
	for _, link := range links {
		if !link.IsInternal {
			continue
		}
		target := strings.ToLower(link.TargetURL + " " + link.Anchor)
		about = about || strings.Contains(target, "about")
		contact = contact || strings.Contains(target, "contact")
	}
	switch {
	case about && contact:
		c.set(CheckPass, "Links to About and Contact.", "")
	case about:
		c.set(CheckWarn, "Links to About, but no Contact page.", fix)
	case contact:
		c.set(CheckWarn, "Links to Contact, but no About page.", fix)
	default:
		c.set(CheckFail, "No link to an About or Contact page.", fix)
	}
	return c
}

// businessSchemaCheck: Organization or LocalBusiness schema says who the
// business is. None fails; missing name, logo, sameAs links or contact
// details warn.
func businessSchemaCheck(ai AIPageSignals) PageCheck {
	c := newCheck("business-schema", aeoEntity, sevHigh, "Who you are (schema)",
		"Organization or LocalBusiness schema tells AI your exact name, logo, contact details and profiles.")
	const fix = "Add Organization or LocalBusiness JSON-LD with name, logo, address or phone, and at least two sameAs profile links."
	business := slices.ContainsFunc(ai.SchemaTypes, func(t string) bool { return businessTypes[strings.ToLower(t)] })
	if !business {
		found := "No Organization or LocalBusiness schema."
		if len(ai.SchemaTypes) > 0 {
			found = "Structured data found (" + strings.Join(ai.SchemaTypes, ", ") + "), but no Organization or LocalBusiness."
		}
		c.set(CheckFail, found, fix)
		return c
	}
	have := map[string]bool{}
	for _, field := range ai.BusinessFields {
		have[field] = true
	}
	var missing []string
	if !have["name"] {
		missing = append(missing, "name")
	}
	if !have["logo"] {
		missing = append(missing, "logo")
	}
	if ai.SameAs < freeSameAsGood {
		missing = append(missing, fmt.Sprintf("sameAs profile links (%d of %d)", ai.SameAs, freeSameAsGood))
	}
	if !have["telephone"] && !have["contactPoint"] && !have["address"] {
		missing = append(missing, "contact or address")
	}
	if len(missing) > 0 {
		c.set(CheckWarn, "Business schema is missing: "+strings.Join(missing, ", ")+".", fix)
		return c
	}
	c.set(CheckPass, "Business schema has name, logo, profile links and contact or address.", "")
	return c
}

// brandNameCheck: the brand named in og:site_name or the schema must appear in
// the title, so an assistant names the business the same way everywhere.
func brandNameCheck(title string, ai AIPageSignals, tech TechnicalSignals) PageCheck {
	c := newCheck("brand-name", aeoEntity, sevMedium, "Brand name consistent",
		"When the brand name in the title matches the name in the schema, assistants attribute the page to the right business.")
	const fix = "Use the same business name in the title, og:site_name and the Organization schema."
	name := firstNonEmpty(ai.BusinessName, tech.SiteName)
	switch {
	case name == "":
		c.set(CheckWarn, "No brand name in og:site_name or the schema.", fix)
	case strings.Contains(strings.ToLower(title), strings.ToLower(name)):
		c.set(CheckPass, fmt.Sprintf("The title names the brand %q.", name), "")
	default:
		c.set(CheckWarn, fmt.Sprintf("The title does not name the brand %q.", name), fix)
	}
	return c
}

// answerSchemaCheck: FAQPage, HowTo, Article and Product schema tell AI what
// kind of answer the page holds.
func answerSchemaCheck(ai AIPageSignals) PageCheck {
	c := newCheck("answer-schema", aeoEntity, sevMedium, "Answer schema for this page",
		"FAQPage, HowTo, Article and Product schema tell AI what kind of answer the page gives.")
	relevant := []string{"FAQPage", "HowTo", "Article", "NewsArticle", "BlogPosting", "Product"}
	var found []string
	for _, t := range ai.SchemaTypes {
		if slices.ContainsFunc(relevant, func(r string) bool { return strings.EqualFold(r, t) }) {
			found = append(found, t)
		}
	}
	if len(found) > 0 {
		c.set(CheckPass, "Schema: "+strings.Join(slices.Compact(slices.Sorted(slices.Values(found))), ", ")+".", "")
		return c
	}
	c.set(CheckWarn, "No FAQPage, HowTo, Article or Product schema.",
		"Add the schema that fits the page: FAQPage for questions, HowTo for steps, Article for posts, Product for what you sell.")
	return c
}
