# Engineering documentation

Code and tests are the source of truth. Feature docs are colocated with their source: each `backend/internal/<package>/` and `frontend/src/client/features/<feature>/` has `README.md`, `flow.md`, and `decision.md`. Start there, then inspect the implementation. The links below identify cross-cutting material.

## Navigation

- [Source and test file map](FILE_MAP.md)
- [Architecture and request flow](architecture/OVERVIEW.md)
- [DataForSEO cost boundary](integrations/DATAFORSEO.md)
- [Security model](security/MODEL.md)
- [Regression testing](testing/REGRESSION.md)
- [API contracts](api/CONVENTIONS.md)
- [Local setup](LOCAL_DEVELOPMENT.md), [deployment](DEPLOY.md), [product](PRODUCT.md)

- [Feature wiring review](audits/2026-10-20-feature-wiring-review.md)

## Repo-wide guidance and references

`README.md`, `CLAUDE.md`, `AGENTS.md`, `.claude/CLAUDE.md`, `ROADMAP.md`, `DECISIONS.md`, `.agents/PAPERCUTS.md`, `backend/README.md`, `backend/internal/analytics/geo/ATTRIBUTION.md`, `backend/internal/sam/product_fact_sheet.md`, `frontend/src/client/features/ai-mcp/agentSetupPrompt.md`. Historical reviews are in `docs/audits/`.

Cross-cutting documents: `docs/FILE_MAP.md`, `docs/architecture/OVERVIEW.md`,
`docs/integrations/DATAFORSEO.md`, `docs/security/MODEL.md`,
`docs/testing/REGRESSION.md`, `docs/api/CONVENTIONS.md`,
`docs/LOCAL_DEVELOPMENT.md`, `docs/DEPLOY.md`, `docs/PRODUCT.md`.

Implemented feature entry points: [Backlinks backend](../backend/internal/backlinks/README.md) and [UI](../frontend/src/client/features/backlinks/README.md); [Domain backend](../backend/internal/domain/README.md) and [UI](../frontend/src/client/features/domain/README.md); [Keyword research](../backend/internal/keywords/README.md); [Rank tracking](../backend/internal/ranktracking/README.md); [AI search](../backend/internal/aisearch/README.md); [Site audit](../backend/internal/audit/README.md); [Content editor](../backend/internal/contenteditor/README.md); [Billing](../backend/internal/billing/README.md); [Authentication](../backend/internal/auth/README.md). Other packages and client features follow the same colocated README/flow/decision convention; `ROADMAP.md` distinguishes current and planned functionality.

Update this index for new cross-cutting docs. The docs-map test enforces repo-wide Markdown discoverability, local index link resolution and the feature document triplets. Human review must still verify claims against code and keep feature descriptions current.
