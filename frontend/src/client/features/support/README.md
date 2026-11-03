# Support reports

The Help page hosts `ReportProblemForm` and retains the support email fallback.
Customers choose Bug, Credits or payment, or Suggestion, describe the issue and
can attach one PNG/JPEG screenshot up to 1 MiB. A saved report returns a ticket
number. `reportApi.ts` owns the authenticated Go API contract.

`ReportProblemForm.test.tsx` covers retry identity, input retention, duplicate
submits and screenshot/path limits. `tests/support-report.spec.ts` checks the
real rendered form at desktop and mobile widths with mocked server responses.
