import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useState } from "react";
import { cn } from "cn";
import { EmptyState } from "@/client/components/EmptyState";
import { PageHeader } from "@/client/components/PageHeader";
import { QueryError } from "@/client/components/QueryState";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { SkeletonCard } from "@/client/components/SkeletonPresets";
import { Button } from "@/client/components/ui/button";
import { Progress } from "@/client/components/ui/progress";
import {
  citeFixKey,
  getCiteFixReport,
  type CiteFix,
  type CiteFixReport,
} from "./citeFixApi";
import { AgentPanel } from "./AgentPanel";
import { AgentResults } from "./AgentResults";
import { ProofCard } from "./ProofCard";
import { AIMentionsSection, RivalComparisonsSection } from "./EvidenceSections";
import { GoogleAIOverviewSection, SEOIssuesSection } from "./CitationSections";
import { scoreBand } from "./scoreBand";
import { TopicsSection } from "./TopicsSection";

/** While a scan runs, check back this often; the report is free to read. */
const RUNNING_REFRESH_MS = 15_000;

const toneText = {
  success: "text-success",
  warning: "text-warning",
  destructive: "text-destructive",
} as const;

const impactStyle = {
  high: "border-destructive/40 text-destructive",
  medium: "border-warning/40 text-warning",
  low: "text-muted-foreground",
} as const;

export function CiteFixPage({ projectId }: { projectId: string }) {
  const report = useQuery({
    queryKey: citeFixKey(projectId),
    queryFn: () => getCiteFixReport(projectId),
    refetchInterval: (query) =>
      query.state.data?.running ? RUNNING_REFRESH_MS : false,
  });

  const scanLink = (label: string, variant: "default" | "outline") => (
    <Button
      variant={variant}
      nativeButton={false}
      render={<Link to="/p/$projectId/audit" params={{ projectId }} />}
    >
      {label}
    </Button>
  );

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="Cite Fix"
        description="Why AI assistants like ChatGPT and Perplexity don't quote your site, and what to fix first. Free: built from your latest site scan."
        actions={
          report.data?.state === "ready"
            ? scanLink("Scan again", "outline")
            : null
        }
      />

      {report.data?.running ? (
        <p
          role="status"
          className="flex items-center gap-2 rounded-lg border bg-muted/40 px-4 py-3 text-sm"
        >
          <Loader2 aria-hidden className="size-4 animate-spin" />A site scan is
          running. This page updates when it finishes.
        </p>
      ) : null}

      {report.isPending ? (
        <SkeletonCard />
      ) : report.isError ? (
        <QueryError
          error={report.error}
          fallback="Could not load Cite Fix."
          onRetry={() => void report.refetch()}
          isRetrying={report.isFetching}
        />
      ) : report.data.state === "no_scan" ? (
        <>
          <EmptyState
            title="Scan your site first"
            description="Cite Fix reads the pages a site scan collects. Run a free scan from Site Audit; your score appears here when it finishes."
            action={
              report.data.running ? null : scanLink("Scan my site", "default")
            }
          />
          <ProofCard projectId={projectId} proof={report.data.proof} />
          <RivalComparisonsSection
            proof={report.data.proof}
            comparisons={report.data.rivals}
          />
        </>
      ) : report.data.state === "needs_rescan" ? (
        <>
          <EmptyState
            title="Scan your site again"
            description="Your last scan did not collect the latest citation and topic signals. A new scan is free."
            action={
              report.data.running ? null : scanLink("Scan again", "default")
            }
          />
          <ProofCard projectId={projectId} proof={report.data.proof} />
          <RivalComparisonsSection
            proof={report.data.proof}
            comparisons={report.data.rivals}
          />
        </>
      ) : (
        <Ready projectId={projectId} report={report.data} />
      )}
      {report.data ? (
        <>
          <AIMentionsSection
            projectId={projectId}
            mentions={report.data.mentions}
          />
          <GoogleAIOverviewSection
            projectId={projectId}
            initial={report.data.googleAiOverview}
          />
          <SEOIssuesSection projectId={projectId} report={report.data} />
        </>
      ) : null}
    </div>
  );
}

function Ready({
  projectId,
  report,
}: {
  projectId: string;
  report: CiteFixReport;
}) {
  const score = report.score ?? 0;
  const band = scoreBand(score);
  return (
    <>
      <section
        aria-label="Cite Fix score"
        className="flex flex-col gap-5 rounded-xl border bg-card p-5"
      >
        <div className="flex flex-wrap items-end gap-x-6 gap-y-2">
          <p className="font-serif text-6xl leading-none tabular-nums">
            {score}
            <span className="text-2xl text-muted-foreground">/100</span>
          </p>
          <div className="flex flex-col gap-1">
            <p className={cn("font-medium", toneText[band.tone])}>
              {band.label}
            </p>
            {report.scan ? (
              <p className="font-mono text-xs text-muted-foreground">
                {report.pagesChecked} of {report.pagesTotal} pages checked ·
                scanned{" "}
                {new Date(report.scan.completedAt).toLocaleDateString("en-IN", {
                  day: "numeric",
                  month: "short",
                  year: "numeric",
                })}
              </p>
            ) : null}
          </div>
        </div>
        <dl className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
          {report.categories.map((category) => (
            <div key={category.id} className="flex flex-col gap-1.5">
              <div className="flex justify-between gap-2 text-sm">
                <dt>{category.label}</dt>
                <dd className="font-mono tabular-nums">{category.score}</dd>
              </div>
              <Progress
                value={category.score}
                aria-label={`${category.label}: ${category.score} of 100`}
              />
            </div>
          ))}
        </dl>
        <p className="text-xs text-muted-foreground">
          A higher score makes your pages easier for AI assistants to read and
          quote. No tool can promise a citation: the assistant decides.
        </p>
      </section>

      <ProofCard projectId={projectId} proof={report.proof} />
      {report.topics ? (
        <TopicsSection projectId={projectId} topics={report.topics} />
      ) : null}
      <RivalComparisonsSection
        proof={report.proof}
        comparisons={report.rivals}
      />

      <section aria-labelledby="cite-fix-fixes" className="flex flex-col gap-3">
        <h2 id="cite-fix-fixes" className="font-serif text-2xl">
          What to fix first
        </h2>
        {report.fixes.length === 0 ? (
          <EmptyState
            title="Nothing to fix"
            description="Every check passed on the pages we scanned. Scan again after you add or change pages."
          />
        ) : (
          <ol className="flex flex-col divide-y rounded-xl border bg-card">
            {report.fixes.map((fix, index) => (
              <FixRow key={fix.id} fix={fix} position={index + 1} />
            ))}
          </ol>
        )}
      </section>

      <AgentResults
        projectId={projectId}
        titles={new Map(report.pages.map((page) => [page.url, page.title]))}
      />

      {report.pages.length ? (
        <AgentPanel projectId={projectId} report={report} />
      ) : null}
    </>
  );
}

function FixRow({ fix, position }: { fix: CiteFix; position: number }) {
  const [open, setOpen] = useState(position === 1);
  const Chevron = open ? ChevronDown : ChevronRight;
  return (
    <li className="flex flex-col gap-3 p-4">
      <button
        type="button"
        className="flex items-start gap-2 text-left"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        <Chevron aria-hidden className="mt-0.5 size-4 shrink-0" />
        <span className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
          <span className="font-medium">
            {position}. {fix.title}
          </span>
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-xs",
              impactStyle[fix.impact],
            )}
          >
            {fix.impact} impact
          </span>
          <span className="font-mono text-xs text-primary">
            +{fix.gain} points
          </span>
          <span className="font-mono text-xs text-muted-foreground">
            {fix.pagesAffected
              ? `${fix.pagesAffected} page${fix.pagesAffected === 1 ? "" : "s"}`
              : "Whole site"}
          </span>
        </span>
      </button>
      {open ? (
        <div className="flex flex-col gap-3 pl-6 text-sm">
          <p>
            <span className="font-medium">Why it matters: </span>
            <span className="text-muted-foreground">{fix.why}</span>
          </p>
          <p>
            <span className="font-medium">How to fix: </span>
            <span className="text-muted-foreground">{fix.how}</span>
          </p>
          {fix.pages.length ? (
            <div className="flex flex-col gap-1">
              <p className="label-mono text-muted-foreground">
                {fix.pagesAffected > fix.pages.length
                  ? `First ${fix.pages.length} of ${fix.pagesAffected} pages`
                  : "Pages"}
              </p>
              <ul className="flex flex-col gap-1">
                {fix.pages.map((page) => (
                  <li key={page.url} className="min-w-0">
                    <SafeExternalLink
                      url={page.url}
                      label={page.title || page.url}
                      className="inline-flex max-w-full items-center gap-1 truncate hover:underline"
                    />
                    {page.evidence?.map((item) => (
                      <p
                        key={item}
                        className="break-words text-muted-foreground"
                      >
                        {item}
                      </p>
                    ))}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}
