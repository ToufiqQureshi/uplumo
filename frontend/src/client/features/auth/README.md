# Sign-in and account access

Frontend sign-up, sign-in, verification, password recovery, Google sign-in and hosted-route guards.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `AuthPage.tsx`
- `useGoogleAuth.ts`
- `useHostedAuthRouteGuard.ts`
- `backend/internal/emailauth`
- `backend/internal/auth`

Owner website analytics stays off on initial credential-link pages. The router
notifies the shared loader after navigation so a later dashboard/checkout page
can initialize tracking without reloading. See `decision.md` for this boundary.
