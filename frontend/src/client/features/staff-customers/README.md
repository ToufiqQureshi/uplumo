# Staff customers

`/admin/customers` shows every signed-up user with onboarding answers, plan, websites and Google connections, with search and a CSV download for marketing. Only Uplumo staff see it; the sidebar shows **Customers** when `/api/v1/admin/access` says `staff: true`.

- `CustomersPage.tsx`: table, client-side search, CSV link.
- `customersApi.ts`: Zod contract and TanStack Query options for the Go API in `backend/internal/customers/`.
- Route: `src/routes/_app/admin.customers.tsx`; sidebar link: `staffNavItem` in `src/client/navigation/items.ts`.
