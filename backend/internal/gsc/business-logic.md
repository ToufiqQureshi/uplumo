# Search Console: rules that protect trust

- **Read-only grant.** Only `webmasters.readonly` is requested. Google's
  "Request indexing" button has no API, and Uplumo does not imitate it.
- **No money on this path.** The analysis uses free reads only. A paid call
  added here must go through the spend guard and be asserted in a test.
- **No guessed brand.** Branded and non-branded exist only when the owner gave
  brand names (at most 20, 100 characters each, checked on the server).
- **No borrowed benchmarks.** "People skip this search" compares a search
  with the median CTR of the same site's other searches at the same rank,
  never with an industry average. A rank needs 8 other searches before it is
  trusted. Possible extra clicks are labelled an estimate.
- **Cut-offs are relative.** A search is judged only above the larger of 20
  impressions and 0.05% of the site's search impressions, so a small and a
  large site are each judged by their own size. All numbers live in
  `DefaultInsightRules`.
- **Real losses before estimates.** The top actions list measured lost clicks
  first, then estimates, then shared searches, at most two of one kind.
- **Branded searches are left out** of the competing-pages and CTR checks when
  brand names are known: a home page and an about page both showing for the
  brand is normal.
- **Say what was not read.** Lists report when the 10,000-row read was full,
  when the date range leaves no earlier period, and that Google hides rare
  searches.
- **Long-tail counts words.** A language written without spaces does not
  reach the long-tail count; the segment simply shows fewer searches.
- **Bad filters never reach Google.** A regex that does not compile is refused
  here, because a request Google rejects is wasted quota.
