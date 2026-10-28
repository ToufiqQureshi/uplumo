import { useChat } from "@ai-sdk/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { DefaultChatTransport, type UIMessage } from "ai";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import {
  ChatMessage,
  messageHasVisibleContent,
} from "@/client/components/chat/ChatMessage";
import { billingStatusQueryOptions } from "@/client/features/billing/plan/planApi";
import { ChatComposer } from "@/client/features/sam/ChatComposer";
import { createSamSession, samChatUrl } from "@/client/features/sam/samApi";
import { invalidateSamSessions } from "@/client/features/sam/samQueries";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { KeywordReportView } from "./KeywordReportView";
import { PRESENT_PART, reportFromPart } from "./keywordReport";

const EXAMPLES = [
  "Find 5 high-volume keywords my business can rank for",
  "Which keywords do I already rank for that I can push to page one?",
  "Give me easy long-tail keywords for a blog post",
  "Plan content around my main service keyword",
];

// The most keyword data one answer can buy: 3 auto researches of 150 rows
// (2 Labs tasks each), 2 SERP checks of 20 results and one 50-keyword metric
// lookup, at DataForSEO rates with the credit markup. backend/internal/sam/
// keywords_mode.go enforces the counts; flow.md has the maths.
const MAX_DATA_CREDITS = 265;

function messageText(message: UIMessage): string {
  return message.parts
    .filter((part) => part.type === "text")
    .map((part) => part.text)
    .join("\n");
}

export function KeywordAgentPage({
  projectId,
  initialPrompt,
}: {
  projectId: string;
  initialPrompt?: string;
}) {
  const status = useQuery(billingStatusQueryOptions);
  const queryClient = useQueryClient();
  const [session, setSession] = useState<{
    id: string;
    firstMessage: string;
  } | null>(null);
  const create = useMutation({
    mutationFn: () => createSamSession({ data: { projectId } }),
    onError: (error) => toast.error(getStandardErrorMessage(error)),
  });

  const start = (text: string) => {
    if (create.isPending) return;
    create.mutate(undefined, {
      onSuccess: ({ id }) => {
        void invalidateSamSessions(projectId);
        setSession({ id, firstMessage: text });
      },
    });
  };

  // The server refuses paid tools on Free too; this saves a Free workspace
  // from typing a question that cannot be answered.
  const isFree = status.data?.plan === "free" && !status.data.unlimited;

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-4 py-6 md:px-6">
      <PageHeader
        title="Keyword Research"
        description="Ask in plain words. The agent researches, picks the keywords worth your time and tells you what to do with them."
      />
      {isFree ? (
        <div className="rounded-xl border bg-card p-6">
          <h2 className="text-base font-semibold">
            Keyword research is part of every paid plan
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Upgrade to ask the keyword agent, see search volumes, difficulty and
            a content plan for your site.
          </p>
          <Button className="mt-4" render={<Link to="/billing/plan" />}>
            See plans
          </Button>
        </div>
      ) : session ? (
        <KeywordChat
          key={session.id}
          projectId={projectId}
          sessionId={session.id}
          firstMessage={session.firstMessage}
          onNew={() => {
            setSession(null);
            void queryClient.invalidateQueries(billingStatusQueryOptions);
          }}
        />
      ) : (
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap gap-2">
            {EXAMPLES.map((example) => (
              <Button
                key={example}
                variant="outline"
                size="sm"
                disabled={create.isPending}
                onClick={() => start(example)}
              >
                {example}
              </Button>
            ))}
          </div>
          <StartComposer
            projectId={projectId}
            initialPrompt={initialPrompt}
            busy={create.isPending}
            onSend={start}
          />
          <CostNote />
        </div>
      )}
    </div>
  );
}

function CostNote() {
  return (
    <p className="text-xs text-muted-foreground">
      Each answer buys at most 3 keyword searches and 2 Google result checks (up
      to {MAX_DATA_CREDITS} credits of data), plus AI usage charged as used.
      Questions about results you already have use no keyword data.
    </p>
  );
}

function StartComposer({
  projectId,
  initialPrompt,
  busy,
  onSend,
}: {
  projectId: string;
  initialPrompt?: string;
  busy: boolean;
  onSend: (text: string) => void;
}) {
  const [draft] = useState(() =>
    initialPrompt ? `Find keywords for "${initialPrompt}"` : "",
  );
  return (
    <div className="flex flex-col gap-2">
      {draft ? (
        <Button
          variant="secondary"
          size="sm"
          className="self-start"
          disabled={busy}
          onClick={() => onSend(draft)}
        >
          {draft}
        </Button>
      ) : null}
      <ChatComposer
        busy={busy}
        projectId={projectId}
        onSend={onSend}
        onStop={() => {}}
      />
    </div>
  );
}

function KeywordChat({
  projectId,
  sessionId,
  firstMessage,
  onNew,
}: {
  projectId: string;
  sessionId: string;
  firstMessage: string;
  onNew: () => void;
}) {
  const transport = useMemo(
    () =>
      new DefaultChatTransport({
        api: samChatUrl(projectId),
        prepareSendMessagesRequest: ({ messages }) => {
          const last = messages[messages.length - 1];
          return {
            body: {
              sessionId,
              mode: "keywords",
              message: { id: last.id, text: messageText(last) },
            },
          };
        },
      }),
    [projectId, sessionId],
  );
  const { messages, sendMessage, status, stop } = useChat({
    id: sessionId,
    transport,
    experimental_throttle: 50,
  });
  const busy = status === "submitted" || status === "streaming";

  // Send the first question once. Every send buys data, so a re-run effect
  // (Strict Mode, a re-render) must never send it twice.
  const sent = useRef(false);
  useEffect(() => {
    if (sent.current) return;
    sent.current = true;
    void sendMessage({ text: firstMessage });
  }, [firstMessage, sendMessage]);

  const send = (text: string) => {
    if (!busy) void sendMessage({ text });
  };
  const last = messages[messages.length - 1];
  const waiting =
    busy && (!last || last.role === "user" || !messageHasVisibleContent(last));

  return (
    <div className="flex flex-col gap-5">
      {messages.map((message) => {
        const reports = message.parts.flatMap((part) => {
          const report = reportFromPart(part);
          return report ? [report] : [];
        });
        const shown = {
          ...message,
          parts: message.parts.filter((part) => part.type !== PRESENT_PART),
        };
        return (
          <div key={message.id} className="flex flex-col gap-4">
            <ChatMessage message={shown} streaming={busy && message === last} />
            {reports.map((report, i) => (
              <KeywordReportView
                key={i}
                report={report}
                projectId={projectId}
                onFollowUp={send}
                busy={busy}
              />
            ))}
          </div>
        );
      })}
      {waiting ? (
        <p className="text-sm text-muted-foreground">Researching…</p>
      ) : null}
      {status === "error" ? (
        <p className="text-sm text-destructive">
          The answer stopped. You can ask again; data already bought is reused.
        </p>
      ) : null}
      <div className="sticky bottom-3 flex flex-col gap-2 rounded-2xl border bg-background p-3 shadow-sm">
        <ChatComposer
          busy={busy}
          projectId={projectId}
          onSend={send}
          onStop={() => void stop()}
        />
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CostNote />
          <Button variant="ghost" size="sm" disabled={busy} onClick={onNew}>
            New research
          </Button>
        </div>
      </div>
    </div>
  );
}
