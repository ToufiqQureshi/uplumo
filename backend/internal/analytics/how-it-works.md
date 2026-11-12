# Analytics: how it works for a user

1. Open Traffic. With no visits yet, copy the tracking code and paste it in
   the site's `<head>`. Open the site once; visits show within a minute.
2. If the page says tracking is half working, the site runs an old code:
   replace it. Until then no numbers are shown, because they would be wrong.
3. Under "What counts as a lead?", tick the actions that mean business for
   you (a booking button, a form, a download). Missing one? Add
   `data-uplumo-event="Name"` to the button, or call `uplumo("Name")`.
4. The top of the page then shows leads, visitors and the share of visitors
   who became leads, each against the previous period with the old number
   ("+25% (was 40)"; no comparison when the previous period has no data),
   and one sentence on where to act: the best source (every source, when
   several tie), or a busy page that converts badly.
5. "Sources that bring leads" shows which channel's visits led to a lead;
   leads from an old tracking code show as "Source not known". "Busy pages
   and their leads" shows where people arrive and where leads happen.
   "Visitors from AI assistants" shows ChatGPT, Perplexity and other AI
   visits and where they landed; it is hidden when there were none.
6. Everything else (chart, pages, actions, campaigns, countries, devices) is
   under "All numbers". The daily new/returning table shows only once browser
   recognition is on. A click to a page the tracker does not run on shows
   "Not tracked" instead of zero arrivals.
