# Analytics: how it works for a user

1. Open Traffic. With no visits yet, copy the tracking code and paste it in
   the site's `<head>`. Open the site once; visits show within a minute.
2. If the page says tracking is half working, the site runs an old code:
   replace it. Until then no numbers are shown, because they would be wrong.
3. Under "What counts as a lead?", tick the actions that mean business for
   you (a booking button, a form, a download). Missing one? Add
   `data-uplumo-event="Name"` to the button, or call `uplumo("Name")`.
4. The top of the page then shows leads, visitors and the share of visitors
   who became leads, each against the previous period, and one sentence on
   where to act: the best source, or a busy page that converts badly.
5. "Sources that bring leads" shows which channel's visits led to a lead.
   "Busy pages and their leads" shows where people arrive and where leads
   happen. "Visitors from AI assistants" shows ChatGPT, Perplexity and other
   AI visits and where they landed; it links to Cite Fix.
6. Everything else (chart, pages, actions, audience, campaigns, countries,
   devices) is under "All numbers".
