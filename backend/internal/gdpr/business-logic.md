# Account deletion rules

- The current signed browser session selects the account. The auth wrapper validates the origin, and the backend rechecks session freshness under lock.
- Only sole-owner workspaces can be erased by this endpoint. Workspace locks protect the membership check from concurrent joins. Billing checkout and deletion use the same per-workspace advisory lock.
- Account and workspace records are erased in one database transaction. Open subscription or pack orders stop the operation, and the payment ledger is retained. The erasure does not call external providers.
- An optional reason increments a category total in the erasure transaction. A failed or retried deletion cannot add an extra response. Staff can see aggregate totals, which contain no account identifiers or individual submissions.
