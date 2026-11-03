import { useQuery } from "@tanstack/react-query";
import {
  Table,
  TableBody,
  TableCard,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import { QueryView } from "./StaffParts";
import { actionsQueryOptions } from "./staffApi";

const labels: Record<string, string> = {
  block: "Blocked",
  unblock: "Unblocked",
  delete: "Deleted",
  sync_payments: "Synced payments",
};

const timeFormat = new Intl.DateTimeFormat("en-IN", {
  dateStyle: "medium",
  timeStyle: "short",
});

export function ActivityPage() {
  const actions = useQuery(actionsQueryOptions());
  return (
    <QueryView query={actions}>
      {({ actions: rows }) => (
        <TableCard>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Staff</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>User</TableHead>
                <TableHead>Detail</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-muted-foreground">
                    No staff actions yet.
                  </TableCell>
                </TableRow>
              ) : (
                rows.map((a, i) => (
                  <TableRow key={`${a.createdAt}-${i}`}>
                    <TableCell className="whitespace-nowrap">
                      {timeFormat.format(new Date(a.createdAt))}
                    </TableCell>
                    <TableCell>{a.actorEmail || "—"}</TableCell>
                    <TableCell>{labels[a.action] ?? a.action}</TableCell>
                    <TableCell>
                      {a.targetEmail ||
                        (a.targetId
                          ? `Deleted user ${a.targetId.slice(0, 8)}`
                          : "—")}
                    </TableCell>
                    <TableCell>{a.detail || "—"}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableCard>
      )}
    </QueryView>
  );
}
