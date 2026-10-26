# AI Visibility and prompt research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Why the feature is structured this way

Brand Lookup compares brand mentions and cited pages across supported answer engines; Prompt Explorer runs a prompt against selected models.

## Brand Lookup answers "does AI recommend me, and what do I fix"

- **Decision:** Add a country picker (URL `loc`, default the project market,
  language from the country), never run the lookup before the country is
  known, show "Where AI recommends you" (unbranded prompts), "Sites AI reads
  about you" (cited sites minus the brand's own) and "Brands AI names with you",
  each with a plain next step, and a no-mentions panel with three first steps.
  The competitor cost line reads 130 credits, 260 in the US.
- **Why:** The page sent `locationCode: 2840` for every project, so an Indian
  business saw US answers. The backend now returns the cited sites, co-mentioned
  brands and a branded flag it used to drop (`backend/internal/aisearch/decision.md`).
- **Preserve:** No paid retry/refetch; results cached before these fields
  parse and render without the new sections; the brand's own domain is never
  listed as a site to fix; the lookup is not enabled while the country is unknown.
- **Trace:** `BrandLookupPage.tsx`, `components/BrandLookupSearchCard.tsx`,
  `components/BrandLookupInsights.tsx` (+ test), `components/BrandLookupResults.tsx`,
  `paid-refetch.test.ts`, `types/schemas/ai-search.ts`,
  `client/hooks/useBrandLookupSearchHistory.ts`, the `brand-lookup` route.
  Not checked in a browser against live data: production's DataForSEO key was
  rejected and the provider account's daily limit was reached on 22 Oct.

## Decisions to preserve

- **These lookups can incur paid provider cost: keep the server plan/cost gate and input validation before provider calls** _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- **Do not add automatic client retries for a paid POST** — an explicit user retry is a new lookup. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- **Citations are untrusted URLs** — only render clickable links for absolute
  HTTP/HTTPS URLs; display other provider values as text. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- **Provider coverage and mention counts are estimates** — the UI must not promise guaranteed ranking or AI inclusion. _Why:_ These provider samples are directional measurements, not a guarantee that a search engine will cite or rank Uplumo customers.
- Read history only after a successful paid Brand Lookup and use stored monthly aggregates; never add a provider call to load the chart. _Why:_ Historical display must not multiply the paid cost of the lookup that collected the data.
- Keep history scoped by organization/project and every result-changing lookup option; show Analytics referrals as a separate measure from citations. _Why:_ Mixed scopes or metrics would produce misleading trends and could expose another project's data.
- Treat the drop notice as an estimate and preserve the existing server-side plan gate. _Why:_ Provider estimates can change and a client-side paywall is not authorization.

## Show the sources and brands each query cited

- **Decision:** `QueryAnswerDetails` now renders `citedSources` (the pages the
  AI cited) and `brandsMentioned` (brands named alongside) in the same
  expandable detail that already shows the answer, retrieved pages and fan-out
  queries.
- **Why:** The backend returns these per query with the already-paid mention;
  the UI hid them, so the customer lost the "who does the AI name instead of me"
  evidence the lookup paid for.
- **Preserve:** No paid call; old cached results (no citedSources/brandsMentioned)
  still render; citation links use the same untrusted-URL handling as retrieved
  pages; both use the same HTTP/HTTPS allowlist before creating a link.
- **Trace:** `components/QueryAnswerDetails.tsx`, `types/schemas/ai-search.ts`.

## Verification pointers

- `aiSearchApi.test.ts`
- `paid-refetch.test.ts`

## No focus or reconnect purchases

- **Decision:** Disable `refetchOnWindowFocus` and `refetchOnReconnect` for the
  Brand Lookup and Prompt Explorer paid query hooks alongside `retry: false`.
- **Why:** A failed paid query ran three API requests after one lookup followed
  by a focus and reconnect event. A provider may already have billed a failure;
  a cache miss or partial answer can therefore turn these refetches into spend.
- **Preserve:** Keep explicit Retry, free history/active-run polling and the
  server's plan gate. Do not change pricing or the provider's model selection.
- **Trace:** `BrandLookupPage.tsx`, `PromptExplorerPage.tsx` and mounted React
  Query regressions in `paid-refetch.test.ts` (failure/focus/reconnect).

## Render provider source titles as nullable

- **Decision:** Let `SourceLink` accept the API's nullable title and use its existing URL-label fallback.
- **Why:** Both cited/retrieved source call sites failed TypeScript checks because the provider schema permits missing titles.
- **Preserve:** Only credential-free HTTP(S) links are clickable; no new fetch or provider work.
- **Trace:** `components/QueryAnswerDetails.tsx`, TypeScript checks; the previously excluded TSX history regression is now included by Vite test discovery.

## Suggested local questions for the tracker

- **Decision:** The tracker form has "Suggest questions". The customer types what they sell and their city or area. `starterQuestions.ts` then adds up to 8 questions:
  - 4 that find a business (best, top rated, affordable, recommended in that city)
  - 2 about the brand
  - up to 2 brand-versus-competitor questions

  `addQuestions` fills the blank row first, skips duplicates (ignoring case) and keeps the 30-question cap.

- **Why:** The tracker started with one empty box. Ubersuggest picks broad industry prompts, such as "best luxury hotels in India" for a mid-range Mumbai hotel, and reports a useless 0% ([review](../../../../../docs/audits/2026-10-26-ubersuggest-competitor.md)). Questions that name the business type and city measure the market the customer actually competes in.
- **Preserve:** No AI or provider call. Suggestions only edit the unsaved form, so the existing "Review cost" step still quotes credits before anything is saved or run.
- **Trace:** `starterQuestions.ts`, `starterQuestions.test.ts`. Each of these 6 mutations fails the test: city phrase, duplicate skip, blank-row replace, 30 cap, competitor cap, empty business. Also `TrackingForm.tsx`.

## "Why AI is not citing you" from saved answers

- **Decision:** `whyNotCited.ts` turns a check's saved answers into at most five ranked reasons, each with evidence and one action: listing/review sites AI linked instead (open the site), competitor pages that answer the question (write the page in the content editor, keyword = the prompt), named but not linked (write the answer page), other businesses AI recommended (track them), community sites (open the site). `WhyNotCitedCard` shows them on the tracker page; the dashboard AI card shows the main reason and links to it.
- **Why:** The owner's core promise is "why not you, what to fix". Ubersuggest's equivalent is generic advice with an empty Opportunities list ([review](../../../../../docs/audits/2026-10-26-ubersuggest-competitor.md)); the real reasons (e.g. AI cites Booking, Tripadvisor and Justdial for a Mumbai hotel) are already in answers we paid for.
- **Preserve:** No provider call and no credits: it reads only `GET .../tracking/runs/{runId}`. Only completed, non-empty answers count. Any citation of the project's domain or a subdomain counts as cited. The editor keyword stays within the route's 200-character limit and only pre-fills the draft (no paid lookup on load). The listing and community site lists are judgement, not provider data; extend them when a market needs it.
- **Trace:** `whyNotCited.ts` and its test (5 mutations: cited skip, status filter, subdomain match, listing classification, keyword cut, each fails it), `components/WhyNotCitedCard.tsx`, `PromptTrackingPage.tsx`, `../dashboard/InsightCards.tsx`. Not run: a check against production answers.
