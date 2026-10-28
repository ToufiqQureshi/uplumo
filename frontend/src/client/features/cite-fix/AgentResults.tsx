import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ChevronDown, ChevronRight, PenLine } from "lucide-react";
import { useState } from "react";
import { CopyButton } from "@/client/components/CopyButton";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { Button } from "@/client/components/ui/button";
import { formatCredits } from "@/client/features/billing/credits/format";
import { saveContentDraft } from "@/client/features/content-editor/contentEditorApi";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  citeFixResultsKey,
  getCiteFixAgentResults,
  type AgentResult,
} from "./citeFixApi";

/** The agent's saved work, newest page first. Reading it is free. */
export function AgentResults({
  projectId,
  titles,
}: {
  projectId: string;
  /** Page titles from the scan, by URL. */
  titles: Map<string, string>;
}) {
  const results = useQuery({
    queryKey: citeFixResultsKey(projectId),
    queryFn: () => getCiteFixAgentResults(projectId),
  });
  const rows = [...(results.data?.results ?? [])];
  // A fresh array, so sorting in place is safe: newest first.
  rows.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  if (rows.length === 0) return null;
  return (
    <section aria-labelledby="cite-fix-results" className="flex flex-col gap-3">
      <h2 id="cite-fix-results" className="font-serif text-2xl">
        Agent results
      </h2>
      <ol className="flex flex-col divide-y rounded-xl border bg-card">
        {rows.map((result, index) => (
          <ResultRow
            key={`${result.url}:${result.mode}`}
            projectId={projectId}
            result={result}
            title={titles.get(result.url) ?? ""}
            initiallyOpen={index === 0}
          />
        ))}
      </ol>
    </section>
  );
}

function ResultRow({
  projectId,
  result,
  title,
  initiallyOpen,
}: {
  projectId: string;
  result: AgentResult;
  title: string;
  initiallyOpen: boolean;
}) {
  const [open, setOpen] = useState(initiallyOpen);
  const navigate = useNavigate();
  const Chevron = open ? ChevronDown : ChevronRight;
  const openInEditor = useMutation({
    retry: false,
    mutationFn: () =>
      saveContentDraft({
        data: {
          projectId,
          title: (result.titleTag || title || result.url).slice(0, 180),
          targetKeyword: result.targetQuestion.slice(0, 120),
          slug: slugOf(result.url),
          metaDescription: result.metaDescription.slice(0, 320),
          content: result.draft,
        },
      }),
    onSuccess: (draft) =>
      void navigate({
        to: "/p/$projectId/content-editor",
        params: { projectId },
        search: { draft: draft.id },
      }),
  });

  return (
    <li className="flex flex-col gap-3 p-4">
      <button
        type="button"
        className="flex items-start gap-2 text-left"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        <Chevron aria-hidden className="mt-0.5 size-4 shrink-0" />
        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span className="truncate font-medium">{title || result.url}</span>
            <span className="rounded-full border px-2 py-0.5 text-xs">
              {result.mode === "deep" ? "Deep" : "Quick"}
            </span>
            <span className="font-mono text-xs text-muted-foreground">
              {result.cached
                ? "free (unchanged page)"
                : `${formatCredits(result.credits)} credits`}
            </span>
          </span>
          <span className="text-sm text-muted-foreground">
            {result.summary}
          </span>
        </span>
      </button>
      {open ? (
        <div className="flex flex-col gap-4 pl-6 text-sm">
          <SafeExternalLink
            url={result.url}
            label={result.url}
            className="inline-flex max-w-full items-center gap-1 truncate text-xs text-muted-foreground hover:underline"
          />
          {result.partial ? (
            <p className="text-xs text-warning">
              The full rewrite did not finish, so you got the Quick result and
              paid only for it.
            </p>
          ) : null}
          {result.targetQuestion ? (
            <p>
              <span className="font-medium">Question to answer: </span>
              {result.targetQuestion}
            </p>
          ) : null}
          {result.recommendations.length ? (
            <div className="flex flex-col gap-1">
              <p className="label-mono text-muted-foreground">Fixes</p>
              <ol className="flex list-decimal flex-col gap-2 pl-5">
                {result.recommendations.map((rec) => (
                  <li key={rec.title}>
                    <span className="font-medium">{rec.title}</span>
                    {rec.detail ? (
                      <span className="text-muted-foreground">
                        {" "}
                        {rec.detail}
                      </span>
                    ) : null}
                  </li>
                ))}
              </ol>
            </div>
          ) : null}
          {result.answerFirst ? (
            <Block label="New opening paragraph" copy={result.answerFirst}>
              <p>{result.answerFirst}</p>
            </Block>
          ) : null}
          {result.faq.length ? (
            <Block
              label="FAQ"
              copy={result.faq
                .map((qa) => `${qa.question}\n${qa.answer}`)
                .join("\n\n")}
            >
              <dl className="flex flex-col gap-2">
                {result.faq.map((qa) => (
                  <div key={qa.question}>
                    <dt className="font-medium">{qa.question}</dt>
                    <dd className="text-muted-foreground">{qa.answer}</dd>
                  </div>
                ))}
              </dl>
            </Block>
          ) : null}
          {result.titleTag || result.metaDescription ? (
            <Block
              label="Title and description"
              copy={`${result.titleTag}\n${result.metaDescription}`}
            >
              <p className="font-medium">{result.titleTag}</p>
              <p className="text-muted-foreground">{result.metaDescription}</p>
            </Block>
          ) : null}
          {result.schemaJsonLd ? (
            <Block label="Schema (JSON-LD)" copy={result.schemaJsonLd}>
              <pre className="max-h-48 overflow-auto rounded-md bg-muted p-2 font-mono text-xs">
                {result.schemaJsonLd}
              </pre>
            </Block>
          ) : null}
          {result.verify.length ? (
            <div className="flex flex-col gap-1">
              <p className="label-mono text-warning">Check before publishing</p>
              <ul className="flex list-disc flex-col gap-1 pl-5 text-muted-foreground">
                {result.verify.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}
          <div className="flex flex-wrap items-center gap-2">
            <Button
              onClick={() => openInEditor.mutate()}
              disabled={openInEditor.isPending}
            >
              <PenLine aria-hidden />
              {result.mode === "deep"
                ? "Open the rewrite in Content Editor"
                : "Open in Content Editor"}
            </Button>
            <CopyButton
              value={result.draft}
              successMessage="Draft copied"
              label="Copy draft"
              size="sm"
            />
          </div>
          {openInEditor.isError ? (
            <p role="alert" className="text-xs text-destructive">
              {getStandardErrorMessage(openInEditor.error)}
            </p>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}

function Block({
  label,
  copy,
  children,
}: {
  label: string;
  copy: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border border-dashed p-3">
      <div className="flex items-center justify-between gap-2">
        <p className="label-mono text-muted-foreground">{label}</p>
        <CopyButton value={copy} successMessage="Copied" />
      </div>
      {children}
    </div>
  );
}

/** The page's last path segment, as the editor's slug. */
function slugOf(url: string): string {
  try {
    const segments = new URL(url).pathname.split("/").filter(Boolean);
    return (segments.at(-1) ?? "").slice(0, 300);
  } catch {
    return "";
  }
}
