import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Sparkles, Search, BarChart3 } from "lucide-react";
import { IntelligenceMetrics } from "./IntelligenceMetrics";
import { IntelligenceOpportunities } from "./IntelligenceOpportunities";
import { Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/client/components/ui/card";
import { QueryError } from "@/client/components/QueryState";
import { saveAnalystDraft } from "@/client/features/sam/analystDraft";
import { growthAnalystPrompt } from "./growthBrief";
import { analyzeGrowth, type GrowthPage } from "./growthApi";
import { GrowthPageCard } from "./GrowthPageCard";
import { GrowthSettings, type GrowthSettingsValue } from "./GrowthSettings";
import { GrowthMonitor } from "./GrowthMonitor";
import { GrowthActions } from "./GrowthActions";
import { GrowthEvidencePanel } from "./GrowthEvidencePanel";

export function SearchGrowthPanel({ projectId }: { projectId: string }) {
  const [settings, setSettings] = useState<GrowthSettingsValue>({
    brands: "",
    goalEvent: "",
    days: 28,
    comparison: "previous",
    expanded: false,
  });
  const { brands, goalEvent, days, comparison, expanded } = settings;
  const [queryKind, setQueryKind] = useState("all");
  const [selectedPage, setSelectedPage] = useState("");
  const navigate = useNavigate();
  const end = new Date();
  end.setUTCDate(end.getUTCDate() - 4);
  const start = new Date(end);
  start.setUTCDate(start.getUTCDate() - days + 1);
  const input = {
    startDate: start.toISOString().slice(0, 10),
    endDate: end.toISOString().slice(0, 10),
    goalEvent,
    comparison,
    expanded,
    brandTerms: brands
      .split(",")
      .map((term) => term.trim())
      .filter(Boolean),
  };
  const analysis = useQuery({
    queryKey: ["search-intelligence", projectId, input],
    queryFn: () => analyzeGrowth(projectId, input),
    enabled: false,
    retry: false,
    staleTime: 10 * 60_000,
  });
  const report = analysis.data;
  const pages = report?.pages.filter((page) => page.findings.length > 0) ?? [];
  function openAnalyst(page?: GrowthPage, question?: string) {
    if (!report) return;
    try {
      saveAnalystDraft(
        projectId,
        growthAnalystPrompt(report, page) +
          (question ? `\n\nMy question: ${question}` : ""),
      );
    } catch {
      toast.error(
        "Your browser could not save the analyst draft. Allow tab storage and try again.",
      );
      return;
    }
    void navigate({ to: "/p/$projectId/sam", params: { projectId } });
  }
  return (
    <Card className="gap-0 overflow-hidden border-violet-500/15 bg-violet-500/[0.025]">
      <CardHeader className="border-b border-border bg-gradient-to-r from-violet-500/10 to-transparent py-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2 text-xl">
              <Sparkles className="size-5 text-violet-500" /> SEO Intelligence
              Hub
            </CardTitle>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Your Google visibility, visitor outcomes and page-specific
              evidence in one workspace.
            </p>
          </div>
          <Button
            disabled={analysis.isFetching}
            onClick={() => {
              setSelectedPage("");
              void analysis.refetch();
            }}
          >
            {analysis.isFetching
              ? "Analyzing…"
              : report
                ? "Refresh analysis"
                : "Find opportunities"}
          </Button>
        </div>
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <div className="flex min-w-0 items-center gap-3 rounded-lg border border-border bg-card p-3">
            <Search className="size-5 shrink-0 text-violet-500" />
            <div className="min-w-0 text-sm">
              <p className="font-medium">Google Search Console</p>
              <p className="break-all text-xs text-muted-foreground">
                {report
                  ? report.siteUrl
                  : "Connection checked when analysis runs"}
              </p>
            </div>
          </div>
          <div className="flex min-w-0 items-center gap-3 rounded-lg border border-border bg-card p-3">
            <BarChart3 className="size-5 shrink-0 text-blue-500" />
            <div className="min-w-0 text-sm">
              <p className="font-medium">Google Analytics 4</p>
              <p className="break-all text-xs text-muted-foreground">
                {report
                  ? `${report.propertyId} · Google organic sessions`
                  : "Use your connected GA4 property"}
              </p>
            </div>
          </div>
        </div>
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer font-medium">
            Choose goal and comparison
          </summary>
          <GrowthSettings
            projectId={projectId}
            settings={settings}
            onChange={setSettings}
          />
        </details>
      </CardHeader>
      <CardContent className="space-y-4 pt-4">
        {analysis.isError ? (
          <>
            <QueryError
              error={analysis.error}
              fallback="Could not analyze your Google data."
            />
            <Link
              className="text-sm underline"
              to="/p/$projectId/settings"
              params={{ projectId }}
            >
              Review Google connections in settings
            </Link>
          </>
        ) : null}
        {!report && !analysis.isError ? (
          <p className="text-sm text-muted-foreground">
            Run an analysis to find the pages losing search traffic, changes in
            organic outcome rates and performing pages with room to grow. Google
            data only; no AI credits used. Choose a registered success event for
            goal-specific insights.
          </p>
        ) : null}
        {report ? (
          <>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p className="text-xs text-muted-foreground">
                {report.range.startDate} – {report.range.endDate} vs{" "}
                {report.previousRange.startDate} –{" "}
                {report.previousRange.endDate} · {report.matchedPages}/
                {report.gscPageCount} Search Console pages matched ·{" "}
                {report.propertyId} · Goal:{" "}
                {report.goalEvent || "all key events"}
              </p>
              <Button variant="outline" size="sm" onClick={() => openAnalyst()}>
                Ask AI analyst
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              AI opens an editable draft in SAM. Review its credit hint and
              press Send to run it.
            </p>
            <p className="text-xs text-muted-foreground">
              Analysis saved {new Date(report.generatedAt).toLocaleString()}.
              Refresh analysis to check again.
            </p>
            {report.truncated ? (
              <p
                role="status"
                className="rounded-lg border border-border bg-card p-3 text-sm"
              >
                Source coverage is limited. Findings cover returned rows only;
                review Data coverage before treating this as a complete site
                analysis.
              </p>
            ) : null}
            <IntelligenceMetrics report={report} />
            <IntelligenceOpportunities
              key={report.id}
              report={report}
              selected={selectedPage || pages[0]?.page || ""}
              onSelect={(page) => {
                setSelectedPage(page);
                document
                  .getElementById("growth-evidence")
                  ?.scrollIntoView({ block: "start" });
              }}
              onAnalyst={(question) => openAnalyst(undefined, question)}
            />
            {(() => {
              const page =
                pages.find((item) => item.page === selectedPage) ?? pages[0];
              return page ? (
                <section id="growth-evidence" className="scroll-mt-4 space-y-3">
                  <h3 className="text-lg font-semibold">
                    Investigate this page
                  </h3>
                  <GrowthPageCard
                    key={`${report.id}:${page.page}`}
                    projectId={projectId}
                    report={report}
                    page={page}
                    queryKind={report.brandTerms.length ? queryKind : "all"}
                    onAnalyst={() => openAnalyst(page)}
                  />
                </section>
              ) : null;
            })()}
            <details className="rounded-lg border border-border p-3 text-sm">
              <summary className="cursor-pointer">
                Data coverage and query filters
              </summary>
              <div className="mt-3 space-y-2">
                <label className="flex items-center gap-2">
                  Query evidence
                  <select
                    className="rounded border border-border bg-background p-1"
                    value={queryKind}
                    onChange={(e) => setQueryKind(e.target.value)}
                  >
                    <option value="all">All queries</option>
                    <option value="branded">Branded</option>
                    <option value="nonbranded">Non-branded</option>
                  </select>
                </label>
                <p className="text-xs text-muted-foreground">
                  Brand terms classify only the returned query evidence, not GA4
                  outcomes or page totals. Refresh analysis after changing brand
                  terms.
                </p>
                <p className="text-xs text-muted-foreground">
                  Coverage bound: {report.rowLimit ?? 1000} rows per source.
                  Search Console can withhold queries even below this bound.
                </p>
                <a
                  className="text-xs underline"
                  href="https://support.google.com/webmasters/answer/12918484"
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  Large site? Review Google's optional BigQuery bulk export
                  (Google Cloud charges may apply).
                </a>
                {report.warnings.map((warning) => (
                  <p key={warning} className="text-xs text-muted-foreground">
                    {warning}
                  </p>
                ))}
                <p className="text-xs text-muted-foreground">
                  Generated {new Date(report.generatedAt).toLocaleString()}.
                  Matching keeps hostnames and case-sensitive paths. Ambiguous
                  Analytics URL variants are left unmatched.
                </p>
              </div>
            </details>
          </>
        ) : null}
        {report ? (
          <GrowthEvidencePanel
            key={report.id}
            projectId={projectId}
            report={report}
          />
        ) : null}
        <GrowthMonitor projectId={projectId} goalEvent={goalEvent} />
        <GrowthActions projectId={projectId} />
      </CardContent>
    </Card>
  );
}
