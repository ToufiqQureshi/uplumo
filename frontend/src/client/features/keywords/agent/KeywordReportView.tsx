import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge } from "@/client/components/ui/badge";
import { Button } from "@/client/components/ui/button";
import { saveKeywords } from "@/client/features/keywords/keywordsApi";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  Card,
  DifficultyMatrix,
  Seasonality,
  Sparkline,
} from "./KeywordCharts";
import {
  difficultyLabel,
  reportTotals,
  type KeywordReport,
  type ReportKeyword,
} from "./keywordReport";

const CHANCE_STYLE: Record<ReportKeyword["chance"], string> = {
  high: "bg-lime-200 text-lime-950",
  medium: "bg-amber-100 text-amber-950",
  low: "bg-rose-100 text-rose-950",
};

export function KeywordReportView({
  report,
  projectId,
  onFollowUp,
  busy,
}: {
  report: KeywordReport;
  projectId: string;
  onFollowUp: (text: string) => void;
  busy: boolean;
}) {
  const totals = reportTotals(report.keywords);
  const withMetrics = report.keywords.filter((k) => k.metrics);
  const save = useMutation({
    mutationFn: () =>
      saveKeywords({
        projectId,
        keywords: withMetrics.map((k) => k.keyword),
        metrics: withMetrics.map((k) => ({
          keyword: k.keyword,
          searchVolume: k.metrics?.searchVolume ?? null,
          cpc: k.metrics?.cpc ?? null,
          keywordDifficulty: k.metrics?.keywordDifficulty ?? null,
          intent: k.metrics?.intent ?? null,
        })),
      }),
    onSuccess: () => toast.success(`Saved ${withMetrics.length} keywords`),
    onError: (error) => toast.error(getStandardErrorMessage(error)),
  });
  const ranked = report.quickWins?.length ?? 0;

  return (
    <div className="flex flex-col gap-4">
      {report.understood?.length ? (
        <div
          className="flex flex-wrap gap-2"
          aria-label="How the agent read your request"
        >
          {report.understood.map((u) => (
            <span
              key={u.label}
              className="rounded-full border bg-muted/50 px-3 py-1 text-xs"
            >
              <span className="text-muted-foreground">{u.label}:</span>{" "}
              {u.value}
            </span>
          ))}
        </div>
      ) : null}

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          {
            label: "Total searches / month",
            value: formatNumber(totals.volume),
            note: `${report.keywords.length} keywords`,
          },
          {
            label: "Average difficulty",
            value:
              totals.averageDifficulty == null
                ? "—"
                : String(totals.averageDifficulty),
            note:
              totals.averageDifficulty == null
                ? ""
                : `out of 100, ${difficultyLabel(totals.averageDifficulty)}`,
          },
          {
            label: "You already rank",
            value: ranked ? `${ranked}` : "—",
            note: ranked
              ? "pages to improve first"
              : "connect Search Console to see",
          },
          {
            label: "Traffic if you reach top 3",
            value: `~${formatNumber(totals.top3Visits)}`,
            note: "visits/month, estimate",
          },
        ].map((kpi) => (
          <div key={kpi.label} className="rounded-xl border bg-card p-3">
            <div className="text-xs text-muted-foreground">{kpi.label}</div>
            <div className="mt-1 text-2xl font-bold tabular-nums">
              {kpi.value}
            </div>
            <div className="text-xs text-muted-foreground">{kpi.note}</div>
          </div>
        ))}
      </div>

      <section className="overflow-hidden rounded-xl border bg-card">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
          <h3 className="text-sm font-semibold">Keywords</h3>
          <Button
            size="sm"
            disabled={
              save.isPending || save.isSuccess || withMetrics.length === 0
            }
            onClick={() => save.mutate()}
          >
            {save.isSuccess ? "Saved" : `Save all ${withMetrics.length}`}
          </Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] text-sm">
            <thead>
              <tr className="text-left text-xs text-muted-foreground">
                <th className="px-4 py-2 font-medium">Keyword</th>
                <th className="px-2 py-2 font-medium">Searches/mo</th>
                <th className="px-2 py-2 font-medium">Trend</th>
                <th className="px-2 py-2 font-medium">Difficulty</th>
                <th className="px-2 py-2 font-medium">CPC</th>
                <th className="px-2 py-2 font-medium">Intent</th>
                <th className="px-4 py-2 font-medium">Chance</th>
              </tr>
            </thead>
            <tbody>
              {report.keywords.map((k) => (
                <tr key={k.keyword} className="border-t align-top">
                  <td className="px-4 py-2.5">
                    <div className="font-medium">{k.keyword}</div>
                    {k.note ? (
                      <div className="text-xs text-muted-foreground">
                        {k.note}
                      </div>
                    ) : null}
                  </td>
                  <td className="px-2 py-2.5 tabular-nums">
                    {formatNumber(k.metrics?.searchVolume)}
                  </td>
                  <td className="px-2 py-2.5">
                    <Sparkline
                      values={k.metrics?.monthlyVolumes ?? []}
                      label={`Monthly searches for ${k.keyword}`}
                    />
                  </td>
                  <td className="px-2 py-2.5 tabular-nums">
                    {k.metrics?.keywordDifficulty == null ? (
                      "—"
                    ) : (
                      <>
                        {k.metrics.keywordDifficulty}{" "}
                        <span className="text-xs text-muted-foreground">
                          {difficultyLabel(k.metrics.keywordDifficulty)}
                        </span>
                      </>
                    )}
                  </td>
                  <td className="px-2 py-2.5 tabular-nums">
                    {k.metrics?.cpc == null
                      ? "—"
                      : `$${k.metrics.cpc.toFixed(2)}`}
                  </td>
                  <td className="px-2 py-2.5 capitalize">
                    {k.metrics?.intent ?? "—"}
                  </td>
                  <td className="px-4 py-2.5">
                    <span
                      className={`rounded-full px-2.5 py-0.5 text-xs font-semibold capitalize ${CHANCE_STYLE[k.chance]}`}
                    >
                      {k.chance}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="border-t px-4 py-2 text-xs text-muted-foreground">
          Keyword data from DataForSEO. Chance and traffic are estimates, not a
          guarantee of ranking.
        </p>
      </section>

      <div className="grid gap-3 md:grid-cols-2">
        {report.keywords[0] ? (
          <Seasonality keyword={report.keywords[0]} />
        ) : null}
        <DifficultyMatrix keywords={report.keywords} />
      </div>

      {report.quickWins?.length ? (
        <Card
          title="You already rank for these. Improve the page you have."
          hint="From your Search Console. Moving up is possible, not guaranteed."
        >
          <div className="grid gap-3 md:grid-cols-2">
            {report.quickWins.map((w) => (
              <div
                key={`${w.page}-${w.keyword}`}
                className="flex flex-col gap-2 rounded-lg border p-3"
              >
                <div className="break-all font-mono text-xs text-muted-foreground">
                  {w.page}
                </div>
                <div className="font-semibold">“{w.keyword}”</div>
                <div className="text-sm">
                  {w.position == null ? "" : `Now #${Math.round(w.position)}`}
                  {w.target ? ` → ${w.target} possible` : ""}
                </div>
                <ol className="list-decimal space-y-1 pl-5 text-sm">
                  {w.actions.map((a) => (
                    <li key={a}>{a}</li>
                  ))}
                </ol>
              </div>
            ))}
          </div>
        </Card>
      ) : null}

      {report.contentIdeas?.length ? (
        <Card title="Content plan: how to use these keywords">
          <div className="grid gap-3 md:grid-cols-2">
            {report.contentIdeas.map((idea) => (
              <div
                key={idea.title}
                className="flex flex-col gap-2 rounded-lg border p-3"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="secondary">{idea.type}</Badge>
                  <span className="text-xs text-muted-foreground">
                    targets: {idea.keywords.join(", ")}
                  </span>
                </div>
                <div className="font-semibold leading-snug">{idea.title}</div>
                <ul className="list-disc space-y-1 pl-5 text-sm">
                  {idea.outline.map((o) => (
                    <li key={o}>{o}</li>
                  ))}
                </ul>
                {idea.why ? (
                  <p className="text-xs text-muted-foreground">{idea.why}</p>
                ) : null}
              </div>
            ))}
          </div>
        </Card>
      ) : null}

      {report.followUps?.length ? (
        <div className="flex flex-wrap gap-2">
          {report.followUps.map((f) => (
            <Button
              key={f}
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => onFollowUp(f)}
            >
              {f}
            </Button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
