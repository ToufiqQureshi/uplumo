import { useQuery } from "@tanstack/react-query";
import { Bar, BarChart } from "recharts";
import {
  ChartGrid,
  ChartXAxis,
  ChartYAxis,
} from "@/client/components/ChartAxes";
import { Card, CardContent } from "@/client/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/client/components/ui/chart";
import { CountTable, QueryView, Section, StatTile } from "./StaffParts";
import { formatMoney, overviewQueryOptions, type Money } from "./staffApi";

const signupConfig = {
  count: { label: "Signups", color: "var(--chart-1)" },
} satisfies ChartConfig;

function moneyLine(values: Money[]) {
  const shown = values.filter((v) => v.amount > 0);
  if (shown.length === 0) return formatMoney(0, "INR");
  return shown.map((v) => formatMoney(v.amount, v.currency)).join(" + ");
}

export function OverviewPage() {
  const overview = useQuery(overviewQueryOptions());
  return (
    <QueryView query={overview}>
      {(o) => (
        <div className="space-y-8">
          <Section title="Users">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatTile
                label="Total users"
                value={o.users}
                hint={`${o.verifiedUsers} verified · ${o.blockedUsers} blocked`}
              />
              <StatTile label="New today" value={o.newToday} />
              <StatTile label="New in 7 days" value={o.new7Days} />
              <StatTile label="New in 30 days" value={o.new30Days} />
            </div>
          </Section>
          <Section
            title="Money"
            description="MRR is every active subscription at its plan price. Collected is real payments after refunds."
          >
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatTile label="Paying workspaces" value={o.payingWorkspaces} />
              <StatTile label="MRR" value={moneyLine(o.mrr)} />
              <StatTile
                label="Collected this month"
                value={moneyLine(o.collectedThisMonth)}
              />
              <StatTile
                label="Collected last month"
                value={moneyLine(o.collectedLastMonth)}
              />
            </div>
          </Section>
          <Section title="Subscriptions needing attention">
            <div className="grid gap-4 sm:grid-cols-3">
              <StatTile
                label="Ending in 7 days"
                value={o.endingNext7Days}
                hint="Cancelled, still running until the period ends"
              />
              <StatTile
                label="Payment failing"
                value={o.paymentFailing}
                hint="Razorpay status halted or pending"
              />
              <StatTile
                label="Cancelled in 30 days"
                value={o.cancelledLast30Days}
              />
            </div>
          </Section>
          <div className="grid gap-6 lg:grid-cols-3">
            <Card className="lg:col-span-2">
              <CardContent className="space-y-2">
                <h2 className="text-lg font-semibold">Signups, last 30 days</h2>
                <ChartContainer config={signupConfig} className="h-56 w-full">
                  <BarChart
                    data={o.signupsByDay.map((d) => ({
                      ...d,
                      label: d.day.slice(5),
                    }))}
                    margin={{ top: 8, right: 8, bottom: 4, left: 0 }}
                  >
                    <ChartGrid />
                    <ChartXAxis dataKey="label" minTickGap={16} />
                    <ChartYAxis allowDecimals={false} />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Bar
                      dataKey="count"
                      fill="var(--color-count)"
                      radius={[4, 4, 0, 0]}
                    />
                  </BarChart>
                </ChartContainer>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="space-y-3">
                <h2 className="text-lg font-semibold">Workspaces by plan</h2>
                <CountTable
                  rows={o.planCounts.map((p) => ({
                    ...p,
                    label: p.label.charAt(0).toUpperCase() + p.label.slice(1),
                  }))}
                />
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </QueryView>
  );
}
