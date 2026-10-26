# Credits audit (25 October 2026)

Question from the owner: "Are credits deducted correctly, or will a bug make
me lose money?" This audit reads the spend path, runs its tests on a real
Postgres, mutates the code to see whether tests catch it, and reads
production totals (read-only `SELECT`, aggregates only). No code changed.

## Verdict

- **Owner losing money: low risk today.** Every recorded provider dollar was
  charged back to a workspace at the 1.28 markup or more.
- **Customer paying more than shown: real gap.** A paid action shows "about
  N credits" but the charge has no cap at N, only at the hold ceiling, which
  can be several times higher (F1).

## Evidence

| Check                                  | Result                                                                                                                                                                                                                                 |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Production provider spend (14–25 Oct)  | $4.24 total in `go_provider_spend`: DataForSEO $4.17 (matches `go_dataforseo_usage`) + OpenRouter $0.065. 3 workspaces, 3 `active` subscriptions.                                                                                      |
| Charged back to customers              | Settled credits ≥ provider cost × 1,280 in every feature (e.g. AI visibility $2.54 → 3,260 credits vs 3,248 needed; all features 5,500 vs 5,426).                                                                                      |
| Stuck holds                            | 0 open reservations, 0 uncertain provider holds.                                                                                                                                                                                       |
| Tests                                  | `billing`, `platform/spend`, `platform/dataforseo`, `platform/openrouter` pass against real Postgres + Redis (142 billing tests, none skipped).                                                                                        |
| Mutation check (break code, see a red) | 9 of 9 caught: markup removed; replay not refused; failed tasks charged; budget cap removed; "nothing delivered" charged; no-credit not refused; daily brake off; Brand Lookup hint lowered; balance row lock + advisory lock removed. |
| Automatic retry of paid calls          | None: every paid DataForSEO POST passes `retrySafe=false`; only free GETs retry.                                                                                                                                                       |
| Free plan                              | 0 monthly credits, so a free signup cannot spend provider money.                                                                                                                                                                       |
| Double click in one tab                | Same `Idempotency-Key` (`frontend/src/client/lib/uplumoApi.ts`) → one reservation; the second click is refused as a replay.                                                                                                            |

## Findings

**F1. Charge can exceed the cost shown (owner decision).** The hint is a
typical figure (`billing/costs.go`); the charge is the real cost × 1.28,
capped only at the hold. Brand Lookup shows "about 372 credits"; its hold is
`(1 + 2×platforms [+ platforms with competitors]) × $0.25`
(`aisearch/service.go`): up to 7 × $0.25 = $1.75 = **2,240 credits** in the
US with competitors, 1,280 elsewhere. The confirm dialog says "a little more
or less". `CLAUDE.md` says never charge more than shown without the owner's
approval. The real worst case is below the hold but has not been measured.

**F2. The hold can refuse a paying customer who could afford the real cost.**
A hold must fit the balance, so a Solo workspace with under 2,240 credits is
refused a US Brand Lookup with competitors, though the measured cost is
~370. Same root as F1: holds far above real prices.

**F3. Uplumo absorbs some costs by design (small, logged only).** Provider-
billed failed tasks, work above a `DoBudgeted` ceiling, "nothing delivered"
results and a shortfall when the balance runs out are paid by Uplumo and
logged (`billing/spendguard.go`, `platform/dataforseo/client.go`), but not
summed anywhere the owner can query. Today the totals above show no net loss.

**F4. No server-side dedupe without an `Idempotency-Key`.** Two tabs, MCP and
API-key clients send no shared key, so only each feature's cache stops a
double purchase. To verify in the DataForSEO audit (next).

**F5. Three `active` subscriptions while there are no paying customers.**
Owner to confirm these are his own or test workspaces.

Known and already disclosed: Business at 1,000 keywords, desktop, depth 40,
weekly, can need 15,400 credits in a five-run month vs. its 15,000 grant
(`billing/README.md`).

## Not done

- No comparison with the DataForSEO or OpenRouter dashboards (owner access);
  the owner should check that the provider invoices match $4.17 / $0.065.
- No live paid call was made.
- SAM (`ai_assistant`) has no production spend yet, so its metering is only
  test-verified.
