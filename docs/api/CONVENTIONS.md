# API conventions

`backend/internal/httpapi/httpapi.go` mounts routes and middleware. Project routes require a signed-in user with organization membership; handlers validate and bound request bodies before service or provider work. Feature-specific handlers define status mappings, including 400 validation, 402 paid-plan gating where configured, and 502/503 provider failures. Frontend requests use `frontend/src/client/lib/uplumoApi.ts` and feature API clients; response contracts are checked with Zod.

OpenAPI files under `backend/api/` describe supported contracts where present, including billing, audit, keywords, rank tracking and AI tracking. Backlinks and Domain route details currently live in their Go handlers and feature READMEs; do not assume there is a generated schema for those paths. When changing a route, update its applicable YAML, client contract, handler tests and feature flow. A generated API documentation pipeline is not configured for every route.

Shared frontend requests retain POST replay keys through invalid successful responses. Failed queries do not retry on remount by default; user-initiated refetch remains available. The existing one-minute in-memory key lifetime does not provide cross-reload persistence.
