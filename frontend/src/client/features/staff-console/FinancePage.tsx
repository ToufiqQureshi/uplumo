import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { Card, CardContent } from "@/client/components/ui/card";
import {
  Table,
  TableBody,
  TableCard,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { QueryView, Section } from "./StaffParts";
import {
  financeQueryOptions,
  formatMoney,
  formatUSD,
  syncPayments,
} from "./staffApi";

function SyncButton() {
  const queryClient = useQueryClient();
  const sync = useMutation({
    mutationFn: syncPayments,
    retry: false,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["staff"] }),
  });
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Button
        variant="outline"
        disabled={sync.isPending}
        onClick={() => sync.mutate()}
      >
        <RefreshCw aria-hidden className="size-4" />
        {sync.isPending ? "Syncing…" : "Sync from Razorpay"}
      </Button>
      <p className="text-sm text-muted-foreground" role="status">
        {sync.isSuccess
          ? `Checked ${sync.data.subscriptions} subscriptions, added ${sync.data.added} payments${sync.data.failed ? `, ${sync.data.failed} failed` : ""}.`
          : sync.isError
            ? getStandardErrorMessage(sync.error)
            : "Imports subscription payments made before the ledger existed. Safe to run again."}
      </p>
    </div>
  );
}

export function FinancePage() {
  const finance = useQuery(financeQueryOptions());
  return (
    <div className="space-y-8">
      <SyncButton />
      <QueryView query={finance}>
        {(f) => {
          const costByMonth = new Map(
            f.providerCost.map((c) => [c.month, c.costUsd]),
          );
          const months = [
            ...new Set([
              ...f.revenue.map((r) => r.month),
              ...f.providerCost.map((c) => c.month),
            ]),
          ];
          months.sort((a, b) => b.localeCompare(a));
          return (
            <>
              <Section
                title="Money in and provider cost, by month"
                description="Revenue is real payments in each currency (never converted). Provider cost is DataForSEO and OpenRouter spend in US dollars."
              >
                <TableCard>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Month</TableHead>
                        <TableHead className="text-right">Revenue</TableHead>
                        <TableHead className="text-right">Refunded</TableHead>
                        <TableHead className="text-right">Payments</TableHead>
                        <TableHead className="text-right">
                          Provider cost
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {months.length === 0 ? (
                        <TableRow>
                          <TableCell
                            colSpan={5}
                            className="text-muted-foreground"
                          >
                            No payments or provider cost yet.
                          </TableCell>
                        </TableRow>
                      ) : (
                        months.map((month) => {
                          const rows = f.revenue.filter(
                            (r) => r.month === month,
                          );
                          return (
                            <TableRow key={month}>
                              <TableCell>{month}</TableCell>
                              <TableCell className="text-right tabular-nums">
                                {rows.length
                                  ? rows
                                      .map((r) =>
                                        formatMoney(r.gross, r.currency),
                                      )
                                      .join(" + ")
                                  : "—"}
                              </TableCell>
                              <TableCell className="text-right tabular-nums">
                                {rows.some((r) => r.refunded > 0)
                                  ? rows
                                      .filter((r) => r.refunded > 0)
                                      .map((r) =>
                                        formatMoney(r.refunded, r.currency),
                                      )
                                      .join(" + ")
                                  : "—"}
                              </TableCell>
                              <TableCell className="text-right tabular-nums">
                                {rows.reduce((n, r) => n + r.payments, 0)}
                              </TableCell>
                              <TableCell className="text-right tabular-nums">
                                {formatUSD(costByMonth.get(month) ?? 0)}
                              </TableCell>
                            </TableRow>
                          );
                        })
                      )}
                    </TableBody>
                  </Table>
                </TableCard>
              </Section>
              <div className="grid gap-6 lg:grid-cols-2">
                <Card>
                  <CardContent className="space-y-3">
                    <h2 className="text-lg font-semibold">
                      Provider cost by feature, last 30 days
                    </h2>
                    {f.costByFeature30Days.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        No provider cost yet.
                      </p>
                    ) : (
                      <ul className="space-y-1 text-sm">
                        {f.costByFeature30Days.map((c) => (
                          <li key={c.feature} className="flex justify-between">
                            <span>{c.feature}</span>
                            <span className="tabular-nums">
                              {formatUSD(c.costUsd)}
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="space-y-3">
                    <h2 className="text-lg font-semibold">
                      Costliest workspaces, last 30 days
                    </h2>
                    {f.topCostWorkspaces30Days.length === 0 ? (
                      <p className="text-sm text-muted-foreground">
                        No provider cost yet.
                      </p>
                    ) : (
                      <ul className="space-y-2 text-sm">
                        {f.topCostWorkspaces30Days.map((w) => (
                          <li
                            key={w.organizationId}
                            className="flex justify-between gap-3"
                          >
                            <span className="min-w-0">
                              <span className="block truncate">
                                {w.ownerEmail || w.workspace}
                              </span>
                              <span className="text-xs capitalize text-muted-foreground">
                                {w.plan}
                              </span>
                            </span>
                            <span className="tabular-nums">
                              {formatUSD(w.costUsd)}
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </CardContent>
                </Card>
              </div>
            </>
          );
        }}
      </QueryView>
    </div>
  );
}
