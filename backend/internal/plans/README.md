# Plans

The one list of what each Uplumo plan costs, limits and includes
(`catalog.go`). Nothing else in the repository states a price or a limit.

| Plan     | Price a month | Sites | Tracked keywords | Members | Monthly credits | Adds                                                                                     |
| -------- | ------------- | ----- | ---------------- | ------- | --------------- | ---------------------------------------------------------------------------------------- |
| Free     | ₹0            | 1     | none             | 1       | 0               | Uplumo Analytics with AI traffic detection, starter site audit                           |
| Basic    | ₹999          | 1     | 100              | 1       | 4,000           | Keyword research, rank tracking, Google data, full site audit, AI writing, IndexNow, SAM |
| Pro      | ₹2,499        | 10    | 500              | 3       | 10,000          | Backlinks, competitor research, AI Visibility, credit packs                              |
| Business | ₹4,999        | 20    | 1,000            | 5       | 15,000          | White-label reports, site add-on                                                         |

The ids (`basic`, `pro`, `business`) predate the names and stay: stored
subscriptions and `RAZORPAY_PLAN_ID_*` use them. Agency was retired on
27 Oct 2026; a stored `agency` row reads as Free and unlocks nothing.

## Packs (one-time purchases)

`Packs()` lists them; `Pack.Feature` is what the plan must include to buy.

| Pack            | Price  | Adds              | Lasts                                          | Who      |
| --------------- | ------ | ----------------- | ---------------------------------------------- | -------- |
| `credits_5000`  | ₹1,499 | 5,000 credits     | while the paid plan is active, ends with it    | Pro, Business |
| `credits_15000` | ₹3,999 | 15,000 credits    | while the paid plan is active, ends with it    | Pro, Business |
| `sites_5`       | ₹999   | 5 more websites   | `SiteAddOnDays` (30) from payment              | Business |

`internal/billing` sells and grants them (`topups.go`).

## Who reads it

- `internal/site` builds the pricing cards and the comparison table from it.
- `internal/billing` serves it to the Billing screen
  (`GET /api/v1/billing/plans`) and answers every plan check from it.
- The React app has no price of its own.

## Rules that must never break

- **A pack never sells a credit for less than the cheapest plan that may
  buy it** (`TestCreditPacksKeepTheMargin`), and never costs Uplumo more
  than a third of its price at the provider.
- **Never show what is not enforced.** A line in `Highlights` must be backed
  by a `Feature` or a limit that some code checks. A promise with no check is
  a refund waiting to happen.
- **The Razorpay plan amount must equal `PriceINR`.** Changing a price means
  a new Razorpay plan and a new `RAZORPAY_PLAN_ID_*` value, in the same deploy.
- **A feature belongs to the lowest plan that includes it and to every plan
  above.** The lists are built that way; keep them so.
- `Unlimited` is for Uplumo's own workspaces. It is not in the catalog and is
  never sold.

## Where each feature is checked

| Feature          | Checked in                                                        |
| ---------------- | ----------------------------------------------------------------- |
| `GoogleData`     | `internal/google`: starting a new Search Console or GA4 connection |
| `Keywords`       | `internal/keywords` research, `internal/ranktracking` metric refresh |
| `RankTracking`   | `internal/ranktracking` manual and scheduled checks               |
| `FullAudit`      | `internal/audit`: the paid audit tier                             |
| `Publishing`     | `internal/sitepublish`: connecting a website and publishing to it |
| `Backlinks`      | `internal/backlinks`                                              |
| `DomainResearch` | `internal/domain`                                                 |
| `AIVisibility`   | `internal/aisearch`                                               |
| `AIWriting`      | `internal/contenteditor` writing assistant                        |
| `WhiteLabel`     | `internal/branding`: saving and resetting branding                |
| `CreditPacks`    | `internal/billing`: listing and buying credit packs (`topups.go`) |
| `SiteAddOn`      | `internal/billing`: buying the site add-on; `SiteLimit` adds it    |
| `IndexNow`       | `internal/indexnow` (gate wired in `cmd/server/main.go`): setup and submit |
| Sites            | `internal/projects`: create and restore                           |
| Tracked keywords | `internal/ranktracking`: adding keywords                          |
| Members          | `internal/team`: inviting                                         |

## Known gaps

- Only **new** Google connections are stopped. A workspace that downgrades
  keeps reading the properties it already connected.
- The tracked-keyword count is read before the insert, not locked with it:
  two requests at once can pass the limit by what one of them adds.
- A workspace over a limit after a downgrade keeps what it has; it only
  cannot add more.
- Keyword research has no monthly cap of its own on Basic.
- The assistant (SAM) is limited by credits, not by a feature of its own.
  Free has no credits; every paid plan can use SAM from their monthly
  grant when the AI provider is configured.
