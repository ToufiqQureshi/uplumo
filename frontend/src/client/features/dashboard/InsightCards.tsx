import { useQuery } from "@tanstack/react-query";
import { sortBy } from "remeda";
import { Link } from "@tanstack/react-router";
import { CardShell } from "@/client/components/CardShell";
import { StatTile } from "@/client/components/StatTile";
import { Button } from "@/client/components/ui/button";
import {
  EmptyCardBody,
  formatDay,
  moreDetailsClass,
  StatGridSkeleton,
} from "@/client/features/dashboard/cardParts";
import {
  analyticsSummaryQueryOptions,
  rangeForDays,
} from "@/client/features/analytics/analyticsApi";
import {
  aiShareOfVisitors,
  aiSourceRows,
  formatShare,
} from "@/client/features/analytics/presentation";
import {
  trackingApi,
  trackingKey,
} from "@/client/features/ai-search/trackingApi";
import {
  getLatestRankResults,
  getRankTrackingConfigSummaries,
} from "@/client/features/rank-tracking/rankTrackingApi";

// Every card here reads data Uplumo already stored. None of these requests
// buys provider work, so opening the dashboard stays free.

const count = new Intl.NumberFormat("en-US");

function LoadError({ what }: { what: string }) {
  return (
    <p className="text-sm text-muted-foreground">
      Couldn&rsquo;t load {what}. Try again shortly.
    </p>
  );
}

/** Visitors and AI traffic from Uplumo Analytics, last 30 days. */
export function AnalyticsSummaryCard({ projectId }: { projectId: string }) {
  const range = rangeForDays(30, new Date());
  const query = useQuery(analyticsSummaryQueryOptions(projectId, range));
  const summary = query.data;
  const action = (
    <Link
      to="/p/$projectId/analytics"
      params={{ projectId }}
      search={{ days: 30 }}
      className={moreDetailsClass}
    >
      More details
    </Link>
  );

  if (query.isError && !summary) {
    return (
      <CardShell title="Visitors and AI traffic" action={action}>
        <LoadError what="your visitors" />
      </CardShell>
    );
  }
  if (!summary) {
    return (
      <CardShell title="Visitors and AI traffic" action={action}>
        <StatGridSkeleton />
      </CardShell>
    );
  }
  if (summary.pageviews === 0) {
    return (
      <CardShell title="Visitors and AI traffic">
        <EmptyCardBody
          message="No visits recorded yet. Add the Uplumo tracking code to your site to see who visits and which AI assistants send them."
          cta={
            <Button
              size="lg"
              nativeButton={false}
              render={
                <Link
                  to="/p/$projectId/analytics"
                  params={{ projectId }}
                  search={{ days: 30 }}
                />
              }
            >
              Get the tracking code
            </Button>
          }
        />
      </CardShell>
    );
  }

  const aiRows = aiSourceRows(summary.aiSources);
  const aiVisitors = aiRows.reduce((total, row) => total + row.visitors, 0);
  const topAi = aiRows.find((row) => row.visitors > 0);
  return (
    <CardShell
      title="Visitors and AI traffic"
      stamp="Uplumo Analytics · last 30 days"
      action={action}
    >
      <div className="grid grid-cols-2 gap-3">
        <StatTile label="Visitors" value={count.format(summary.visitors)} />
        <StatTile label="From AI" value={count.format(aiVisitors)} />
        <StatTile
          label="AI share"
          value={formatShare(aiShareOfVisitors(summary.channels))}
        />
        <StatTile
          label={summary.goalEventName ? "Goal completions" : "Goal not set"}
          value={
            summary.goalEventName
              ? count.format(summary.conversionSessions)
              : "—"
          }
        />
      </div>
      <p className="mt-4 text-sm text-muted-foreground">
        {topAi
          ? `${topAi.label} sends you the most AI visitors.`
          : "No AI assistant has sent a visitor yet. AI visibility below shows where you are mentioned."}
      </p>
    </CardShell>
  );
}

/** The latest saved AI visibility check: how often AI names and cites you. */
export function AiVisibilityCard({ projectId }: { projectId: string }) {
  const query = useQuery({
    queryKey: trackingKey(projectId),
    queryFn: () => trackingApi.state(projectId),
    staleTime: 30_000,
  });
  const state = query.data;
  const action = (
    <Link
      to="/p/$projectId/prompt-tracking"
      params={{ projectId }}
      className={moreDetailsClass}
    >
      More details
    </Link>
  );

  if (query.isError && !state) {
    return (
      <CardShell title="AI visibility" action={action}>
        <LoadError what="AI visibility" />
      </CardShell>
    );
  }
  if (!state) {
    return (
      <CardShell title="AI visibility" action={action}>
        <StatGridSkeleton />
      </CardShell>
    );
  }
  const latest = state.runs.find(
    (run) => run.status === "completed" && run.answered > 0,
  );
  if (!latest) {
    return (
      <CardShell title="AI visibility">
        <EmptyCardBody
          message="Save the questions your customers ask ChatGPT and Gemini, and see whether AI recommends you or a competitor."
          cta={
            <Button
              size="lg"
              nativeButton={false}
              render={
                <Link
                  to="/p/$projectId/prompt-tracking"
                  params={{ projectId }}
                />
              }
            >
              Set up AI tracking
            </Button>
          }
        />
      </CardShell>
    );
  }

  const mentionRate = latest.mentioned / latest.answered;
  const citationRate = latest.cited / latest.answered;
  const next =
    mentionRate === 0
      ? "AI did not name you in any answer. Write a page that answers these questions directly."
      : citationRate === 0
        ? "AI names you but links to other sites. Add a clear answer and FAQ on your own page so AI can cite it."
        : "AI names and cites you. Keep these pages fresh and track more questions.";
  return (
    <CardShell
      title="AI visibility"
      stamp={`Last check ${formatDay(latest.finishedAt ?? latest.createdAt)} · ${latest.answered} answers${
        state.nextCheckAt ? ` · next ${formatDay(state.nextCheckAt)}` : ""
      }`}
      action={action}
    >
      <div className="grid grid-cols-2 gap-3">
        <StatTile label="Mentioned" value={formatShare(mentionRate)} />
        <StatTile label="Cited" value={formatShare(citationRate)} />
      </div>
      <p className="mt-4 text-sm text-muted-foreground">{next}</p>
    </CardShell>
  );
}

/** Tracked keywords from the first active tracker: where you rank now. */
export function RankTrackingCard({ projectId }: { projectId: string }) {
  const summariesQuery = useQuery({
    queryKey: ["dashboardRankSummaries", projectId],
    queryFn: () => getRankTrackingConfigSummaries({ data: { projectId } }),
  });
  const config = summariesQuery.data?.find(
    (item) => item.isActive && item.keywordCount > 0,
  );
  const resultsQuery = useQuery({
    queryKey: ["dashboardRankResults", projectId, config?.id],
    queryFn: () =>
      getLatestRankResults({
        data: { projectId, configId: config!.id, comparePeriod: "7d" },
      }),
    enabled: Boolean(config),
  });
  const action = (
    <Link
      to="/p/$projectId/rank-tracking"
      params={{ projectId }}
      className={moreDetailsClass}
    >
      More details
    </Link>
  );

  if (summariesQuery.isError || resultsQuery.isError) {
    return (
      <CardShell title="Rank tracking" action={action}>
        <LoadError what="your rankings" />
      </CardShell>
    );
  }
  if (!summariesQuery.data || (config && !resultsQuery.data)) {
    return (
      <CardShell title="Rank tracking" action={action}>
        <StatGridSkeleton />
      </CardShell>
    );
  }
  if (!config || !resultsQuery.data) {
    return (
      <CardShell title="Rank tracking">
        <EmptyCardBody
          message="Track the keywords that bring you customers and see your Google position change every week."
          cta={
            <Button
              size="lg"
              nativeButton={false}
              render={
                <Link to="/p/$projectId/rank-tracking" params={{ projectId }} />
              }
            >
              Track keywords
            </Button>
          }
        />
      </CardShell>
    );
  }

  const device = config.devices === "mobile" ? "mobile" : "desktop";
  const rows = resultsQuery.data.rows.map((row) => ({
    keyword: row.keyword,
    position: row[device].position,
    previous: row[device].previousPosition,
  }));
  const ranked = rows.filter((row) => row.position !== null);
  const topTen = ranked.filter((row) => (row.position ?? 101) <= 10).length;
  const shown = sortBy(rows, (row) => row.position ?? 1000).slice(0, 5);
  return (
    <CardShell
      title="Rank tracking"
      stamp={`${config.domain} · ${device}${
        config.lastCheckedAt
          ? ` · checked ${formatDay(config.lastCheckedAt)}`
          : ""
      }`}
      action={action}
    >
      <div className="grid grid-cols-2 gap-3">
        <StatTile label="Tracked" value={count.format(rows.length)} />
        <StatTile label="In top 10" value={count.format(topTen)} />
      </div>
      <ul className="mt-4 space-y-2">
        {shown.map((row) => (
          <li
            key={row.keyword}
            className="flex items-center justify-between gap-2 text-sm"
          >
            <span className="truncate">{row.keyword}</span>
            <span className="shrink-0 tabular-nums text-muted-foreground">
              {row.position === null ? "not in results" : `#${row.position}`}
              {row.position !== null &&
              row.previous !== null &&
              row.previous !== row.position
                ? row.previous > row.position
                  ? ` ▲${row.previous - row.position}`
                  : ` ▼${row.position - row.previous}`
                : ""}
            </span>
          </li>
        ))}
      </ul>
    </CardShell>
  );
}
