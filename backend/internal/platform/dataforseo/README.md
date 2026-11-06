# Uplumo DataForSEO integration: required reading

Before DataForSEO work, read this practical guide and [the Uplumo cost/value rules](business-logic.md). Use [the full reference snapshot](reference.md) for the relevant endpoint's complete request/response contract. See [the integration flow](flow.md) for the reviewed code path.

The guide below is dated provider research. **Uplumo rules take precedence over generic integration advice**: no automatic paid retries; no paid work on page load; shared spend guard and usage recording; owner authorization for runtime money/scope changes; no silent loss of promised customer value. Generic examples are not new Uplumo features or production prices. This PR changes no runtime behavior and has 0 paid calls / 0 customer credits.

The full reference is about 28 MB. Search by endpoint, then read its complete relevant section; do not send the entire file to an LLM on every request. Full collection/provenance details are embedded there. The separate original research bundle mentioned below is a local artifact, not a promised product download.

---

# DataForSEO Developer Guide aur Paisa Bachane ka Playbook

Research date: **6 November 2026 (UTC)**. Language: Hinglish; exact API field names aur official specifications English mein preserve ki gayi hain.

## 1. Is file ko kaise use karna hai

Yeh research public official DataForSEO v3 documentation, uske LLM-friendly index, current pricing pages, Help Center aur official OpenAPI repository par based hai. Koi paid API call nahi ki gayi; account credentials access nahi kiye gaye. Neeche ke calculations published USD prices par examples hain, account-specific quotation ya billing guarantee nahi.

**Pehle practical guide padho; implementation ke waqt exact endpoint ki reference padho.** `reference.md` mein yahi guide, endpoint index, documented cost warnings, pricing snapshots, full downloaded endpoint documentation aur OpenAPI specification ek hi Markdown file mein hain. `README.md` uska aasani se padhne wala practical hissa hai. Reference copies official source material hain; recommendations aur calculations is guide ki analysis hain.

Scope: official index aur navigation se discover hue pages ka dated snapshot. “Saari docs padh li aur kabhi kuch change nahi hoga” ka claim sahi nahi hoga: docs, prices, models aur supported locations badalte hain. Corpus ki coverage, failures, source URLs aur checksums manifest mein hain. Crawl/download aur structured extraction ko manual line-by-line review ke barabar na samjho. Main workflows, pricing, cost-sensitive parameters aur operational pitfalls ko detail mein review kiya gaya hai; full field-level source reference preserve hai.

Useful navigation in master file: `PART A` practical guide; `PART B` coverage/index; `PART C` cost-sensitive input fields; `PART D` official pricing/Help Center snapshots; `PART E` official endpoint documents; `PART F` OpenAPI source.

Official entry points:

- Docs: https://docs.dataforseo.com/v3/
- All indexed Markdown pages: https://docs.dataforseo.com/v3/llms.txt/
- Pricing: https://dataforseo.com/pricing/
- Machine-readable specification: https://github.com/dataforseo/OpenApiDocumentation
- API credentials: https://app.dataforseo.com/api-access
- API limits and cost settings: https://app.dataforseo.com/api-settings

## 2. Sabse pehle samjho: DataForSEO mein alag products ka data alag hai

| Product | Kya milta hai | Typical use | Cost/quality decision |
|---|---|---|---|
| SERP API | Search engine pages se collected results; organic aur special verticals | Rank tracking, search results, SERP features | Fresh collection; mode, depth aur add-ons se bill badalta hai |
| AI Optimization | LLM responses, scraped AI interfaces, mentions database, AI keyword demand | Brand visibility in AI, citations, prompt monitoring | In chaaron ko interchangeable mat samjho; provider tokens bhi charge ho sakte hain |
| Keywords Data | Google Ads/Bing Ads metrics, Google Trends, DataForSEO Trends, clickstream | Search volume, CPC, ideas, ad forecasts, trends | Google Ads batch ko fill karna bahut valuable; source/aggregation alag hota hai |
| DataForSEO Labs | Existing keyword/SERP/app/Amazon databases par analytics | Keywords, competitors, intersections, historical visibility | `live` response ka matlab freshly scraped SERP nahi; task + result-item billing |
| Backlinks | Continuously crawled proprietary link index | Link audits, domains, anchors, opportunities | Summary/bulk first; detailed rows sirf zaroorat par |
| OnPage | Website crawler, page checks, parsing, screenshots, Lighthouse | Technical SEO audit | Crawl scope aur rendering flags sabse bade cost controls |
| Domain Analytics | Technologies aur WHOIS information | Lead enrichment, tech segmentation, domain research | Domain lookup aur large discovery query ka billing model alag |
| Content Analysis | Web content mentions, sentiment, trends aur aggregation | Brand/content research | Search ke bajay aggregate endpoint se kaam ho sakta hai |
| Merchant | Google Shopping aur Amazon products/sellers/product details/reviews | Ecommerce intelligence | Marketplace, identifiers aur result blocks matter |
| App Data | Google Play/App Store searches, lists, details, reviews, listings | ASO aur app intelligence | Scraped app details aur database listings alag services |
| Business Data | Business listings, Google profiles/reviews/hotels/Q&A, Trustpilot, Tripadvisor, social data | Local SEO, reputation, lead discovery | IDs reuse karo; review counts aur identifier surcharge check karo |
| Databases | Bulk database exports/products | Bahut bade offline research jobs | API vs database delivery/price compare karo; current quote chahiye |
| Appendix | User data, errors, ID recovery, sandbox, webhooks, status, AI responses | Production operations | Recovery aur duplicate charges rokne ke tools |

### SERP breadth jo reference mein cover hai

Google Organic, AI Mode, Maps, Local Finder, News, Images, Search by Image, Jobs, Autocomplete, Datasets, Advertisers/Ads Search, Finance aur indexed Events material; Bing, Yahoo, Baidu, Naver, Seznam; YouTube search, info, subtitles aur comments. Har search engine mein same methods aur parameters available nahi hote.

### Labs breadth

Google: keywords for site, related keywords, suggestions, ideas, difficulty, intent, overview, historical keyword data, categories/market analysis, top searches, SERP competitors, ranked keywords, competitor domains, intersections, subdomains, relevant pages, rank overview, historical SERPs/rank/traffic aur bulk traffic.

Amazon: bulk volume, related/ranked keywords, product rank, competitors aur keyword intersections. Google Play/App Store: bulk app metrics, keywords for app, competitors aur intersections. Labs organic keyword competition aur Google Ads paid competition ko mix mat karo.

### AI Optimization breadth

LLM Mentions: search mentions, target/multi-target metrics, top pages/domains/brands/categories, historical/delta/new-lost time series aur Lite variants. AI Keyword Data: AI-platform keyword search volume. LLM Responses: supported ChatGPT/Claude/Gemini/Perplexity provider/model endpoints. LLM Scraper: supported ChatGPT/Gemini UI collections. Availability endpoint reference se verify karo; sab provider/method combinations available nahi.

## 3. Authentication, request aur response contract

Production base: `https://api.dataforseo.com/v3/`. Free dummy-data environment: `https://sandbox.dataforseo.com/v3/`.

HTTP Basic Auth use hota hai: `Authorization: Basic BASE64(api_login:api_password)`. API password dashboard login password se alag hai. URL mein credentials mat bhejo; server-side secret configuration use karo. API ko use karne se pehle alag token/login request ki zaroorat nahi.

POST body aam taur par JSON task-array hoti hai, single task ke liye bhi `[{...}]`. UTF-8 use karo. Integer ko integer, boolean ko boolean bhejo; `"false"` boolean `false` nahi hai. GET aur POST methods endpoint ke hisaab se follow karo; OnPage ke existing crawl data retrieval mein bhi POST endpoints hain.

Standard envelope mein usually `version`, `status_code`, `status_message`, `time`, `cost`, `tasks_count`, `tasks_error`, `tasks` aate hain. Har task mein `id`, task-level status, `cost`, `result_count`, `path`, `data`, `result` ho sakte hain. Har result ka shape product-specific hai.

**HTTP 200 ko successful data mat maano.** Envelope aur har task ka `status_code` inspect karo. Batch mein kuch tasks succeed aur kuch fail ho sakte hain. `tasks_error` aur per-task output preserve karo. `result: null` queued task create hone par normal ho sakta hai. Null search volume ko zero mein convert karne se false conclusions aa sakte hain.

`cost` USD mein hai. Top-level total aur task-level costs ko jod kar double-count mat karo: ek total hai, doosra uska breakdown. Historical docs ke sample response `cost` ko current price mat maano.

Compression: response support aur SDK behavior check karo; callbacks gzip-compressed ho sakte hain. XML suffix `.xml` aur supported HTML result endpoints par `.html` available hain. `.ai` ka behavior section 14 mein hai.

Source: https://docs.dataforseo.com/v3/auth/ and https://docs.dataforseo.com/v3/

## 4. Standard, Priority aur Live: kis kaam ke liye kya

| Method | Flow | Kab use karo | Important caveat |
|---|---|---|---|
| Standard | Task POST → task ID → wait/callback → Task GET | Scheduled/batch reports, rank tracking, offline jobs | Usually economical; task submitted hote hi charge lag sakta hai |
| High priority | Standard flow with supported `priority: 2` | Faster queued results required | Extra charge; har product mein same premium nahi |
| Live | Single POST returns result | User ko genuinely immediate result chahiye | Usually premium where both modes exist; many database APIs are Live-only |

Google Organic published snapshot: Standard **$0.0006/SERP**, Priority **$0.0012/SERP**, Live **$0.002/SERP**; one billing SERP = up to 10 organic results. Standard average ~5 minutes with target 45 minutes; Priority ~1 minute; Live ~6 seconds average. Yeh hard application SLA nahi samjho.

**Same 10,000 one-page Google Organic tasks:** Standard $6; Priority $12; Live $20. Standard vs Live base saving = **70%**. Yeh depth/add-ons ke bina example hai; doosre APIs par 70% assume mat karo.

A successful Standard creation `20100` ka `id` turant persist karo. `tag` project/customer/job correlation ke liye useful hai; **tag ko provider-side idempotency key mat samjho**.

Batch POST overhead kam karta hai, lekin 100 SERP tasks ek POST mein bhejne se woh ek billable SERP nahi ban jaate. Ek task mein multiple keywords/targets support karne wala actual bulk endpoint alag savings deta hai.

Sources: https://dataforseo.com/pricing/google-serp/google-organic-serp-api and https://dataforseo.com/help-center/live-vs-standard-method

## 5. Rate limits, task limits aur concurrency

Current docs kai endpoints ke liye **10,000 calls/minute** batati hain. Isko universal limit mat banao. Har endpoint ki reference aur response headers `X-RateLimit-Limit`, `X-RateLimit-Remaining` check karo. Purane tutorials mein 2,000/minute mil sakta hai; current endpoint docs ko prefer karo.

Common patterns:

- Supported Standard Task POST: generally up to **100 tasks/request**; >100 par extra tasks `40006` de sakte hain.
- Live: generally **one task/request**, lekin us ek task ke `keywords`/`targets` array mein endpoint limit tak bulk inputs ho sakte hain.
- Google Ads Keywords Data Live: **12 requests/minute/account** according to published pricing/help.
- Google Trends Live: Help Center **250 requests/minute across all clients** batata hai; isko apne account ka reserved throughput mat samjho.
- Labs, Backlinks, Content Analysis, Domain Analytics, Clickstream, DataForSEO Trends aur selected OnPage Live endpoints: **30 simultaneous queries** limit documented.
- LLM Responses Live: **30 concurrent tasks/account per platform** in reviewed ChatGPT Live docs.
- ID List: **10 calls/minute**, up to 1,000 IDs/call.

Token bucket/rate limiter + concurrency semaphore dono chahiye. Steady traffic rakho; thousands of requests ke sudden bursts avoid karo. SERP Live ke liye official recommendation **120-second timeout** hai. LLM responses jaise endpoints ka alag execution profile hai.

Sources: https://dataforseo.com/help-center/best-practices-live-endpoints-in-dataforseo-api and https://docs.dataforseo.com/v3/appendix/id_list/

## 6. SERP parameters: practical implementation map

Yeh Google Organic ka starting map hai; exact method ke complete tables master appendix mein hain. Same name ka meaning doosre endpoints mein badal sakta hai.

| Field/group | Meaning aur constraint | Cost implication |
|---|---|---|
| `keyword` | Organic query; up to 700 characters in reviewed task endpoint; `%`/`+` decoding documented | Certain search operators trigger 5× charge |
| `location_code` / `location_name` / `location_coordinate` | Supported alternatives; one appropriate location selection do | Har additional geo = separate task if collecting separately |
| `language_code` / `language_name` | Supported language selection | Har language variant alag cached dataset/task |
| `device` / `os` | Desktop/mobile; valid OS pair choose karo | Har combination ke liye alag task chahiye |
| `depth` | Number of desired results; default 10 for Google Organic | Page/block rounding se charge increases |
| `max_crawl_pages` | Max SERP pages crawler can visit; reviewed maximum 100 | Depth ke saath complement karta hai; alone simple result limit nahi |
| `stop_crawl_on_match` | Up to 10 target objects; `match_type` + `match_value` | Target milte hi crawl stop karke extra pages bach sakte hain |
| `target_search_mode` | `any` (default) vs `all`; stop targets ke saath | `all` se more pages crawl ho sakte hain |
| `find_targets_in` / `ignore_targets_in` | Relevant SERP item types select/exclude | Local pack match ko organic match samajh kar jaldi stop hone se bachao |
| `priority` | Standard normal 1 vs high 2 | High priority premium |
| `load_async_ai_overview` | Async AI Overview load karna | Extra charge; absent/not-async case refunds documented |
| `people_also_ask_click_depth` | 1–4 click depth | $0.00015 per performed click; depth ko click count mat samjho |
| `calculate_rectangles` | Pixel rankings; advanced output | Paid add-on |
| `browser_screen_width/height/resolution_ratio` | Pixel geometry viewport | Rectangles dependency; unsupported combinations reject ho sakte hain |
| `expand_ai_overview` | Expand AI overview in supported HTML results | JSON result ke liye same behavior assume mat karo |
| `group_organic_results` | Related result grouping | Parsing/rank interpretation changes |
| `search_param` / `url` / `se_domain` | Engine-specific query settings | Unsupported URL parameters auto-remove ho sakte hain |
| `remove_from_url` | Up to 10 URL parameters remove | Store normalized URL without losing necessary attribution |
| `tag`, `pingback_url`, `postback_url`, `postback_data` | Tracking/delivery | Duplicate resubmission avoid karne mein help |

Google Organic `task_post` max depth **700**, reviewed Live Regular/Advanced max depth **200**. Ek method ka max doosre mein copy mat karo. Organic output mein `rank_group` item-type ke andar rank hai, `rank_absolute` mixed SERP elements ke beech position hai; dono ko same organic rank samajhkar report mat karo. `item_types`, `items_count`, `pages_count`, actual source `datetime` aur corrected/spelled query inspect karo.

Google Organic coordinate format `latitude,longitude,radius` hai; Google Ads coordinate field ka geographic meaning alag hai.

### Search operator cost trap

`site:`, `intitle:`, `inurl:`, `intext:`, `filetype:`, `allintitle:`, `allinurl:`, `allintext:`, `allinanchor:`, `inanchor:`, `link:`, `define:`, `id:`, `info:` jaise operators relevant docs/pricing par 5× surcharge trigger karte hain. Exact list endpoint-specific hai; `cache:` support bhi pages mein inconsistent hai.

Agar goal apne domain ke ranking keywords dhoondhna hai, har keyword par `site:domain` Google scrape ke bajay Labs Ranked Keywords suitable ho sakta hai. **Data semantics change hoti hai**: normal query ki ranking aur `site:` query ranking same question nahi hain. Sirf surcharge bachane ke liye required query ka meaning mat badlo.

### Rank tracking ke liye stop-early

Supported Google Organic task payload example; yeh **example only hai, submit nahi kiya gaya**:

```json
[
  {
    "keyword": "example keyword",
    "location_code": 2840,
    "language_code": "en",
    "device": "desktop",
    "os": "windows",
    "priority": 1,
    "depth": 100,
    "max_crawl_pages": 10,
    "stop_crawl_on_match": [
      {"match_type": "with_subdomains", "match_value": "example.com"}
    ],
    "target_search_mode": "any",
    "find_targets_in": ["organic"],
    "tag": "project-example:job-001"
  }
]
```

Iska purpose top-100 ceiling ke andar organic target milne par stop karna hai. Not-found ka matlab selected scope mein nahi mila; search engine ne har requested result return kiya, yeh guarantee nahi. Full top-100 competitor dataset chahiye toh stop early use mat karo. Geographies aur locale codes official location list se lo; yahan 2840 = US example hai, India nahi.

Source: https://docs.dataforseo.com/v3/serp/google/organic/task_post/

## 7. SERP billing rules aur source conflicts

Without paid add-ons, Google Organic ka planning model:

`cost ≈ billable SERP pages × mode-specific base price`

`ceil(depth / 10)` useful upper planning approximation hai jab results available hon aur extra page crawler settings na hon. Actual collection, returned pages, refunds aur options se amount change ho sakta hai. `depth: 11` ko 11/10 linear bill mat samjho. First-page-only collection chahiye toh supported `max_crawl_pages: 1` useful hai: result count guess karne ke bajay page collection explicitly cap hoti hai, lekin requested top-N completeness reduce ho sakti hai.

**Important documented disagreement:** current Organic pricing page `calculate_rectangles` aur async AI Overview ko **+1 chosen-mode base price** batati hai; reviewed Standard `task_post` parameter tables **+$0.0006** fixed extra batati hain, jabki Live Advanced tables **+$0.002** batati hain. Normal Standard aur Live base se yeh match karta hai; Priority base $0.0012 ke saath fixed task-post number vs pricing multiplier ka mismatch bachta hai. Pricing page ke operator table mein “x5 per operator used” likha hai, lekin usi page ki FAQ aur endpoint docs keyword mein operator hone par x5 batati hain. Multiple operators par `5^n` formula invent mat karo.

Practical resolution: base pricing aur paid-feature estimate alag rakho; conflicting options ke liye higher plausible cost reserve karo. Production account se kabhi kaam shuru ho toh tiny required sample ka actual returned `cost`, refunds aur account ledger reconcile karo, ya official support se rule confirm karo. Is research mein paid verification nahi hua. Source contradictions ko hide karke exact combined formula dena reliable nahi hota.

Regular/Advanced/HTML Google Organic pricing mein included hain; **Advanced ko automatically premium mat samjho**. Extra dataset chahiye toh existing task ID ka supported advanced result retrieve karo, naya scrape post mat karo.

Other hidden charges:

- Baidu Organic `get_website_url: true`: reviewed endpoint docs **10×** task price batati hain.
- Google Finance Explore `news_type: local_market/world_markets`: reviewed docs **2×**.
- Google Finance Quote non-default `window` (anything other than `1D`): reviewed docs **2×**.
- Supported other engines/AI Mode mein rectangles ka rule different/multiplicative ho sakta hai.
- SERP Screenshot aur AI Summary independent endpoint costs ho sakte hain; basic SERP price mein free assume mat karo.

Sources: Google Organic pricing and corresponding endpoint tables in master reference.

## 8. Keywords Data: sabse practical batching savings

### Google Ads Search Volume

Reviewed limits: **1,000 keywords/task**, **80 characters/keyword**, **10 words/phrase**. Lowercasing documented. Invalid symbols, restricted categories aur Google grouping se missing/combined data aa sakta hai.

Parameters include `keywords`, supported location alternatives, `search_partners`, `date_from`, `date_to`, `include_adult_keywords`, `sort_by`, supported language fields, callbacks aur `tag`. Country/geographic meaning endpoint se verify karo; Google Ads coordinates country data de sakte hain, city-level circle nahi.

Published price: **$0.06/Standard task** or **$0.09/Live task**, not per individual keyword. Search Volume mein 1 keyword aur 1,000 keywords ka same per-task charge ho sakta hai.

For 10,000 keywords:

| Plan | Tasks | Standard cost | Live cost |
|---|---:|---:|---:|
| One keyword per task | 10,000 | $600 | $900 |
| 1,000 keywords per compatible task | 10 | $0.60 | $0.90 |

Same input coverage, valid batching aur no retries ke assumptions par **99.9% task-fee reduction**. Sab keywords ko bas concatenate nahi karna: same location, date range, network aur other result-defining settings wale inputs group karo.

**Similar-keyword trap:** Google Ads variants/synonyms ko combine kar sakta hai. Official advice synonymous inputs ko separate tasks mein distribute karne ka hai. Exact-word deduplication safe starting point hai; “30% similar text” heuristic official article mein aati hai, lekin lexical similarity semantic equivalence prove nahi karti. Arbitrary synonym deletion se valid keywords lose mat karo. Same cohort mein unrelated keyword variants fill karke batch utilization improve karo.

**Refresh timing:** Google Ads Status `actual_data` check karo; monthly update se pehle daily re-query usually nayi monthly data nahi deta. Dates aur data availability current Status se align karo.

**Language conflict:** endpoint tables language fields list karti hain; current official best-practices article kehta hai Google Ads keyword collection mein results language-independent hain aur language set na karo. Is behavior ko Google Ads-specific treat karo; SERP/Labs language fields globally remove mat karo.

Keywords For Site, Keywords For Keywords, Ad Traffic By Keywords ke inputs/output meanings separate hain. `bid` ad forecast modeling input hai; isko API request ka spending budget mat samjho.

Sources: https://dataforseo.com/pricing/keywords-data/google-ads and https://dataforseo.com/help-center/best-practices-for-handling-keywords-data-api-requests

### Bing, Trends aur Clickstream

Bing Ads Search Volume, History, Keywords for Site/Keywords, performance, URL suggestions aur audience estimation reference mein hain. Bing pricing per task hai, lekin Google ka 1,000-keyword cap har Bing endpoint par copy mat karo.

Google Trends index/relative interest absolute monthly search volume nahi hai. Keyword count, location, date range, category, item types, normalization aur supported comparisons endpoint-specific hain. DataForSEO Trends aur Clickstream alag datasets hain, drop-in replacements nahi.

Google Trends published snapshot: Standard **$0.0027/task**, Live **$0.011/task**. Five valid keywords per task use hon toh pricing page ka $540/$2,200 per 1M keywords example samajh aata hai; per-keyword amount ko universal flat rate mat samjho.

DataForSEO Trends Explore/Subregion/Demography/Merged alag prices hain. Sirf interest-over-time chahiye toh expensive merged response na mangao. Clickstream normalization/estimated demand ko Ads exact metric label karke display mat karo.

## 9. Labs: task fee aur per-item bill dono optimize karo

Most Google Labs endpoints published base: **$0.012/task + $0.00012/item**. Historical Rank Overview aur some historical/category bulk endpoints: **$0.12 + $0.0012/item**. Historical SERPs: **$0.00012/SERP snapshot**. Endpoint exceptions full pricing snapshot mein hain.

Keyword Overview **700 keywords** tak accept karta hai; Search Intent reviewed/pricing example **1,000** tak. “Labs mein sab 1,000” galat hai. `include_clickstream_data: true` supported Google endpoints mein **request price double** karta hai. Default false rakho jab use nahi chahiye. `include_serp_info` ko paid clickstream ke saath confuse mat karo.

Parameters: seed keyword / `keywords`, `target`, location + language, categories, `filters`, `order_by`, `limit`, `offset`, seed inclusion, historical dates, SERP info, intersection targeting, clickstream aur tags. Nested filter paths endpoint ke response schema se match karne chahiye.

**Important `depth` difference:** Related Keywords ka `depth: 0..4` related-keyword graph traversal hai; Google SERP result count nahi. Reviewed maxima approximately 0→seed, 1→8, 2→72, 3→584, 4→4,680 candidate keywords hain; returned results `limit`/pagination se controlled hain.

### Result utilization examples

Agar actually 1,000 rows chahiye:

- 10 calls × 100 rows: `10 × .012 + 1000 × .00012 = $0.24`.
- 1 call × 1,000 rows: `.012 + 1000 × .00012 = $0.132`.

Lekin sirf 100 rows chahiye toh 1,000 mangana saving nahi: 100-row call $0.024; 1,000-row call $0.132. **Batch utilization aur unnecessary overfetch alag cheezein hain.**

Suggestions: seed/topic set pehle chhota rakho, filters server par lagao, useful rows sort karo, phir expand. Ranked Keywords se already volume/CPC/difficulty/intent mil raha ho toh har field ke liye duplicate enrichment call mat karo. Coverage/timestamp required ho toh fresh SERP alag step hai. Missing database keyword ko zero demand mat label karo; Keyword Overview docs ke mutabik omitted keywords ke liye per-keyword charge nahi hota, par task charge apne rule se lag sakta hai.

Historical SERP use karke current live ranking ka claim mat karo. ETV estimated organic traffic hai, actual analytics sessions nahi. Competition index paid Ads competition hai; keyword difficulty organic ranking metric hai.

Sources: https://dataforseo.com/pricing/dataforseo-labs/dataforseo-google-api and exact Labs endpoint documents.

## 10. Backlinks: aggregate pehle, detail baad mein

Published snapshot: **$0.024/task + $0.000036/row**. Common detailed endpoint limit 1,000 rows; bulk endpoints up to 1,000 targets. Target domain/subdomain/page syntax aur `include_subdomains` semantics verify karo.

Main inputs: `target` or `targets`, `mode`, `filters`, `order_by`, `limit`, `offset`, `search_after_token`, `backlinks_status_type`, `include_subdomains`, `exclude_internal_backlinks`, `rank_scale`, dates where supported, `tag`.

Savings workflow:

1. Many domains ke liye Bulk Ranks / Bulk Referring Domains / other required bulk metric se shortlist.
2. Final candidates ke Summary dekho.
3. Detailed referring domains/backlinks/anchors sirf actual audit/opportunity extraction ke liye.
4. `mode: one_per_domain` useful ho sakta hai agar goal unique link sources hai; all backlinks audit ke liye isse data omit ho jayega.
5. `filters` aur sorting ka separate fee documented nahi; per returned-row savings ho sakti hai, task fee phir bhi hai.
6. Backlinks detailed endpoint ka `offset` maximum **20,000**; uske baad supported `search_after_token` use karo. Token ko manually invent nahi karna.
7. `backlinks_status_type` default reviewed endpoint mein `live` hai; lost-link report ke liye explicit matching scope chahiye.

If you actually need 1,000 rows: one 1,000-row call `.024 + .036 = $0.06`; ten 100-row calls `.24 + .036 = $0.276`. Same complete dataset ke liye ~78.3% saving. Sirf 100 rows chahiye toh unnecessary 900 aur rows mat kharido.

Rank default scale **0–1000** ho sakti hai; supported `rank_scale: one_hundred` se 0–100 choose kar sakte ho. Vendor link rank ko Google PageRank ya third-party Domain Authority ka exact equivalent mat present karo.

Source: https://dataforseo.com/pricing/backlinks/backlinks and https://docs.dataforseo.com/v3/backlinks/backlinks/live/

## 11. OnPage: rendering ko har page par enable mat karo

Published snapshot:

| Configuration | Per-page total | 10,000 pages |
|---|---:|---:|
| Base crawl | $0.00015 | $1.50 |
| Load resources only | $0.00045 | $4.50 |
| JavaScript option alone as priced | $0.0015 | $15.00 |
| Browser rendering bundle | $0.0051 | $51.00 |
| Base + content parsing | $0.0003 | $3.00 |
| Base + keyword density | $0.0003 | $3.00 |

Rendering bundle mein JS/resources included hain according to pricing page; unke fees dobara add mat karo. Request mein rendering ke saath `enable_javascript: true` aur `load_resources: true` dependencies docs ke mutabik phir bhi bhejni hoti hain. Extra content parsing/density jaise separate options ka addition alag hai.

Scope inputs: required `target`, `max_crawl_pages`; optional `start_url`, `priority_urls`, `max_crawl_depth`, sitemap settings, subdomain settings, robots controls, crawl delay, cookies, browser/JS/resource options, raw HTML, checks/custom JS, parsing/density, pingback aur tag. Complete set appendix mein hai.

Cost-conscious audit:

- Pehle 50–200 representative pages base mode mein inspect karo, phir required scope tak expand.
- Homepage, category, product, blog, pagination aur JavaScript-dependent templates ka representative sample lo.
- `max_crawl_pages` ko actual budget cap se derive karo; URL query loops aur unlimited pagination control karo.
- Sitemap-only crawl ke liye reviewed docs `respect_sitemap: true` dependency batati hain. `respect_sitemap` enable hone par `max_crawl_depth` ignored ho sakta hai.
- JS rendering un pages par enable karo jahan content/links runtime se generate hote hain ya CWV lab diagnostics chahiye.
- `enable_content_parsing` aur `calculate_keyword_density` tabhi jab output consume karoge.
- `store_raw_html` upfront enable karo agar later raw HTML chahiye; retrieval flag ki dependency pehle decide karo.
- Existing crawl `id` se summary/pages/resources/links/duplicates/redirects/non-indexable/microdata/parsing reports nikalo. Har report ke liye new crawl mat chalao.
- `force_stop` ongoing waste rok sakta hai; already crawled pages ko undo/refund samajhkar use mat karo.

Example mixed strategy: 10,000 base pages = $1.50 plus 200 pages ka **separate** browser-rendered pass = $1.02; total $2.52 vs all 10,000 browser rendered $51 (~95.1% lower). Coverage tradeoff clear hai: sirf 200 pages rendered honge. Do passes mein 200 pages ka base scan duplicated hai aur is calculation mein included hai.

Lighthouse published **$0.005/task** both Standard and Live; **har API mein Standard cheaper nahi**. Rendered OnPage/Lighthouse lab observations ko real-user CrUX field metrics samajhkar report mat karo. Docs legacy FID terminology use kar sakti hain; modern CWV interpretation independently current standards ke saath align karo.

Source: https://dataforseo.com/pricing/on-page/onpage-api and https://docs.dataforseo.com/v3/on_page/task_post/

## 12. Business Data, Merchant, App Data aur Domain/Content APIs

### Business Data

Listings Search database discovery ke liye; My Business Info fresh profile collection ke liye. Google Maps SERP local rankings aur business listings database same deliverable nahi. Location/category filters pehle lagao; known `place_id`/`cid` preserve karo. Google identifiers ko large integer mein unsafe cast mat karo: strings use karna supported fields mein safer hai.

Google Reviews standard billing block 10; Extended Reviews 20; Trustpilot reviews 20; Tripadvisor reviews 10; Q&A 20 in current pricing snapshots. `depth: 21` ko 21 individual review price assume mat karo. Sort, language, duplicate IDs aur incremental state track karo.

**Extended Reviews identifier trap:** docs keyword search par standard Reviews rate ka **3×**, `cid`/`place_id` par **2×** batati hain; pricing page surcharge ko task/identifier style mein explain karti hai. Known identifiers ambiguity aur documented premium dono kam kar sakte hain. Full formula source ke mutabik validate karo. Standard Reviews vs Extended Reviews ko reviews-per-block aur returned fields ke saath compare karo.

Hotels ke dates, adults/children, occupancy, currency aur location settings result-defining hain. Cache mein check-in/check-out ko ignore karke old price display mat karo. Review freshness aur full historical completeness alag goals hain; newest-first capped refresh sirf incremental monitoring ke liye suitable ho sakta hai.

### Merchant

Google Shopping: products, sellers, product info, reviews aur Sellers Ad URL. Amazon: products, ASIN aur sellers. Marketplace location/language, product ID/ASIN, currency, filters, sorting aur depth preserve karo.

Google Shopping `get_shops_on_google: true` reviewed Sellers docs mein cost double karta hai. `search_param` ko `price_min`, `price_max`, `sort_by` override/ignore kar sakte hain. Old/new Shopping markup aur billing block differ ho sakte hain; docs/price comparison zaroori hai. Sellers Ad URL GET **paid endpoint** hai: current page $0.000001/URL batati hai. “GET always free” galat rule hai.

### App Data

Google Play search default block 30, list block 100, reviews block 150; Apple search/list block 100, reviews block 25 in pricing/reference snapshots. Max depth endpoint-specific hai. Same `depth` both stores par same bill nahi deta.

`app_id`, location, language, category, list collection, review sorting/date and pagination settings verify karo. Database `app_listings/search/live` discovery aur fresh app profile scrape alag freshness/price hain. Aggregate App Metrics aur competitor shortlist ke liye Labs app endpoints useful ho sakte hain.

### Domain Analytics

Technology aggregation/summary/stats, technology filters, domains by technology/HTML terms, domain technology lookup aur WHOIS overview cover hain. Domain Technologies published $0.012/task; other technology endpoints $0.012 + $0.0012/returned item. WHOIS $0.12 + $0.0012/item in current pricing. Exact cap, required input, location filters aur database timestamps verify karo; private/redacted WHOIS fields absent ho sakte hain.

### Content Analysis

Search, Summary, Sentiment, Rating Distribution, Phrase Trends aur Category Trends reference mein hain. Aggregates chahiye toh thousands of content rows fetch karke locally summarize karna avoid kar sakte ho. Published $0.024/task + $0.000036/row; exact endpoint response unit inspect karo. Sentiment model inference ko manually verified fact label mat karo.

### Database exports

Backlink summary, Google/Bing keyword/SERP/historical data, Amazon products, app stores, WHOIS aur business listings database products cover hain. Bahut large repeated exports ke liye API per-row total vs bulk database purchase/delivery compare karo. Current licensing, update cadence, schema, date partition, file format aur price official product/quote se verify karo. Is research mein koi purchase nahi hua.

## 13. AI Optimization: token cost ko ignore mat karo

LLM Responses provider API output hai; LLM Scraper consumer UI output capture karta hai; LLM Mentions indexed prompt/mention dataset hai. Inmein model, geography, source coverage, citations aur freshness different ho sakti hai. Ek ko doosre ka exact ground truth mat samjho.

Published snapshots:

- LLM Responses Standard: **$0.0002 task fee + $0.01 advance payment**; provider/model usage ke final reconciliation ko full pricing FAQ se samjho: usage $0.01 se kam ho toh difference refund; 72 hours mein failed task ke liye $0.01 advance refund documented hai. $0.0102 ko guaranteed fixed final cost mat banao.
- LLM Responses Live: **$0.0006 + provider LLM price**.
- LLM Scraper Standard/Priority/Live: **$0.0012 / $0.0024 / $0.004 per result**.
- LLM Mentions: **$0.10/task + $0.001/row** on current pricing page. Lite endpoint ko sirf naam ki wajah se free/discounted assume mat karo; shape/coverage compare karo.
- AI Keyword Search Volume: **$0.01/task + $0.0001/row**.

Use `models` discovery to choose supported model names/features. Parameters reviewed for ChatGPT Responses include `user_prompt` (reviewed ChatGPT limit 500 characters), `model_name`, `max_output_tokens`, `temperature`, `top_p`, `web_search`, `force_web_search`, `message_chain`, callbacks/tag as supported. Exact names/required fields model/provider se verify karo.

Cost tips:

1. Prompt short aur specific rakho; repeated long histories na bhejo.
2. Output ko required task ke hisaab se constrain karo. Reviewed ChatGPT defaults `max_output_tokens: 2048`; min reasoning 1024/non-reasoning 16; max 4096. **Live docs warning: reasoning/web search cases actual output specified cap exceed kar sakta hai.** Isse hard wallet cap mat banao.
3. `web_search` tabhi jab research needs current web sources; model support mandatory.
4. `temperature` aur `top_p` reviewed endpoint mein ek saath supported nahi; reasoning model mein temperature unsupported ho sakta hai.
5. Existing Mentions database se answer milta ho toh naye expensive prompts scrape karna avoid kar sakte ho; indexed coverage tradeoff accept karke.
6. Brand portfolio ke liye supported Multi-Target Metrics vs individual calls compare karo; max inputs aur returned units verify karo.
7. Scheduled monitoring ke liye Standard, on-demand immediate output ke liye Live; LLM Responses Standard turnaround up to 72 hours ho sakta hai.
8. `cost` includes provider usage where documented; `money_spent` ko cost ke upar dobara add mat karo.

Sources: https://dataforseo.com/pricing/ai-optimization/llm-responses and relevant Models/Live endpoint reference.

## 14. Free sandbox, `.ai` responses aur free recovery tools

### Sandbox ka maximum use

Registered users ke liye sandbox **free dummy data** deta hai. Same payload/schema workflows test kar sakte ho; real rank/search-volume validation ke liye useful nahi. Callback handling bhi test kar sakte ho.

Special SERP example all-feature fixture:

`GET https://sandbox.dataforseo.com/v3/serp/google/organic/task_get/advanced/00000000-0000-0000-0000-000000000000`

Source: https://docs.dataforseo.com/v3/appendix/sandbox/

**Free unlimited trial ka matlab unlimited real paid data nahi.** Current Help Center registration par $1 test credit, expiry-free use period batata hai; real calls ki quantity credit se limited hoti hai. Production API Playground calls bhi real requests ho sakti hain.

### `.ai` output ka actual benefit

Supported Live/Task GET endpoints mein `.ai` suffix cropped JSON deta hai, no extra feature charge documented. Empty/null/false fields remove, floats third decimal tak round, position/xpath remove, monthly searches compact, default depth/limit 10; location list country-only by default.

Yeh **LLM token/network/storage savings** de sakta hai, automatic DataForSEO unit-price discount nahi. Defaults badalne se result quantity/bill indirectly badal sakta hai. Important: service envelope mein `cost` remove ho sakta hai; financial ledger ke liye normal JSON response/account telemetry chahiye. Missing field `.ai` compression ki wajah se ho sakta hai. False boolean remove hone se “not present” aur “false” ki distinction consuming app mein preserve/interpret karni hogi. Rounding financial/precision-sensitive analysis ke liye unsuitable ho sakti hai.

Source: https://docs.dataforseo.com/v3/appendix/ai_optimized_response/

### Recovery tools

- Existing task ID se supported Task GET repeat retrieval: queued result ko dobara scrape karne ki zaroorat nahi; billing exact endpoint se verify karo.
- `/$api/id_list`: free ID/metadata recovery; six-month ID lookback, metadata one month; older metadata availability ko result retention se confuse mat karo.
- `/appendix/webhook_resend`: up to 100 IDs, **free resend; not double-charged**. Failed callback par new paid task create karna usually unnecessary.
- User Data: account balance/usage monitoring. Account-wide totals shared accounts mein project attribution nahi dete; own tags/ledger chahiye.
- API Errors / Tasks Fixed / Status: incident recovery ke liye before resubmitting everything.

## 15. Retention aur freshness policy

Current docs introduction: Standard JSON generally **30 days**; most Live results stored nahi; SERP JSON **Standard aur Live dono 30 days**; HTML generally **7 days**. Help Center retention article generic Live non-storage bolta hai aur newer docs ka SERP Live exception clearly list nahi karta. **SERP Live exception ko docs ke saath note karo, phir bhi response local persist karo.**

SERP screenshot source task eligibility 7 days; returned screenshot URL Help Center ke mutabik 1 day accessible. Images timely download/store karo. Task JSON retention aur screenshot file lifetime alag hain.

Old task result ko aaj retrieve karna usko fresh crawl nahi banata. `retrieved_at`, source task datetime, collection timestamp aur source update time separate store karo. Labs keyword metrics monthly updates ho sakti hain; other database refreshes 30–90 days range mein pricing FAQ batati hai, endpoint-specific exceptions hain. Backlinks continuous crawling bhi every page freshly crawled “abhi” hone ki guarantee nahi.

Sources: https://docs.dataforseo.com/v3/ and https://dataforseo.com/help-center/how-long-do-you-keep-results

## 16. Production architecture jo duplicate spending rokta hai

Suggested flow:

`request → validation → canonical query key → cache → in-flight dedupe → budget reservation → provider submit → persist task ID → wait/get → validate → persist raw result → derive reports → reconcile cost`

Canonical key mein endpoint, semantic payload, location, language, device, OS, time range, engine/model, depth/scope, filters/sort, result options aur freshness bucket include karo. Callback URL aur random `tag` jaise non-semantic delivery fields ko carefully exclude kar sakte ho. Doosri taraf currency, hotel dates, search partners, rank scope, clickstream flag jaise meaning-changing fields ko ignore mat karo.

Bulk result reuse ke liye task-level cache ke saath item-level lookup useful hai: e.g. keyword+location+date settings ke stored metrics. Bulk arrays ko sort/dedupe tabhi karo jab endpoint mein order/duplicate semantics irrelevant hon. Seed order/cross-keyword comparison change ho sakta hai; generic normalizer se query meaning change mat karo.

Suggested app-level cache TTLs, **recommendations only; official guarantees nahi**:

| Data | Starting refresh policy | Kab faster refresh chahiye |
|---|---|---|
| Metadata: locations/languages/categories | Weekly or monthly, source-change aware | Newly added locations/features |
| Monthly Ads/Labs keyword metrics | Source update detected hone par | Freshness-sensitive monthly launch |
| Ranking SERPs | Business requirement: daily/weekly | Volatile high-value keywords |
| Backlink aggregates | Daily/weekly | Active link monitoring |
| Technologies/WHOIS | Weekly/monthly | Acquisition/security/research need |
| Hotel/product price | Minutes/hours or on-demand | Booking/current price decisions |
| Reviews | Incremental cadence by volume | Reputation alert use case |

Concurrency lock ke bina do users same uncached query simultaneously hit karke double task create kar sakte hain. In-flight dedupe + cache-warming queue use karo. Same cached SERP se all tracked competitor domains, features aur relevant URLs derive karo; har competitor ke liye same keyword/geo/device scrape duplicate hai.

Job states useful hain: `planned`, `reserved`, `submitting`, `submitted`, `pending`, `ready`, `retrieved`, `partial`, `failed`, `unknown_submission`. Network timeout after POST mein task create hua ho sakta hai; **blind automatic re-POST duplicate charge kar sakta hai**. Durable submission intent + tag/time window/ID List reconciliation use karo; ambiguity resolved na ho toh budgeted bounded retry policy rakho.

### Minimal cost ledger fields

`tenant_id`, `project_id`, `job_id`, canonical key, endpoint, normalized request, `tag`, task ID, mode, submitted/completed timestamps, requested units, returned units, estimated reserve, actual reported cost, refunds/adjustments where available, status, retry reason, cache hit/miss, source timestamp.

Use Decimal/cents-compatible precise storage for billing totals; float rounding aur `.ai` third-decimal truncation se financial ledger corrupt na karo. Account balance differences mein concurrent unrelated jobs/refunds bhi ho sakte hain.

### Budget rules

- Daily account/API/endpoint limits dashboard par configure karo; documented `40203` cost limit error handle karo.
- Per customer/project/job app-level budget aur max rows/pages/retries rakho.
- Submission se **pehle** worst-plausible cost reserve karo; concurrency mein atomic reservation chahiye.
- Spend thresholds par queue stop/degrade karo; arbitrary cheaper data ko same freshness label ke saath substitute mat karo.
- Planned `keyword × location × language × device × pages × runs` cardinality estimate dikhao before large collection.
- Recurring workflows mein dry-run planner sirf request counts/cost estimates banaye; actual paid API hit na kare.

## 17. Errors aur retry decision table

| Code/status | Meaning | Action |
|---|---|---|
| HTTP 200 + `20000` | Success at that level | Per-task status, result shape aur item count bhi check |
| `20100` | Task Created | ID save karo, wait/get; resubmit mat karo |
| `40601` / `40602` | Handed / queued | Existing task wait/poll |
| `40102` | No search results | Valid empty/business outcome distinguish karo; blind loop mat chalao |
| `40106` | Partial results | Available data save; missing pages charged nahi per docs; only gap recovery if needed |
| `40006` | Too many tasks in request | Chunk correctly; succeeded tasks dobara mat post karo |
| `40501` | Invalid field | Payload fix; same invalid request retry mat karo |
| `40200` | Payment required | Spending stop; account state inspect |
| `40202` | Rate limit exceeded | Throttle/backoff+jitter; applicable headers inspect |
| `40203` | Configured cost limit exceeded | Limit respect karo; auto-increase mat karo |
| `40209` | Too many simultaneous queries | Concurrency reduce karo |
| `40401` / `40402` | Task missing / invalid path | ID/method/path/retention verify |
| `40103` | Execution failed, resubmit suggested | Bounded retry after preserving error/account state |
| `50000` / `50301` / `50401` | Internal / upstream / timeout | Endpoint-aware bounded backoff; ambiguous submit reconcile |

Official error index: https://docs.dataforseo.com/v3/appendix/errors/

Webhooks ko fast acknowledge karo: reviewed task docs server response ke liye **10 seconds** batati hain. Gzip decode, durable enqueue/store, idempotent processing task ID par, phir heavy computation worker mein. Acknowledging before any durable record se result lose ho sakta hai. Callback down ho toh Tasks Ready/ID recovery/Webhook Resend pehle; paid re-crawl last resort.

Tasks Ready ko permanent full history mat samjho: already retrieved/delivered tasks list se hat sakte hain. Own task registry retain karo. Polling frequency product latency/volume se align karo; every pending task every second poll karna unnecessary load hai. Free GET polling kam karna usually provider charge saving nahi; infrastructure/request-cap saving hai.

## 18. Ready-to-use money-saving recipes

### A. Daily rank tracker

Same semantic keyword/geo/language/device request dedupe → Standard normal priority → required depth only → supported stop-on-match if full competitor SERP unnecessary → IDs persist → callbacks/Tasks Ready → one advanced result se all competitor tracking → raw result store → daily freshness cache.

### B. Keyword research tool

Small seed discovery → server filters/shortlist → existing returned metrics reuse → remaining Search Volume inputs exact dedupe + compatible grouping + up to 1,000 Google Ads keywords/task → synonyms distribute if separate reporting required → update-aware refresh → live SERP only shortlisted high-value keywords.

### C. Backlink opportunity tool

Bulk domain metrics → shortlist → competitors/intersections → required referring domains/pages → detail rows only where needed → search-after pagination → date/status-aware refresh. Summary counts ko recreate karne ke liye all backlinks export mat karo.

### D. Technical audit

Budgeted base crawl → template-level classification → optional reports from same task → JS/browser only required templates → expensive screenshots only final issues → store evidence → same audit render/download on refresh par naya crawl mat trigger karo.

### E. Local SEO/reviews monitor

One discovery step se identifiers save → exact entity task → normal queue → newest-first capped review refresh where supported → unique review ID dedupe → periodic broader reconciliation if completeness needed. Geo-grid monitoring mein har grid point billable task ho sakta hai; grid resolution demand ke hisaab se choose karo.

### F. AI visibility dashboard

Mentions database coverage check → multi-target/aggregate metrics where suitable → required model/location prompt set only → stable weekly scheduled scraper if freshness allows → paid web search/token-heavy generations sampled → provider usage reconcile. Automated repeat prompts statistical variation bhi laate hain; unnecessary reruns ko “validation” mat banao.

## 19. 25 rules jo implementation mein enforce karne layak hain

1. Sandbox mein parser/workflow build karo.
2. Paid Playground ko free sandbox mat samjho.
3. Standard use karo where adequate and actually cheaper.
4. `priority: 2` sirf explicit latency need par.
5. SERP depth default se badhane ka budget reason ho.
6. Operator-containing keywords ko cost classifier flag kare.
7. SERP paid options default off, features required hone par on.
8. Same SERP se sab domains ki ranking derive karo.
9. Supported stop-on-match sirf suitable incomplete-SERP workflows mein.
10. Google Ads compatible task batches near 1,000 keywords.
11. Similar variants ke grouping semantics preserve karo.
12. Labs keyword overview cap 700 ya exact endpoint cap follow karo.
13. `include_clickstream_data` double-cost toggle explicit ho.
14. Existing metrics ko baar-baar enrich mat karo.
15. Required total rows ke liye task overhead optimize karo.
16. Extra unused rows ko savings mat bolo.
17. Backlinks aggregate/bulk before detail.
18. OnPage page cap mandatory; browser rendering scoped.
19. Known product/app/business IDs reuse karo.
20. Live-only result immediately persist karo.
21. Every created task ka ID durable store karo.
22. Ambiguous paid POST par blind retry mat karo.
23. Failed callback ke liye free resend/recovery first.
24. Actual task cost + account ledger reconcile; envelope total double-count mat karo.
25. Pricing changes aur source disagreements explicit track karo.

## 20. Abhi kya verify nahi hua

User-specific account pricing/discounts, credit/payment terms, real API outputs, callback delivery on your server, paid-parameter conflicting billing, source latency under your workload, exact current model/provider charges, bulk database purchase quotes aur application-specific requirements verify nahi hue. Is guide ke implementation examples execute nahi hue. Yeh research aur reference artifact hai, production integration ka tested claim nahi.

Persistent learning is saved in the files; future chat sessions ko automatic permanent memory ka promise nahi. Repo mein is guide aur exact endpoint reference ko future task context mein padhne se sources aur findings reuse ho sakti hain.
