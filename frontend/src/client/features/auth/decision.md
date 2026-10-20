# Sign-in and account access: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Why the feature is structured this way

Frontend sign-up, sign-in, verification, password recovery, Google sign-in and hosted-route guards.

## Decisions to preserve

- **Never store plaintext passwords, verification tokens or OAuth secrets in frontend state or logs** _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- **Keep origin/callback allowlists, CSRF protections, CAPTCHA/rate limits and secure cookie rules intact** _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- **A client-side session cache is not an authorization check** — every Go API route checks the session. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.

## Verification pointers

The shared owner analytics loader excludes initial auth/credential-link pages.
After client navigation resolves, `src/router.tsx` emits `uplumo:navigation`
without credentials or user data; the loader can then initialize on an eligible
dashboard/checkout page. Auth-first sessions must not permanently miss app
tracking. This notification changes no auth guard or session behavior. See the
backend site `decision.md` and `scripts/owner-tracking.test.ts` for boundaries.

- `passwordSchema.test.ts`
