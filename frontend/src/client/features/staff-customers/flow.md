# Staff customers: flow

1. The sidebar asks `/api/v1/admin/access` once per session (no retry) and shows **Customers** only for staff.
2. The page loads `/api/v1/admin/customers` (no retry); non-staff get the 403 message instead of a table.
3. Search filters the loaded rows in the browser by email, name, workspace, plan, domains and onboarding answers.
4. **Download CSV** is a plain link to `/api/v1/admin/customers.csv`; the session cookie authorises it and the server sets the file name.
5. No paid provider is called; the server enforces access, the link visibility is only convenience.
