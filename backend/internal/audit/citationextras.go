package audit

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxTopicLinks      = 100
	maxTopicHeadings   = 30
	maxEvidence        = 5
	maxEvidenceChars   = 200
	maxParagraphBytes  = 16 * 1024
	maxCitationNesting = 256
	maxStyleBytes      = 64 * 1024
)

var (
	snippetDirective  = regexp.MustCompile(`(?i)(?:^|[^a-z-])nosnippet(?:$|[^a-z-])|(?:^|[^a-z-])max-snippet\s*:\s*0(?:$|[^0-9])`)
	hiddenStyle       = regexp.MustCompile(`(?i)(?:display\s*:\s*none|visibility\s*:\s*hidden|opacity\s*:\s*0(?:\s|;|$)|font-size\s*:\s*0(?:px|pt|em|rem)?(?:\s|;|$))`)
	hiddenClassRule   = regexp.MustCompile(`(?s)\.([a-zA-Z_][a-zA-Z0-9_-]*)\s*\{([^}]*)\}`)
	injectionText     = regexp.MustCompile(`(?i)\bignore\s+(?:all\s+)?(?:previous|prior|earlier|above)\s+instructions\b`)
	definition        = regexp.MustCompile(`(?i)^[\pL\pN][\pL\pN\s'’,-]{0,79}\s+is\s+\S`)
	inYear            = regexp.MustCompile(`(?i)\bin\s+(?:19|20)[0-9]{2}\b`)
	copyrightYears    = regexp.MustCompile(`(?i)(?:©|copyright)\s*(?:\(c\)\s*)?(?:19|20)[0-9]{2}(?:\s*[-–—]\s*(?:19|20)[0-9]{2})?`)
	yearPattern       = regexp.MustCompile(`(?:19|20)[0-9]{2}`)
	versionPattern    = regexp.MustCompile(`(?i)\b(?:version|ver\.?|v)\s*[0-9]+\.[0-9]+(?:\.[0-9]+)?\b`)
	hiddenInstruction = regexp.MustCompile(`(?i)\b(?:ignore|disregard|forget)\s+(?:all\s+)?(?:previous|prior|earlier|above)\s+instructions\b|\b(?:assistant|language model|ai agent)\s*[:,]\s*(?:always|never|recommend|prioritize|ignore|do not)\b`)
)

func blocksSnippet(directives string) bool { return snippetDirective.MatchString(directives) }

type citationFrame struct {
	tag                              string
	body, ignored, hidden, noSnippet bool
	mainContent                      bool
}

// citationAccumulator examines the same HTML tokens as the page analyzer. It
// keeps only bounded evidence, headings and one paragraph, never a second DOM.
type citationAccumulator struct {
	pageURL                              string
	stack                                []citationFrame
	overflowDepth                        int
	styleOpen                            bool
	styleText                            strings.Builder
	hiddenText                           strings.Builder
	hiddenClasses                        map[string]bool
	paragraph                            strings.Builder
	paragraphOpen, paragraphAfterHeading bool
	awaitingParagraph                    bool
	heading                              strings.Builder
	headingLevel                         int
	sectionOpen                          bool
	sectionWords, sections, chunkReady   int
	textWords, noSnippetWords            int
	capsule, defined, snippetBlocked     bool
	authorityLinks                       int
	headings                             []string
	topicLinks                           []string
	topicLinkSet                         map[string]bool
	evidence                             []CitationEvidence
}

func (c *citationAccumulator) current() citationFrame {
	if len(c.stack) == 0 {
		return citationFrame{}
	}
	return c.stack[len(c.stack)-1]
}

func (c *citationAccumulator) addEvidence(kind, value string) {
	value = strings.TrimSpace(collapseWhitespace(value))
	if value == "" || len(c.evidence) >= maxEvidence {
		return
	}
	value = truncateChars(value, maxEvidenceChars)
	for _, existing := range c.evidence {
		if existing.Kind == kind && existing.Text == value {
			return
		}
	}
	c.evidence = append(c.evidence, CitationEvidence{Kind: kind, Text: value})
}

func (c *citationAccumulator) open(name string, attributes map[string]string, selfClosing bool) {
	if c.overflowDepth > 0 || len(c.stack) >= maxCitationNesting {
		if !selfClosing && !voidHTMLTag(name) {
			c.overflowDepth++
		}
		c.addEvidence("Analysis limit", "The HTML is nested too deeply to inspect every element.")
		return
	}
	parent := c.current()
	if name == "style" {
		c.styleOpen = true
		c.styleText.Reset()
	}
	frame := parent
	frame.tag = name
	if name == "main" || name == "article" {
		frame.mainContent = true
	}
	if name == "body" {
		frame.body = true
	}
	if _, ok := chromeTags[name]; ok && (name != "header" || !parent.mainContent) {
		frame.ignored = true
	}
	if _, ok := nonContentTags[name]; ok {
		frame.ignored = true
	}
	if _, ok := attributes["hidden"]; ok {
		frame.hidden = true
	}
	if hiddenStyle.MatchString(attributes["style"]) {
		frame.hidden = true
	}
	for _, class := range strings.Fields(attributes["class"]) {
		if c.hiddenClasses[class] {
			frame.hidden = true
		}
	}
	if _, ok := attributes["data-nosnippet"]; ok {
		frame.noSnippet = true
	}
	if name == "meta" && (strings.EqualFold(attributes["name"], "robots") || strings.EqualFold(attributes["name"], "googlebot")) {
		c.snippetBlocked = c.snippetBlocked || blocksSnippet(attributes["content"])
	}
	for key, value := range attributes {
		if strings.HasPrefix(key, "data-ai-") || strings.HasPrefix(key, "data-prompt-") {
			c.addEvidence("AI instruction attribute", key+"="+value)
		}
	}
	active := frame.body && !frame.ignored && !frame.hidden
	if active && name == "a" {
		if authoritativeSource(attributes["href"], c.pageURL) {
			c.authorityLinks++
		}
		href := strings.TrimSpace(attributes["href"])
		if target, ok := normalizeURL(href, c.pageURL); href != "" && !strings.HasPrefix(href, "#") && ok && isSameOrigin(target, c.pageURL) && len(c.topicLinks) < maxTopicLinks {
			if c.topicLinkSet == nil {
				c.topicLinkSet = map[string]bool{}
			}
			if !c.topicLinkSet[target] {
				c.topicLinkSet[target] = true
				c.topicLinks = append(c.topicLinks, target)
			}
		}
	}
	if active && name == "p" {
		c.closeParagraph()
		c.paragraphOpen = true
		c.paragraphAfterHeading = c.awaitingParagraph
		c.awaitingParagraph = false
	}
	if active {
		if level, ok := headingLevels[name]; ok {
			c.closeParagraph()
			c.closeSection()
			c.sectionOpen = true
			c.heading.Reset()
			c.headingLevel = level
			c.awaitingParagraph = level <= 2
		}
	}
	if !selfClosing && !voidHTMLTag(name) {
		c.stack = append(c.stack, frame)
	}
}

// Joiners between non-Latin letters and marks are normal in Indic and other
// scripts. Isolated joiners and invisible controls still merit review.
func suspiciousInvisibleText(value string) bool {
	for i, r := range value {
		switch r {
		case '\u200b', '\u2060', '\ufeff', '\u202a', '\u202b', '\u202d', '\u202e':
			return true
		case '\u200c', '\u200d':
			before, _ := utf8.DecodeLastRuneInString(value[:i])
			after, _ := utf8.DecodeRuneInString(value[i+utf8.RuneLen(r):])
			if before <= 127 || after <= 127 || !unicode.IsLetter(before) && !unicode.IsMark(before) || !unicode.IsLetter(after) && !unicode.IsMark(after) {
				return true
			}
		}
	}
	return false
}

func voidHTMLTag(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

func (c *citationAccumulator) text(value string) {
	if c.styleOpen {
		if c.styleText.Len() < maxStyleBytes {
			c.styleText.WriteString(value[:min(len(value), maxStyleBytes-c.styleText.Len())])
		}
		return
	}
	frame := c.current()
	if !frame.body || frame.ignored {
		return
	}
	if frame.hidden {
		if c.hiddenText.Len() < maxParagraphBytes {
			c.hiddenText.WriteByte(' ')
			c.hiddenText.WriteString(value[:min(len(value), maxParagraphBytes-c.hiddenText.Len())])
		}
		return
	}
	if injectionText.MatchString(value) {
		c.addEvidence("Instruction in page text", value)
	}
	if suspiciousInvisibleText(value) {
		c.addEvidence("Invisible characters", value)
	}
	words := len(strings.Fields(value))
	c.textWords += words
	if frame.noSnippet {
		c.noSnippetWords += words
	}
	if c.sectionOpen && c.headingLevel == 0 {
		c.sectionWords += words
	}
	if c.paragraphOpen && c.paragraph.Len() < maxParagraphBytes {
		c.paragraph.WriteString(value[:min(len(value), maxParagraphBytes-c.paragraph.Len())])
	}
	if c.headingLevel > 0 && c.heading.Len() < maxEvidenceChars*4 {
		c.heading.WriteString(value[:min(len(value), maxEvidenceChars*4-c.heading.Len())])
	}
}

func (c *citationAccumulator) comment(value string) {
	if injectionText.MatchString(value) {
		c.addEvidence("Instruction in HTML comment", value)
	}
}

func (c *citationAccumulator) close(name string) {
	if c.overflowDepth > 0 {
		c.overflowDepth--
		return
	}
	if name == "style" && c.styleOpen {
		for _, rule := range hiddenClassRule.FindAllStringSubmatch(c.styleText.String(), 256) {
			if hiddenStyle.MatchString(rule[2]) {
				if c.hiddenClasses == nil {
					c.hiddenClasses = map[string]bool{}
				}
				c.hiddenClasses[rule[1]] = true
			}
		}
		c.styleOpen = false
	}
	if len(c.stack) > 0 && c.stack[len(c.stack)-1].tag == name && c.stack[len(c.stack)-1].hidden && (len(c.stack) < 2 || !c.stack[len(c.stack)-2].hidden) {
		if hiddenInstruction.MatchString(c.hiddenText.String()) {
			c.addEvidence("Hidden instruction", c.hiddenText.String())
		}
		c.hiddenText.Reset()
	}
	if name == "p" {
		c.closeParagraph()
	}
	if level, ok := headingLevels[name]; ok && level == c.headingLevel {
		if level <= 2 && len(c.headings) < maxTopicHeadings {
			value := truncateChars(strings.TrimSpace(collapseWhitespace(c.heading.String())), maxEvidenceChars)
			if value != "" {
				c.headings = append(c.headings, value)
			}
		}
		c.headingLevel = 0
	}
	for i := len(c.stack) - 1; i >= 0; i-- {
		if c.stack[i].tag == name {
			c.stack = c.stack[:i]
			break
		}
	}
}

func (c *citationAccumulator) closeParagraph() {
	if !c.paragraphOpen {
		return
	}
	text := strings.TrimSpace(collapseWhitespace(c.paragraph.String()))
	if c.paragraphAfterHeading && definition.MatchString(text) {
		c.defined = true
	}
	words := strings.Fields(text)
	if len(words) >= 30 && len(words) <= 120 && startsWithStatement(text) && hasConcreteFact(words) {
		c.capsule = true
	}
	if injectionText.MatchString(text) {
		c.addEvidence("Instruction in page text", text)
	}
	c.paragraph.Reset()
	c.paragraphOpen = false
}

func (c *citationAccumulator) closeSection() {
	if !c.sectionOpen {
		return
	}
	c.sections++
	if c.sectionWords >= 100 && c.sectionWords <= 150 {
		c.chunkReady++
	}
	c.sectionWords = 0
}

func startsWithStatement(text string) bool {
	first := text
	if i := strings.IndexAny(text, ".!?"); i >= 0 {
		first = text[:i+1]
	}
	return !strings.Contains(first, "?") && !isQuestionHeading(first)
}

func hasConcreteFact(words []string) bool {
	for i, word := range words {
		if strings.IndexFunc(word, unicode.IsDigit) >= 0 {
			return true
		}
		word = strings.TrimLeftFunc(word, func(r rune) bool { return !unicode.IsLetter(r) })
		if i > 0 && word != "" {
			r, _ := utf8.DecodeRuneInString(word)
			if unicode.IsUpper(r) {
				return true
			}
		}
	}
	return false
}

func authoritativeSource(raw, pageURL string) bool {
	base, err := url.Parse(pageURL)
	if err != nil {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	u = base.ResolveReference(u)
	if (u.Scheme != "http" && u.Scheme != "https") || strings.EqualFold(u.Hostname(), base.Hostname()) {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	for _, domain := range []string{"wikipedia.org", "iso.org", "w3.org", "ietf.org", "unicode.org", "who.int"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	parts := strings.Split(host, ".")
	for i, part := range parts {
		if i > 0 && (part == "gov" || part == "edu" || part == "ac") && (i == len(parts)-1 || i == len(parts)-2 && len(parts[len(parts)-1]) == 2) {
			return true
		}
	}
	return false
}

func (c *citationAccumulator) finish(signals *AIPageSignals, bodyText string) {
	c.closeParagraph()
	c.closeSection()
	signals.Version = CitationSignalsVersion
	signals.SnippetBlocked = c.snippetBlocked
	signals.SnippetTextWords, signals.NoSnippetWords = c.textWords, c.noSnippetWords
	signals.AnswerCapsule, signals.DefinitionAfterHeading = c.capsule, c.defined
	signals.Sections, signals.ChunkReadySections = c.sections, c.chunkReady
	signals.AuthorityLinks = c.authorityLinks
	signals.InjectionEvidence, signals.TopicHeadings, signals.TopicLinks = c.evidence, c.headings, c.topicLinks
	for _, claim := range inYear.FindAllString(bodyText, 20) {
		if year, err := strconv.Atoi(yearPattern.FindString(claim)); err == nil {
			signals.DecayYears = append(signals.DecayYears, year)
		}
	}
	for _, claim := range copyrightYears.FindAllString(bodyText, 20) {
		years := yearPattern.FindAllString(claim, -1)
		if len(years) > 0 {
			if year, err := strconv.Atoi(years[len(years)-1]); err == nil {
				signals.DecayYears = append(signals.DecayYears, year)
			}
		}
	}
	signals.DatedClaims = statisticPattern.MatchString(bodyText) || versionPattern.MatchString(bodyText)
}
