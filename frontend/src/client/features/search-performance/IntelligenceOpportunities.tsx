import { useState } from "react";
import { ArrowUpRight, SearchCheck, Sparkles } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Badge } from "@/client/components/ui/badge";
import type { GrowthPage, GrowthReport } from "./growthApi";
import { growthCount, growthRate } from "./growthFormat";

export function IntelligenceOpportunities({
  report,
  selected,
  onSelect,
  onAnalyst,
}: {
  report: GrowthReport;
  selected: string;
  onSelect: (page: string) => void;
  onAnalyst: (question: string) => void;
}) {
  const [pageIndex, setPageIndex] = useState(0);
  const [customQuestion, setQuestion] = useState("");
  const pages = report.pages.filter((page) => page.findings.length > 0);
  return (
    <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_300px]">
      <section className="min-w-0 overflow-hidden rounded-xl border border-border bg-card">
        <div className="flex items-center gap-2 border-b border-border p-5">
          <SearchCheck className="size-5 text-violet-500" />
          <div>
            <h3 className="font-semibold">Your next best actions</h3>
            <p className="text-xs text-muted-foreground">
              Measured findings for specific pages. Open one to investigate and
              save a baseline.
            </p>
          </div>
        </div>
        {pages.length === 0 ? (
          <p className="p-5 text-sm text-muted-foreground">
            No findings passed the evidence thresholds. Check coverage and
            tracking before drawing conclusions.
          </p>
        ) : (
          <div className="max-h-[34rem] overflow-auto divide-y divide-border">
            {pages.slice(pageIndex * 10, pageIndex * 10 + 10).map((page) => (
              <Opportunity
                key={page.page}
                page={page}
                selected={selected === page.page}
                onSelect={onSelect}
                limited={report.limited}
                goalEvent={report.goalEvent}
              />
            ))}
          </div>
        )}
        {pages.length > 10 ? (
          <div className="flex items-center justify-between gap-2 border-t border-border p-3">
            <Button
              size="sm"
              variant="outline"
              disabled={pageIndex === 0}
              onClick={() => setPageIndex((value) => value - 1)}
            >
              Previous findings
            </Button>
            <span className="text-xs">
              {pageIndex + 1} / {Math.ceil(pages.length / 10)}
            </span>
            <Button
              size="sm"
              variant="outline"
              disabled={(pageIndex + 1) * 10 >= pages.length}
              onClick={() => setPageIndex((value) => value + 1)}
            >
              Next findings
            </Button>
          </div>
        ) : null}
      </section>
      <aside className="space-y-4 rounded-xl border border-violet-500/20 bg-violet-500/5 p-5">
        <div className="flex items-center gap-2">
          <Sparkles className="size-5 text-violet-500" />
          <h3 className="font-semibold">Your SEO analyst</h3>
        </div>
        <p className="text-sm text-muted-foreground">
          Ask about this report. Your dates, selected goal and page context go
          into an editable SAM draft.
        </p>
        {[
          "Which pages account for the biggest click losses?",
          "Where did organic visitors stop reaching my selected goal?",
          "Which performing pages could benefit from more search reach?",
          "Could tracking or device mix explain the changes?",
        ].map((question) => (
          <Button
            key={question}
            variant="outline"
            className="h-auto w-full justify-between whitespace-normal py-3 text-left"
            onClick={() => onAnalyst(question)}
          >
            {question}
            <ArrowUpRight className="size-4 shrink-0" />
          </Button>
        ))}
        <form
          className="space-y-2"
          onSubmit={(event) => {
            event.preventDefault();
            if (customQuestion.trim()) onAnalyst(customQuestion.trim());
          }}
        >
          <label className="grid gap-2 text-xs font-medium">
            Ask about this report
            <Input
              value={customQuestion}
              onChange={(event) => setQuestion(event.target.value)}
              maxLength={1000}
              placeholder="What changed on my booking pages?"
            />
          </label>
          <Button
            type="submit"
            className="w-full"
            disabled={!customQuestion.trim()}
          >
            Prepare analyst question
          </Button>
        </form>
        <p className="text-xs text-muted-foreground">
          Opening a draft uses no AI credits. Review the credit hint and press
          Send in SAM to run it. Findings are evidence for investigation, not
          proven causes.
        </p>
      </aside>
    </div>
  );
}

function Opportunity({
  page,
  selected,
  onSelect,
  limited,
  goalEvent,
}: {
  limited: boolean;
  goalEvent?: string;
  page: GrowthPage;
  selected: boolean;
  onSelect: (page: string) => void;
}) {
  const finding = page.findings[0];
  if (!finding) return null;
  const lost = page.previous
    ? page.previous.clicks - page.current.clicks
    : null;
  return (
    <article className={`space-y-3 p-5 ${selected ? "bg-violet-500/5" : ""}`}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <p className="font-semibold text-foreground">{finding.title}</p>
        <Badge variant="secondary">{page.findings.length} findings</Badge>
      </div>
      <p className="break-all text-xs text-muted-foreground">{page.page}</p>

      {/* Measured Evidence & Action Callout */}
      <div className="rounded-lg border border-border/80 bg-background/60 p-3 space-y-1.5 text-xs">
        <p className="text-muted-foreground leading-relaxed">
          <strong className="text-foreground font-medium">
            Observed Evidence:
          </strong>{" "}
          {finding.evidence}
        </p>
        <p className="text-violet-700 dark:text-violet-300 font-medium leading-relaxed">
          💡 <strong>Recommended Decision:</strong> {finding.action}
        </p>
      </div>

      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span>
          {growthCount(page.current.clicks)} clicks
          {lost !== null && lost > 0 ? ` · ${growthCount(lost)} fewer` : ""}
        </span>
        <span>{growthCount(page.current.sessions)} organic sessions</span>
        <span>
          {growthRate(
            limited
              ? null
              : goalEvent
                ? (page.current.goalRate ?? null)
                : page.current.keyEventRate,
          )}{" "}
          session outcome rate
        </span>
      </div>

      <Button
        variant={selected ? "secondary" : "outline"}
        size="sm"
        onClick={() => onSelect(page.page)}
      >
        View evidence & action plan <ArrowUpRight />
      </Button>
    </article>
  );
}
