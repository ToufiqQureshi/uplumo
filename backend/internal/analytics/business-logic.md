# Analytics: rules that protect trust

- **We never guess what a lead is.** The owner picks lead actions; until then
  no lead number, rate or insight is shown. (An earlier version guessed from
  event names and told a site without WhatsApp that nobody tapped WhatsApp.)
- **Broken tracking hides the numbers.** Page views without visits mean an old
  snippet; sources and rates would be wrong, so only the fix is shown.
- **Visitors are visitor-days.** The visitor hash rotates daily with a secret
  salt, so nobody can be followed across days. The UI says "once per day".
- **A visit belongs to the source of its first page**, so source shares add up
  and a visit crossing the range edge keeps its real source.
- **Leads tie to a visit only with a session.** Old snippets send none; their
  leads count in the total and show as "Source not known" (`untiedVisitors`),
  so the sources table adds up to the total.
- **Time counts only while someone is there.** The tracker counts a page's
  time while the tab is visible and up to 2 minutes after the last input
  (scroll, key, pointer, touch). A tab left open does not add hours, and it
  does not keep its session alive past the 30-minute idle limit.
- **Laptops are not tablets.** Screen width only makes a Mac user agent a
  tablet (iPadOS reports a Mac); Windows and Linux laptops at 125-150%
  scaling report tablet-sized widths.
- **No zero we cannot measure.** A click to a page with no pageview in the
  range (one the tracker does not run on) shows "Not tracked", not 0 arrivals.
- **One answer per question.** Sources are shown once (visitors and leads per
  source); AI visitors once. Cards with nothing to show are hidden.
- **Uplumo's own site counts only public pages.** `site/static/analytics.js`
  loads Uplumo's tracker only where the script tag has `data-public-site`
  (the public templates), so customers using the app are not website
  visitors (owner, 12 Nov 2026). GA4 still runs on both.
- **Insights state only facts the data shows**: no generic advice, no copy
  that assumes a button, country or business type.
- **No personal data:** no IP, phone number, email or form answer is stored.
- **Free:** Analytics makes no paid call; nothing here can cost credits.
- Lead actions: at most 20 names of 1 to 64 characters per project; saving
  replaces the set in one transaction so a reader never sees a mix.
