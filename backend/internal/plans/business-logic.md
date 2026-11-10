# Plans: website limits and the website add-on

- **Included websites.** Free 1, Basic 1, Pro 3, Business 10 (`Sites` in
  `catalog.go`). New websites stop at this number plus any running add-on
  (`billing.Service.SiteLimit`, enforced in `projects.Repository.CreateLimited`).
- **Website add-on.** Pack `sites_1`: one extra website for ₹999 (or $12),
  valid 30 days from payment (`SiteAddOnDays`). Every paid plan may buy it
  (`SiteAddOn` feature); Free may not. A customer buys one per extra website.
  The price on plan cards, the pricing page and the FAQ is read from the pack,
  so it cannot drift.
- **Lowered allowances keep existing sites working.** Pro went from 10 to 5
  (3 Nov 2026) and to 3 (6 Nov 2026); `PauseAboveSites` stays 10, so a Pro
  workspace that already runs up to 10 sites keeps paid work running. Only
  new sites stop at the current limit. Business works the same with 20.
- **Retired 5-website pack.** `sites_5` (₹999 for 5 websites, Business only)
  is no longer sold. Bought rows still count by the `sites` stored on the
  payment row, so a running purchase lasts its full 30 days.
- **Expired add-ons.** Sites added under an add-on keep working after it ends,
  but no new site fits; add-ons that ran out one after another never stack
  into a permanent allowance (`billing.repository.addOnSites`).
