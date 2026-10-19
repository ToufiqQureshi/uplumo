import { useState } from "react";
import { Eye, MousePointerClick } from "lucide-react";
import { Line, LineChart } from "recharts";
import {
  ChartGrid,
  ChartXAxis,
  ChartYAxis,
} from "@/client/components/ChartAxes";
import { SegmentedToggle } from "@/client/components/SegmentedToggle";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/client/components/ui/chart";
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/client/components/ui/card";
import { formatCount } from "@/client/features/search-performance/SearchPerformanceColumns";
import type { Report } from "@/client/features/search-performance/SearchPerformanceColumns";

const METRICS = [
  { value: "clicks", label: "Clicks", icon: <MousePointerClick /> },
  { value: "impressions", label: "Impressions", icon: <Eye /> },
] as const;
type Metric = (typeof METRICS)[number]["value"];
const CONFIG = {
  clicks: { label: "Clicks", color: "var(--chart-1)" },
  impressions: { label: "Impressions", color: "var(--chart-2)" },
} satisfies ChartConfig;

function formatDay(date: string): string {
  return new Date(`${date}T00:00:00Z`).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

export function SearchPerformanceTrend({ report }: { report: Report }) {
  const [metric, setMetric] = useState<Metric>("clicks");
  return (
    <Card className="gap-0">
      <CardHeader className="flex-row items-center justify-between border-b py-3">
        <div>
          <CardTitle>Search trend</CardTitle>
          <p className="mt-1 text-xs text-muted-foreground">
            Daily {metric} from {report.range.startDate} to{" "}
            {report.range.endDate}
          </p>
        </div>
        <CardAction>
          <SegmentedToggle
            items={[...METRICS]}
            value={metric}
            onChange={setMetric}
            showLabels
          />
        </CardAction>
      </CardHeader>
      <CardContent className="pt-4">
        {!report.range.comparisonAvailable ? (
          <p className="mb-3 text-xs text-muted-foreground" role="status">
            Comparison is unavailable beyond Search Console&apos;s 16-month
            history.
          </p>
        ) : null}
        {report.daily.length ? (
          <ChartContainer config={CONFIG} className="h-56">
            <LineChart
              data={report.daily}
              margin={{ top: 8, right: 8, left: 0 }}
            >
              <ChartGrid />
              <ChartXAxis dataKey="date" tickFormatter={formatDay} />
              <ChartYAxis allowDecimals={false} tickFormatter={formatCount} />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(label: unknown) =>
                      typeof label === "string" ? formatDay(label) : ""
                    }
                    valueFormatter={(value) => formatCount(Number(value))}
                  />
                }
              />
              <Line
                type="monotone"
                dataKey={metric}
                stroke={`var(--color-${metric})`}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
                isAnimationActive={false}
              />
            </LineChart>
          </ChartContainer>
        ) : (
          <p className="py-10 text-center text-sm text-muted-foreground">
            No daily search data for this range.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
