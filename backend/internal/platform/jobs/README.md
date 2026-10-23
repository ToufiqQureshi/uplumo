# Postgres background job queue

Enqueues, leases, retries and completes asynchronous jobs with optional idempotency keys.

Internal package guide: [flow.md](flow.md) describes the code path; [decision.md](decision.md) records invariants to preserve.

## Source map

- `queue.go`
- `worker.go`
- `queue_test.go`
- `queue_integration_test.go`

`Worker.RunSupervised` restarts polling five seconds after a database/lease failure, stops on cancellation or invalid configuration, and logs the affected queue. Audit, rank tracking and AI tracking share it. It does not bypass job attempt limits or replay uncertain provider submissions; handlers retain their own idempotency/checkpoints. Recovery is tested with an injected pool-acquisition failure.
