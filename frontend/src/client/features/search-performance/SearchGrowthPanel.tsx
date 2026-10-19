import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
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
  const [showAll, setShowAll] = useState(false);
  const [pageNumber, setPageNumber] = useState(0);
  const navigate = useNavigate();
  const analysis = useMutation({
    mutationFn: () => {
      const end = new Date();
      end.setUTCDate(end.getUTCDate() - 4);
      const start = new Date(end);
      start.setUTCDate(start.getUTCDate() - days + 1);
      return analyzeGrowth(projectId, {
        startDate: start.toISOString().slice(0, 10),
        endDate: end.toISOString().slice(0, 10),
        goalEvent,
        comparison,
        expanded,
        brandTerms: brands
          .split(",")
          .map((x) => x.trim())
          .filter(Boolean),
      });
    },
    retry: false,
    onSuccess: () => {
      setShowAll(false);
      setPageNumber(0);
    },
  });
  const report = analysis.data;
  const pages = report?.pages.filter((page) => page.findings.length > 0) ?? [];
  function openAnalyst(page?: GrowthPage) {
    if (!report) return;
    try {
      saveAnalystDraft(projectId, growthAnalystPrompt(report, page));
    } catch {
      toast.error(
        "Your browser could not save the analyst draft. Allow tab storage and try again.",
      );
      return;
    }
    void navigate({ to: "/p/$projectId/sam", params: { projectId } });
  }
  return (
    <Card className="gap-0">
      <CardHeader className="border-b py-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle>Turn search traffic into useful outcomes</CardTitle>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Search Console shows how people find you. GA4 shows what happens
              after they arrive. Find your next action from both.
            </p>
          </div>
          <Button
            disabled={analysis.isPending}
            onClick={() => analysis.mutate()}
          >
            {analysis.isPending
              ? "Analyzing…"
              : report
                ? "Refresh analysis"
                : "Find opportunities"}
          </Button>
        </div>
        <GrowthSettings
          projectId={projectId}
          settings={settings}
          onChange={setSettings}
        />
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
            Connect GSC and GA4, then find opportunities. We will show evidence
            and possible explanations, with one next action per finding.
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
            {pages.length === 0 ? (
              <p className="rounded-lg border p-4 text-sm">
                No strong opportunities met the evidence thresholds in this
                report. This does not mean your site has no issues; review
                coverage and tracking before drawing conclusions.
              </p>
            ) : null}
            {(showAll
              ? pages.slice(pageNumber * 10, (pageNumber + 1) * 10)
              : pages.slice(0, 3)
            ).map((page) => (
              <GrowthPageCard
                key={`${report.id}:${page.page}`}
                projectId={projectId}
                report={report}
                page={page}
                queryKind={report.brandTerms.length ? queryKind : "all"}
                onAnalyst={() => openAnalyst(page)}
              />
            ))}
            {pages.length > 3 ? (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowAll(!showAll)}
              >
                {showAll
                  ? "Show top 3"
                  : `Browse ${pages.length} opportunities, 10 at a time`}
              </Button>
            ) : null}
            {showAll ? (
              <div className="flex flex-wrap items-center gap-2 text-sm">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={pageNumber === 0}
                  onClick={() => setPageNumber((n) => n - 1)}
                >
                  Previous opportunities
                </Button>
                <span>
                  Page {pageNumber + 1} of {Math.ceil(pages.length / 10)}
                </span>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={(pageNumber + 1) * 10 >= pages.length}
                  onClick={() => setPageNumber((n) => n + 1)}
                >
                  Next opportunities
                </Button>
              </div>
            ) : null}
            <details className="rounded-lg border p-3 text-sm">
              <summary className="cursor-pointer">
                Data coverage and query filters
              </summary>
              <div className="mt-3 space-y-2">
                <label className="flex items-center gap-2">
                  Query evidence
                  <select
                    className="rounded border bg-background p-1"
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
