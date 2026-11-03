# Staff console

`/staff` is the staff-only business console with its own shell (no product sidebar): Overview, Subscriptions, Customers, Marketing, Finance and Activity. The app sidebar shows **Staff console** only when `/api/v1/admin/access` says `staff: true`; anyone else opening `/staff` sees the normal not-found page, and the Go API answers 404.

- `StaffLayout.tsx`: auth gate, staff check, tab navigation. Routes in `src/routes/staff/`.
- `OverviewPage.tsx`, `SubscriptionsPage.tsx`, `MarketingPage.tsx`, `FinancePage.tsx`, `ActivityPage.tsx`: read-only views of `backend/internal/staff`.
- `CustomersPage.tsx` + `UserActionDialog.tsx`: user table, CSV export, Block / Unblock / Delete.
- `staffApi.ts` (console contract, money formatting) and `customersApi.ts` (customer list, staff access).
- `StaffParts.tsx`: stat tiles, sections, count bars, query states.
