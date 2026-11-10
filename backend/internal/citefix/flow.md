# Cite Fix: request flow

## Opening the page (free, 0 paid calls)

Screen → `POST .../cite-fix/report` → `Store.Report`:

1. Latest completed site scan: score, fixes, pages, topics (or `no_scan` /
   `needs_rescan`). Addresses of one page (same path and text, another query
   string) are merged first; article-only checks run on articles only.
2. `proof`: the latest finished AI Visibility run's saved answers → how many
   cite the customer, rivals, every missed question and engine (up to 60) with
   the sites cited instead (URLs canonical: no `#:~:text` or `utm_`), plus the
   history of comparable checks (`visibility`).
3. Saved mentions and Google AI Overview results, as stored.
4. `rivals`: up to 8 AI-cited pages, one per missed question before any gets
   a second. Each is read from the 7-day cache or fetched for free (robots.txt
   respected, SSRF guard, 12 s per page, 8 s for the whole step). The page's
   signals and outbound link hosts are cached; the report turns the hosts into
   `listsRivals` (other AI-cited sites the page links to) and `linksYou`.

The screen then builds, without another request: the headline, the question
× engine grid (cited / lost / never), rivals by average spot, list pages
(links to 2+ rivals, not to the customer), the signal comparison and the
ordered next steps (`citeFixInsights.ts`).

## Paid actions (one click, one charge)

- **Google AI Overview check:** confirm dialog with the credit cap → one
  DataForSEO batch for the saved questions → results saved 7 days; no
  automatic retry.
- **AI agent:** the customer picks pages and mode; credits are held up to the
  shown cap per page before the model calls; the same page and content
  answers from cache for 0 credits. The prompt carries at most 5 missed
  questions and 3 rival pages, whatever the report shows.
- **Outreach pitch:** "Draft a pitch" on a list page → confirm with the cap
  (`plans.CiteFixPitchMaxCredits`, sent as `agent.pitchMaxCredits`) →
  `POST .../cite-fix/pitch` → only a cached AI-cited page linking to 2+ other
  sites and not to the customer qualifies → one OpenRouter call under one
  hold → saved; the same page within 30 days is shown again for 0 credits.

## Proof after a fix (free)

`report.fixProofs`: for each page the agent fixed (last 180 days), AI
visitors to that path and the leads in their visits (owner-chosen lead
actions, from Analytics) for the same number of days before and after the
fix (at most 28), and whether the latest AI check cites the page.
`proof.said`: up to 6 excerpts of saved answers that name the brand or domain.
