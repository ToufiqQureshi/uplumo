# Business logic

- **Pro and Business only** (owner, 9 Nov 2026): every route checks
  `plans.LocalVisibility`; the weekly tick skips workspaces without it.
- **Nothing paid without a click.** Page load and polling read the database.
  The watch is off until turned on, and turning it on shows its weekly cost.
- **Pay once per task.** One running run per project and kind (unique index);
  each task posted under `local:<run>:<task>` and stored at once, so a retried
  job or a second worker collects instead of buying again. No paid call is
  retried automatically; `task_get` is free.
- **24-hour cache** per project and input (business, search term, grid size,
  competitors). A cached answer says nothing was charged. A run that came
  back with parts missing is not reused, so a second click buys them.
- **Priority queue, not live**: about a minute's wait for roughly half the live
  price. Only the business search is live (the customer waits for the list).
- **Charged = what DataForSEO billed**, never above the quote on the button:
  the spend guard holds each task's published ceiling and settles the real
  cost. AI calls hold a fixed ceiling from capped input size and output
  tokens, and charge nothing when the answer is unusable.
- **No made-up numbers.** Fixes appear only when most competitors' profiles
  show the gap; nothing is a fixed benchmark. Theme counts are review ids the
  server verified, never numbers from the model. Reply drafts are written only
  for reviews in the stored result, so the endpoint is not a free AI writer.
- **Country-neutral.** Searches use the project's country and language; the
  Maps search term and the AI prompt are the customer's own words.
- **Saved coordinates are Google's**: a business can only be saved from the
  project's own latest search result.
- **Changing the business stops the watch**, which followed the old one's
  competitors.
