# SAM project assistant: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/sam/sessions/...
- **Source:** `handler.go`, `chat.go`, `agent.go`, `tools.go`, `repository.go`

## Request or job flow

Authenticated user opens a project session; handler checks access/plan/credits; agent loads compact history/context and calls configured model; each tool call is re-authorized and bounded; turn output and actual usage are persisted/settled.

## Dependencies and cost boundary

DeepSeek model provider (`platform/deepseek`); Postgres sessions/messages; Redis limits; billing; projectcontext and internal services.

## Failure, security and data invariants

- Require a per-turn cost ceiling and stop mid-turn if credits/limits are exhausted.
- Never let a model-selected project ID change authorization scope.
- Persist partial answers and settle real usage on provider/tool error; avoid charging reported-zero usage as free.

Search growth handoff is an editable draft, never an automatic send. A request
with `mode: search_growth` selects a server-enforced read-only tool subset before
streaming. `get_search_growth_report` verifies metrics through the project-bound
MCP runner. Provider completions retain ordinary SAM metering; paid research,
site changes, jobs and memory writes cannot execute in analyst mode.
