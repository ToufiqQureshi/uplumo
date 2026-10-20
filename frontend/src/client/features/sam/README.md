# SAM assistant

Streams project-aware AI answers and tool actions with persisted sessions and credit controls.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `SamConversation.tsx`
- `samApi.ts`
- `useSamSessions.ts`
- `backend/internal/sam/agent.go`
- `backend/internal/sam/chat.go`
- `backend/internal/sam/tools.go`

Search Growth can prepare an editable draft in tab-local, project-scoped storage.
`analystDraft.ts` bounds it to 16,000 characters and 30 minutes; opening SAM never
sends it. Search analyst mode is explicit in the composer and passed to the Go
server, which restricts tool execution. The existing AI credit hint and Send
remain the paid-action boundary. Switching to general assistant is explicit.
