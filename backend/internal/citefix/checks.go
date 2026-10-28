package citefix

// Category groups checks into the six parts of the score. Weights sum to 100,
// so a category's weight is the most points it can add to the site score.
type Category struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Weight int    `json:"weight"`
}

// categories are listed in the order the page shows them. Access and answers
// weigh most: an assistant that cannot read a page, or cannot find a direct
// answer in it, never quotes it, however trusted the site is.
var categories = []Category{
	{ID: "access", Label: "AI can read the site", Weight: 20},
	{ID: "identity", Label: "AI knows who you are", Weight: 15},
	{ID: "trust", Label: "Author and freshness", Weight: 15},
	{ID: "answers", Label: "Direct answers", Weight: 20},
	{ID: "evidence", Label: "Facts and sources", Weight: 15},
	{ID: "content", Label: "Content depth", Weight: 15},
}

// check is one test. A site check passes or fails once for the whole site; a
// page check is scored by the share of content pages that pass it.
type check struct {
	id       string
	category string
	// weight is the check's share of its category.
	weight int
	page   bool
	title  string
	why    string
	how    string
}

// checks hold the copy the customer reads. Plain words: most customers are
// business owners, not SEO specialists.
var checks = []check{
	{
		id: "ai-search-bots-allowed", category: "access", weight: 3,
		title: "AI search bots are blocked in robots.txt",
		why:   "ChatGPT search, Perplexity and Claude search read your pages with their own bots. A bot that robots.txt blocks cannot link or quote your site.",
		how:   "Remove the Disallow rules for OAI-SearchBot, ChatGPT-User, PerplexityBot and Claude-SearchBot in robots.txt, and check your firewall or CDN bot protection does not block them. Blocking only training bots (GPTBot, ClaudeBot, Google-Extended) is a choice and does not lower this score.",
	},
	{
		id: "pages-readable", category: "access", weight: 2, page: true,
		title: "Pages show no text without JavaScript",
		why:   "AI bots read the HTML your server sends and most do not run JavaScript. A page that builds its text in the browser looks empty to them.",
		how:   "Turn on server-side rendering or pre-rendering for these pages, so the HTML already holds the headings and text.",
	},
	{
		id: "llms-txt", category: "access", weight: 1,
		title: "No llms.txt file",
		why:   "llms.txt is a short list of your most important pages for AI tools. Few assistants read it yet, so it matters least here.",
		how:   "Add a plain-text /llms.txt with your main pages, one line each saying what the page answers. Do this after the other fixes.",
	},
	{
		id: "business-identity", category: "identity", weight: 2,
		title: "AI cannot read who the business is",
		why:   "Organization or LocalBusiness schema on the home page tells an assistant your name, address, phone and what you do, so it can name you with confidence.",
		how:   "Add Organization or LocalBusiness JSON-LD to the home page with your name, logo, address, phone, opening hours and social profiles.",
	},
	{
		id: "structured-data", category: "identity", weight: 2, page: true,
		title: "Pages have no structured data",
		why:   "Schema (Article, FAQPage, Product, Service) labels what a page is about, so an assistant can pick the right page for a question.",
		how:   "Add the schema type that matches each page: Article for posts, Product for products, Service for service pages, FAQPage where you answer questions.",
	},
	{
		id: "one-phone-number", category: "identity", weight: 1,
		title: "Different phone numbers across the site",
		why:   "Assistants trust details that match everywhere. Several numbers make it unclear which one to give a customer.",
		how:   "Use one main number on the website, Google Business Profile and listings, or label each number clearly (sales, support).",
	},
	{
		id: "author", category: "trust", weight: 2, page: true,
		title: "Pages do not say who wrote them",
		why:   "Assistants prefer content from a named person or business they can check. An unsigned page is harder to trust.",
		how:   "Show the author's name with a short bio and link to an author or About page. Add author to the page's Article schema.",
	},
	{
		id: "dated", category: "trust", weight: 1, page: true,
		title: "Pages show no publish or update date",
		why:   "Without a date an assistant cannot tell whether your prices, rules or facts are current, so it picks a page that has one.",
		how:   "Show the published and last-updated date on the page and add datePublished and dateModified to its schema.",
	},
	{
		id: "fresh", category: "trust", weight: 1, page: true,
		title: "Pages not updated in the last year",
		why:   "For questions that change (prices, rules, best options), assistants favour recently updated pages.",
		how:   "Review these pages, update facts that changed, and change the updated date only when you change the content.",
	},
	{
		id: "answer-first", category: "answers", weight: 2, page: true,
		title: "The answer is not in the first paragraph",
		why:   "Assistants quote short, direct answers. A page that opens with a long story or no paragraph at all gives them nothing to lift.",
		how:   "Start the page with a 1 to 3 sentence answer (10 to 80 words) to the question the page is about, then add the detail.",
	},
	{
		id: "question-headings", category: "answers", weight: 1, page: true,
		title: "No questions as headings",
		why:   "People ask assistants questions. A heading that matches the question, with the answer right below it, is easy to find and quote.",
		how:   "Add H2 or H3 headings phrased as your customers' real questions (from Search Console and the AI Visibility Tracker) and answer each in its first sentence.",
	},
	{
		id: "scannable", category: "answers", weight: 1, page: true,
		title: "No lists or tables",
		why:   "Steps, comparisons and prices in a list or table are easier for an assistant to read and reuse than the same facts in a long paragraph.",
		how:   "Put steps in a numbered list, options in a bullet list and comparisons or prices in a table.",
	},
	{
		id: "statistics", category: "evidence", weight: 2, page: true,
		title: "No numbers or statistics",
		why:   "Research on AI search (the KDD 2024 GEO study) found that adding statistics is one of the changes that most raised how often a page was cited.",
		how:   "Add specific figures you can stand behind: prices, percentages, distances, counts, results, with where each number comes from.",
	},
	{
		id: "sources", category: "evidence", weight: 2, page: true,
		title: "No links to sources",
		why:   "Citing sources was another change the KDD 2024 GEO study found raised citations. It shows your facts can be checked.",
		how:   "Link to the official or original source for the facts you state: government pages, studies, manufacturer specs.",
	},
	{
		id: "quotes", category: "evidence", weight: 2, page: true,
		title: "No quotes from experts or customers",
		why:   "Adding quotations was the third change the KDD 2024 GEO study found raises citations: a marked quote is a sentence an assistant can attribute.",
		how:   "Add a short quote in a blockquote with the person's name and role.",
	},
	{
		id: "not-thin", category: "content", weight: 2, page: true,
		title: "Pages with under 300 words",
		why:   "A page with little text rarely answers a question fully, so assistants quote a fuller page instead.",
		how:   "Expand these pages with the questions customers ask about the topic, or merge them into a stronger page.",
	},
	{
		id: "unique", category: "content", weight: 1, page: true,
		title: "Pages with the same text as another page",
		why:   "When pages repeat each other, the assistant may pick the wrong one, or neither.",
		how:   "Keep one page per topic: merge duplicates, add a canonical link from the copies, or rewrite each for its own question.",
	},
}

// Thresholds behind the page checks.
const (
	// minLeadWords and maxLeadWords bound an answer-first lead: shorter is a
	// label or a fragment, longer is a story, not a quotable answer.
	minLeadWords = 10
	maxLeadWords = 80
	// minWords is the thin-content line the site audit also uses.
	minWords = 300
	// freshDays is how recent an update must be to count as fresh.
	freshDays = 365
)

// searchBots are the robots.txt agents that fetch pages to answer a question
// and link the source. Blocking one stops that assistant from citing the site.
// Training-only agents are left out on purpose: blocking them is a choice.
var searchBots = []string{"OAI-SearchBot", "ChatGPT-User", "PerplexityBot", "Claude-SearchBot"}
