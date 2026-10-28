import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Bot, Check, CircleAlert, Loader2 } from "lucide-react";
import { useState } from "react";
import { cn } from "cn";
import { ConfirmDialog } from "@/client/components/ConfirmDialog";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { Button } from "@/client/components/ui/button";
import { Checkbox } from "@/client/components/ui/checkbox";
import { formatCredits } from "@/client/features/billing/credits/format";
import {
  billingUsageQueryKey,
  creditLevel,
  useBillingUsage,
} from "@/client/features/billing/usageApi";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { ApiError } from "@/client/lib/uplumoApi";
import { BILLING_ROUTE } from "@/shared/billing";
import { batchMaxCredits, defaultSelection } from "./agentSelection";
import {
  citeFixResultsKey,
  runCiteFixAgent,
  type AgentMode,
  type CiteFixReport,
} from "./citeFixApi";
import { scoreBand } from "./scoreBand";

type PageState = {
  status: "waiting" | "running" | "done" | "failed";
  note?: string;
};

/** Errors after which the next page cannot succeed either. */
const STOP_CODES = new Set([
  "insufficient_credits",
  "daily_credit_limit",
  "usage_paused",
  "plan_limits_exceeded",
  "plan_required",
  "rate_limited",
]);

const toneText = {
  success: "text-success",
  warning: "text-warning",
  destructive: "text-destructive",
} as const;

/**
 * The weakest pages with a checkbox each, and the paid agent that works on
 * the checked ones. The cost of the whole batch at its cap is shown, and
 * confirmed, before anything is bought; pages run one at a time and a failed
 * page is never retried by the browser.
 */
export function AgentPanel({
  projectId,
  report,
}: {
  projectId: string;
  report: CiteFixReport;
}) {
  const offer = report.agent;
  const client = useQueryClient();
  const { data: usage } = useBillingUsage();
  const [selected, setSelected] = useState(() =>
    defaultSelection(report.pages, offer),
  );
  const [mode, setMode] = useState<AgentMode>("quick");
  const [confirming, setConfirming] = useState(false);
  const [progress, setProgress] = useState<Record<string, PageState>>({});
  const [busy, setBusy] = useState(false);
  const titles = new Map(report.fixes.map((fix) => [fix.id, fix.title]));
  const run = useMutation({
    retry: false,
    mutationFn: (url: string) =>
      runCiteFixAgent(projectId, {
        url,
        mode,
        idempotencyKey: crypto.randomUUID(),
      }),
  });

  const maxCredits = batchMaxCredits(selected.length, mode, offer);
  const charged = usage !== undefined && creditLevel(usage) !== "not_charged";
  const short = charged && usage.balanceCredits < maxCredits;
  const blocked = !offer.available || !offer.planAllowed;
  const full = selected.length >= offer.maxPages;

  const toggle = (url: string, on: boolean) =>
    setSelected((current) =>
      on ? [...current, url] : current.filter((u) => u !== url),
    );

  async function start() {
    setConfirming(false);
    setBusy(true);
    const urls = [...selected];
    setProgress(
      Object.fromEntries(urls.map((u) => [u, { status: "waiting" }])),
    );
    for (const [index, url] of urls.entries()) {
      setProgress((p) => ({ ...p, [url]: { status: "running" } }));
      try {
        const result = await run.mutateAsync(url);
        setProgress((p) => ({
          ...p,
          [url]: {
            status: "done",
            note: result.cached
              ? "Same page as last time: free"
              : `${formatCredits(result.credits)} credits`,
          },
        }));
      } catch (error) {
        setProgress((p) => ({
          ...p,
          [url]: { status: "failed", note: getStandardErrorMessage(error) },
        }));
        if (error instanceof ApiError && STOP_CODES.has(error.code)) {
          const rest = urls.slice(index + 1);
          setProgress((p) => ({
            ...p,
            ...Object.fromEntries(
              rest.map((u) => [u, { status: "failed", note: "Not started" }]),
            ),
          }));
          break;
        }
      }
    }
    setBusy(false);
    void client.invalidateQueries({ queryKey: citeFixResultsKey(projectId) });
    void client.invalidateQueries({ queryKey: billingUsageQueryKey });
  }

  return (
    <section aria-labelledby="cite-fix-pages" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <h2 id="cite-fix-pages" className="font-serif text-2xl">
            Pages, weakest first
          </h2>
          <p className="text-sm text-muted-foreground">
            Pick pages for the AI agent. It reads each page, your Search Console
            queries and the AI answers that cite others, then writes the fixes
            and a new draft for the page.
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-3 rounded-xl border bg-card p-4">
        <div className="flex flex-wrap items-center gap-2">
          <Bot aria-hidden className="size-4 text-primary" />
          <span className="font-medium">AI agent</span>
          <div role="group" aria-label="Agent mode" className="flex gap-1">
            {(
              [
                [
                  "quick",
                  "Quick",
                  offer.quickMaxCredits,
                  "Fix list, answer, FAQ, schema",
                ],
                [
                  "deep",
                  "Deep",
                  offer.deepMaxCredits,
                  "Also rewrites the whole page",
                ],
              ] as const
            ).map(([value, label, credits, hint]) => (
              <Button
                key={value}
                size="sm"
                variant={mode === value ? "secondary" : "ghost"}
                aria-pressed={mode === value}
                title={hint}
                disabled={busy}
                onClick={() => setMode(value)}
              >
                {label}
                <span className="font-mono text-xs opacity-70">
                  ≤{credits}/page
                </span>
              </Button>
            ))}
          </div>
          <span className="ml-auto font-mono text-xs text-muted-foreground">
            {selected.length}/{offer.maxPages} pages · at most{" "}
            {formatCredits(maxCredits)} credits
            {charged ? ` · ${formatCredits(usage.balanceCredits)} left` : ""}
          </span>
          <Button
            disabled={blocked || busy || selected.length === 0 || short}
            onClick={() => setConfirming(true)}
          >
            {busy ? (
              <Loader2 aria-hidden className="animate-spin" />
            ) : (
              <Bot aria-hidden />
            )}
            {busy
              ? "Working…"
              : `Fix ${selected.length} page${selected.length === 1 ? "" : "s"}`}
          </Button>
        </div>
        {!offer.available ? (
          <p className="text-xs text-muted-foreground">
            The AI agent is not available on this server.
          </p>
        ) : !offer.planAllowed ? (
          <p className="text-xs text-muted-foreground">
            The AI agent is on every paid plan.{" "}
            <Link to={BILLING_ROUTE} className="underline underline-offset-4">
              See plans
            </Link>
          </p>
        ) : short ? (
          <p className="text-xs text-destructive">
            Not enough credits for {selected.length} pages at the most they can
            cost. Pick fewer pages or use Quick.{" "}
            <Link to={BILLING_ROUTE} className="underline underline-offset-4">
              Add credits
            </Link>
          </p>
        ) : (
          <p className="text-xs text-muted-foreground">
            You pay what the AI really costs, usually far below the cap. The
            same unchanged page in the same mode is free the second time.
          </p>
        )}
      </div>

      <div className="overflow-x-auto rounded-xl border bg-card">
        <table className="w-full text-sm">
          <thead className="text-left text-xs text-muted-foreground">
            <tr className="border-b">
              <th className="w-10 p-3">
                <span className="sr-only">Pick</span>
              </th>
              <th className="p-3 font-medium">Page</th>
              <th className="p-3 text-right font-medium">Score</th>
              <th className="p-3 font-medium">Missing</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {report.pages.map((page) => {
              const checked = selected.includes(page.url);
              const state = progress[page.url];
              return (
                <tr key={page.url} className="align-top">
                  <td className="p-3">
                    <Checkbox
                      aria-label={`Fix ${page.title || page.url} with the agent`}
                      checked={checked}
                      disabled={busy || (!checked && full)}
                      onCheckedChange={(on) => toggle(page.url, on)}
                    />
                  </td>
                  <td className="max-w-xs p-3">
                    <p className="truncate font-medium">
                      {page.title || page.url}
                    </p>
                    <SafeExternalLink
                      url={page.url}
                      label={page.url}
                      className="inline-flex max-w-full items-center gap-1 truncate text-xs text-muted-foreground hover:underline"
                    />
                    {state ? <ProgressNote state={state} /> : null}
                  </td>
                  <td
                    className={cn(
                      "p-3 text-right font-mono tabular-nums",
                      toneText[scoreBand(page.score).tone],
                    )}
                  >
                    {page.score}
                  </td>
                  <td className="p-3 text-xs text-muted-foreground">
                    {page.failed.length === 0
                      ? "Nothing"
                      : page.failed
                          .map((id) => titles.get(id) ?? id)
                          .join(" · ")}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {confirming ? (
        <ConfirmDialog
          title={`Fix ${selected.length} page${selected.length === 1 ? "" : "s"} with the AI agent?`}
          confirmLabel="Start"
          onClose={() => setConfirming(false)}
          onConfirm={() => void start()}
        >
          {mode === "deep" ? "Deep" : "Quick"} mode costs at most{" "}
          {formatCredits(maxCredits)} credits for these pages
          {charged ? ` (you have ${formatCredits(usage.balanceCredits)})` : ""}.
          You are charged the real cost, page by page, never more than that.
        </ConfirmDialog>
      ) : null}
    </section>
  );
}

function ProgressNote({ state }: { state: PageState }) {
  const icon =
    state.status === "running" ? (
      <Loader2 aria-hidden className="size-3 animate-spin" />
    ) : state.status === "done" ? (
      <Check aria-hidden className="size-3 text-success" />
    ) : state.status === "failed" ? (
      <CircleAlert aria-hidden className="size-3 text-destructive" />
    ) : null;
  const text =
    state.note ??
    (state.status === "running" ? "Agent is working…" : "Waiting");
  return (
    <p
      role="status"
      className={cn(
        "mt-1 flex items-center gap-1 text-xs",
        state.status === "failed"
          ? "text-destructive"
          : "text-muted-foreground",
      )}
    >
      {icon}
      {text}
    </p>
  );
}
