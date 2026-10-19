import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
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
import { GrowthActions } from "./GrowthActions";
import { GrowthEvidencePanel } from "./GrowthEvidencePanel";

export function SearchGrowthPanel({ projectId }: { projectId: string }) {
  const [brands, setBrands] = useState("");
  const [days, setDays] = useState(28);
  const [queryKind, setQueryKind] = useState("all");
  const [showAll, setShowAll] = useState(false);
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
        brandTerms: brands
          .split(",")
          .map((x) => x.trim())
          .filter(Boolean),
      });
    },
    retry: false,
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
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer text-muted-foreground">
            Analysis settings
          </summary>
          <div className="mt-3 flex flex-wrap items-end gap-3">
            <label className="grid gap-1">
              Brand names, separated by commas
              <Input
                value={brands}
                maxLength={1000}
                placeholder="Your brand, alternate spelling"
                onChange={(e) => setBrands(e.target.value)}
              />
            </label>
            <label className="grid gap-1">
              Comparison window
              <select
                className="rounded-md border bg-background p-2"
                value={days}
                onChange={(e) => setDays(Number(e.target.value))}
              >
                <option value={14}>14 days</option>
                <option value={28}>28 days</option>
              </select>
            </label>
            <p className="text-xs text-muted-foreground">
              Google organic traffic only. Independent of the report filters
              below. No AI credits used.
            </p>
          </div>
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
                {report.propertyId}
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
            {(showAll ? pages : pages.slice(0, 3)).map((page) => (
              <GrowthPageCard
                key={page.page}
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
                  : `Show all ${pages.length} opportunities`}
              </Button>
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
        <GrowthActions projectId={projectId} />
      </CardContent>
    </Card>
  );
}
