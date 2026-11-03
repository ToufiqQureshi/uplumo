import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/client/components/ui/badge";
import {
  Table,
  TableBody,
  TableCard,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@/client/components/ui/toggle-group";
import { QueryView } from "./StaffParts";
import {
  dateFormat,
  formatMoney,
  subscriptionsQueryOptions,
  type StaffSubscription,
} from "./staffApi";

const live = new Set(["authenticated", "active", "pending"]);
const DAY = 24 * 60 * 60 * 1000;

const filters = ["all", "live", "ending", "failing", "cancelled"] as const;
type Filter = (typeof filters)[number];

function matches(s: StaffSubscription, filter: Filter) {
  const end = s.currentPeriodEnd ? new Date(s.currentPeriodEnd).getTime() : 0;
  switch (filter) {
    case "live":
      return live.has(s.status);
    case "ending":
      return live.has(s.status) && end > 0 && end - Date.now() < 30 * DAY;
    case "failing":
      return s.status === "halted" || s.status === "pending";
    case "cancelled":
      return s.status === "cancelled" || s.cancelAtPeriodEnd;
    default:
      return true;
  }
}

function statusLabel(s: StaffSubscription) {
  if (s.cancelAtPeriodEnd && live.has(s.status)) return "Cancelled, running";
  if (s.status === "halted") return "Payment failing";
  return s.status.charAt(0).toUpperCase() + s.status.slice(1);
}

export function SubscriptionsPage() {
  const subs = useQuery(subscriptionsQueryOptions());
  const [filter, setFilter] = useState<Filter>("all");
  return (
    <QueryView query={subs}>
      {({ subscriptions }) => {
        const rows = subscriptions.filter((s) => matches(s, filter));
        return (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <ToggleGroup
                value={[filter]}
                onValueChange={(v) => {
                  const next = filters.find((f) => f === v[0]);
                  if (next) setFilter(next);
                }}
                variant="outline"
                aria-label="Filter subscriptions"
              >
                <ToggleGroupItem value="all">All</ToggleGroupItem>
                <ToggleGroupItem value="live">Paying</ToggleGroupItem>
                <ToggleGroupItem value="ending">
                  Renews in 30 days
                </ToggleGroupItem>
                <ToggleGroupItem value="failing">
                  Payment failing
                </ToggleGroupItem>
                <ToggleGroupItem value="cancelled">Cancelled</ToggleGroupItem>
              </ToggleGroup>
              <p className="text-sm text-muted-foreground" role="status">
                {rows.length} of {subscriptions.length}
              </p>
            </div>
            <TableCard>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Workspace</TableHead>
                    <TableHead>Plan</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Current period ends</TableHead>
                    <TableHead>Last change</TableHead>
                    <TableHead className="text-right">Paid in total</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((s) => (
                    <TableRow key={s.organizationId}>
                      <TableCell>
                        <div className="font-medium">
                          {s.ownerEmail || "No owner"}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {s.workspace}
                        </div>
                      </TableCell>
                      <TableCell className="capitalize">
                        {s.plan} · {s.currency}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            live.has(s.status) && !s.cancelAtPeriodEnd
                              ? "default"
                              : "outline"
                          }
                        >
                          {statusLabel(s)}
                        </Badge>
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        {s.currentPeriodEnd
                          ? dateFormat.format(new Date(s.currentPeriodEnd))
                          : "—"}
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        {dateFormat.format(new Date(s.updatedAt))}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {formatMoney(s.paidTotal, s.currency)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableCard>
          </div>
        );
      }}
    </QueryView>
  );
}
