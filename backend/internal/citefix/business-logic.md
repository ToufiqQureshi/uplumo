# Cite Fix: rules that protect money and trust

- **Opening the page is free.** The report reads saved data and fetches rival
  pages over plain HTTP; it never calls DataForSEO or a model. Rival pages are
  cached 7 days per project, failed fetches are not retried for 24 hours.
- **Rival fetching is polite and bounded:** robots.txt is obeyed, private and
  internal addresses are refused (SSRF guard), at most 8 pages per report,
  12 s per page and 8 s for the whole step, so a slow site never delays the
  report. App-shell pages are reported, never rendered (rendering costs).
- **One workspace never sees another's cache.**
- **Paid actions show their cost first and charge at most that.** The agent
  holds credits before the model calls and charges the provider's real cost
  up to the hold; an identical rerun is free. The agent prompt keeps at most
  5 missed questions and 3 rival pages, so a richer report never makes a run
  cost more.
- **No guessing.** List pages are found from the page's own links to other
  AI-cited sites, not from word lists. Steps appear only when the data shows
  their problem. The UI says a signal gap is a likely reason, not proof: no
  tool can promise a citation.
- **One page, one row:** cited URLs lose `#:~:text=` highlights and `utm_`
  tags, so one page is never counted as several.
- **Global:** engine names and dates come from shared labels and the viewer's
  locale; nothing assumes one country.
- **Pitch drafts:** one click, one charge, at most 25 credits (plan catalog);
  replays by key and drafts of the same page within 30 days are free; one
  draft per page at a time; 20 per project per 10 minutes; a failed or
  unusable answer costs the customer nothing. Drafts are offered only for
  pages Cite Fix found, never for an arbitrary URL.
- **Fix proof is "likely linked, not proven"**: AI assistants do not say who
  clicked, so the UI never claims a fix caused the change.
- **Topics name a subject, not a shared word:** a one-word topic needs at
  least 3 pages, and plural spellings fold into a singular the same scan
  has ("small businesses" into "small business").
- **A list page lists firms, not footer links:** a host that most read pages
  (3 or more, and at least half) link to is site furniture such as social
  profiles, and does not make a page a list of rivals.
