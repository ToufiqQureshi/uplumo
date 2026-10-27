# Insights: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when behavior or invariants change.

## Every page ends in an action, backed by numbers

- **Decision:** Next Actions, AI Opportunities, AI Sentiment and AI Prompt Ideas each show one insight sentence, the numbers behind it with their source, and AI buttons that do the work on the page. Copy is plain English (owner, 27 Oct 2026).
- **Why:** `FOUNDER.md` asks for "why, and what to fix" instead of dashboards; the owner rejected screens that only list data and asked for AI on every page, charged in credits.
- **Preserve:** Page loads never call a paid provider. Estimates are labelled "≈" and come from the customer's own data (impressions, positions); with no data the page says what is missing instead of guessing.
- **Trace:** `NextActionsPage.tsx`, `AiOpportunitiesPage.tsx`, `AiSentimentPage.tsx`, `PromptIdeasPage.tsx`; `insights.test.ts`.

## Reuse the writer and the existing "why not cited", do not add an AI system

- **Decision:** AI buttons call the content writer (`content/assistant/generate`) with five new insight actions (`review_replies`, `outreach_email`, `business_post`, `action_plan`, `prompt_ideas`) or existing ones (`title_meta`, `faq`, `outline`). AI Opportunities reuses `ai-search/whyNotCited.ts`.
- **Why:** The writer already holds credits before the call, settles actual cost, caps one action at 15 credits and never retries; a second AI path would duplicate that money logic.
- **Preserve:** `retry: false` on every paid mutation; a fresh idempotency key per click; the 15-credit cap shown before the click (`MAX_AI_ACTION_CREDITS`).
- **Trace:** `AiAction.tsx`; backend `contenteditor/writer.go`, `TestInsightActionsNeedTheirEvidence`, `TestInsightActionsUseTheirOwnPromptAndAreMetered`.

## Done/ignored is shared by the team, "to do" is no row

- **Decision:** Next Actions keys are derived from the source (`audit:<issueType>`, `gsc:q:<hash>`, `ai:not_named`) and stored per project, not per member.
- **Why:** A team works one list; a fix one member marks done must not reappear for the others.
- **Preserve:** Keys match `^[a-z][a-z0-9_:.-]{0,119}$`; at most 500 decisions per project; the project's organization is checked on every read and write.
- **Trace:** `nextActions.ts`, `insightsApi.ts`; backend `dashboard/actions.go`, `TestNextActionsAreSharedByTheTeamAndScopedToTheProjectOrganization`.
