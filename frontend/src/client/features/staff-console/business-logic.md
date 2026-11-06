# Staff console: decisions

- **Own shell, no product navigation.** _Why:_ the owner wants business data in one place without Uplumo features around it. The link in the app sidebar is the only way in, and only staff see it.
- **Not-found for non-staff.** _Why:_ the console should not advertise itself; the server enforces access (`backend/internal/staff/business-logic.md`), the UI only mirrors it.
- **Real numbers only.** Revenue comes from the payment ledger; MRR is active subscriptions at catalog price, labelled as such. INR and USD are shown side by side, never converted. _Why:_ owner asked for no estimates or mocks.
- **Destructive actions need a dialog; delete needs the typed email.** _Why:_ deletion cannot be undone. Block explains that a paid plan keeps running.
- **Charts are minimal.** One single-series bar chart (signups per day) with tooltip; everything else is numbers, tables and share bars. _Why:_ the owner reads totals, not trends, and tables export cleanly.
- Customer list search is client-side; move to server paging when users pass a few thousand.
