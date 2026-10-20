# IndexNow screen: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Decisions to preserve

- **Nothing outbound on page load.** Checking the key file reaches the
  customer's site and counts against the hourly limit, so it runs only on a
  click. _Why:_ an outbound call must follow an explicit user action.
- **The key file is built in the browser.** _Why:_ no download route to
  maintain; the key is not a secret.
- **Honest copy about Google.** _Why:_ Google does not use IndexNow, and a
  public promise the product does not keep is a bug (`CLAUDE.md`).
- **The plans link is a convenience; the server enforces the plan.**
