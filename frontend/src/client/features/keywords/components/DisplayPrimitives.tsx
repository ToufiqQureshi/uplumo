import type { KeywordResearchRow } from "@/types/keywords";
import { Card } from "@/client/components/ui/card";
import { Separator } from "@/client/components/ui/separator";
import { ScoreBadge } from "@/client/components/table/ScoreBadge";
import { IntentBadge } from "./IntentBadge";
import { Area, AreaChart } from "recharts";
import {
  ChartGrid,
  ChartXAxis,
  ChartYAxis,
} from "@/client/components/ChartAxes";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/client/components/ui/chart";
import type { MonthlySearch } from "@/types/keywords";
import {
  formatNumber,
  formatCompactNumber,
  lastTwelveMonths,
  MONTH_SHORT_LABELS,
} from "../utils";

export type SortField =
  | "keyword"
  | "searchVolume"
  | "cpc"
  | "competition"
  | "keywordDifficulty";
export type SortDir = "asc" | "desc";

const trendChartConfig = {
  searchVolume: { label: "Search volume", color: "var(--color-primary)" },
} satisfies ChartConfig;

export function AreaTrendChart({ trend }: { trend: MonthlySearch[] }) {
  const last12 = lastTwelveMonths(trend);
  if (last12.length === 0) return null;

  const data = last12.map((m) => ({
    month: MONTH_SHORT_LABELS[m.month - 1],
    searchVolume: m.searchVolume,
  }));

  return (
    <ChartContainer
      config={trendChartConfig}
      className="h-[210px]"
      aria-label="Search trend chart"
    >
      <AreaChart
        data={data}
        margin={{ top: 8, right: 8, left: 0, bottom: 4 }}
        accessibilityLayer
      >
        <defs>
          <linearGradient id="trendGrad" x1="0" y1="0" x2="0" y2="1">
            <stop
              offset="0%"
              stopColor="var(--color-searchVolume)"
              stopOpacity="var(--trend-fill-start-opacity)"
            />
            <stop
              offset="100%"
              stopColor="var(--color-searchVolume)"
              stopOpacity="var(--trend-fill-end-opacity)"
            />
          </linearGradient>
        </defs>
        <ChartGrid />
        <ChartXAxis dataKey="month" minTickGap={5} />
        <ChartYAxis
          tickFormatter={(value: number | string) =>
            formatCompactNumber(Number(value))
          }
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Area
          type="monotone"
          dataKey="searchVolume"
          stroke="var(--color-searchVolume)"
          strokeWidth={2}
          fill="url(#trendGrad)"
          isAnimationActive={false}
          dot={{ r: 3, fill: "var(--color-searchVolume)", strokeWidth: 0 }}
          activeDot={{ r: 5, fill: "var(--color-searchVolume)" }}
        />
      </AreaChart>
    </ChartContainer>
  );
}

export function OverviewStats({ keyword }: { keyword: KeywordResearchRow }) {
  return (
    <Card className="shrink-0 flex-row items-center gap-4 px-4 py-2.5 min-h-[48px]">
      <div className="flex items-center gap-2 min-w-0 shrink-0">
        <span className="font-bold text-base truncate max-w-[240px] capitalize">
          {keyword.keyword}
        </span>
        <ScoreBadge value={keyword.keywordDifficulty} />
      </div>

      <Separator orientation="vertical" className="h-6" />

      <div className="flex items-center gap-4 text-sm flex-wrap min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-muted-foreground">Vol</span>
          <span className="font-semibold tabular-nums">
            {formatNumber(keyword.searchVolume)}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-muted-foreground">CPC</span>
          <span className="font-semibold tabular-nums">
            {keyword.cpc == null ? "-" : `$${keyword.cpc.toFixed(2)}`}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <span className="text-muted-foreground">Comp</span>
          <span className="font-semibold tabular-nums">
            {keyword.competition == null ? "-" : keyword.competition.toFixed(2)}
          </span>
        </div>
        <IntentBadge intent={keyword.intent} />
      </div>
    </Card>
  );
}
