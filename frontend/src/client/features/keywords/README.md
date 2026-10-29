# Keyword research and saved keywords (frontend)

## What this does

`/p/$projectId/keywords` is the Keywords agent: the customer asks in plain
words and gets a table, charts, quick wins and a content plan.
`/p/$projectId/saved` is the saved-keywords list. All data comes from the Go
API.

## Flow

- `agent/KeywordAgentPage.tsx`: Free workspaces see an upgrade card and send
  nothing. A paid workspace picks an example or types a question; the first
  send creates a SAM session and streams the reply from
  `POST .../sam/sessions/chat` with `mode: "keywords"` (see
  `backend/internal/sam/README.md`). A ref guarantees the first question is
  sent once, even when React re-runs the effect. "New research" starts a new
  session.
- The agent's text and tool steps render through the shared `ChatMessage`.
  Its `present_keywords` tool part renders as `KeywordReportView` instead:
  "how I read you" chips, KPI tiles, the keyword table with monthly
  sparklines, the seasonality chart, the searches-vs-difficulty matrix,
  quick wins from Search Console, the content plan, follow-up chips and Save
  all. Every number comes from the research the reply bought
  (`keywordReport.ts` validates it with Zod); a keyword without research
  shows dashes.
- The cost note shows the most one answer can charge, data and AI together
  (`MAX_REPLY_CREDITS`, 1,200), before anything is sent. The server enforces
  the same number (`KeywordsReplyMaxCredits`); a Go test fails if they differ.
- An old `?q=` link offers the seed as a one-click question; it never sends
  on its own.
- Saved keywords: filtered/sorted/paged list (page sizes 50/100/250), bulk
  remove (max 2000 ids), tag assign (append/replace), tag rename/recolor, and
  a delete that fails with `tag_in_use` while any keyword still carries the
  tag. Refresh re-buys metrics for every saved keyword.
- Export CSV/Sheets: runs the same list filters through `saved/export` and
  hands the rows to the shared exporter.

## Go endpoints

`POST /api/v1/projects/{projectId}/keywords/...` — `saved/refresh|save|list|export|remove`,
`saved/tags/assign|update|delete` (the page no longer calls `research` or
`serp` directly; the agent reaches them through its tools).
Contract: `backend/api/keywords.yaml`; response shapes in
`keywordsApi.ts` (Zod) mirror `backend/internal/keywords`.

## Rules that must never break

- Response validation through `apiRequest` + Zod: a drifted server shape is an
  "unexpected answer" error, never a half-rendered table.
- Never sneak a paid call past the user: nothing sends on page load, focus or
  re-render, the first question is sent once, and the cost ceiling is shown
  before the send.
- Playwright intercepts research HTTP requests and installs its own authenticated session/project fixtures; it never calls live DataForSEO.
- Customer runtime always uses the Go API; there is no build-time fixture bypass.
