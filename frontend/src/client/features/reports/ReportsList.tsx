import { Link } from "@tanstack/react-router";
import { FileText, Sparkles, Trash2 } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { EmptyState } from "@/client/components/EmptyState";
import { RowActionsMenu } from "@/client/components/RowActionsMenu";
import { DropdownMenuItem } from "@/client/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCard,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import { formatCreatedBy } from "@/client/features/reports/shared";
import { formatRelativeTime } from "@/client/lib/relative-time";
import type { ReportListItem } from "@/client/features/reports/reportApi";
import { REPORT_APP_LIST_LIMIT } from "@/types/schemas/reports";

export function ReportsList({
  projectId,
  reports,
  onDelete,
  onCreate,
}: {
  projectId: string;
  reports: ReportListItem[];
  onDelete: (report: ReportListItem) => void;
  onCreate: () => void;
}) {
  if (reports.length === 0) {
    return (
      <EmptyState
        icon={FileText}
        title="No reports yet"
        description="Turn this project's search, traffic, rankings and site health into an interactive report you can send to a client. Uses no credits."
        action={
          <Button onClick={onCreate}>
            <Sparkles data-icon="inline-start" />
            Create client report
          </Button>
        }
      />
    );
  }

  return (
    <div className="space-y-3">
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Title</TableHead>
              <TableHead>Created by</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Updated</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {reports.map((report) => (
              <TableRow key={report.id}>
                <TableCell className="max-w-[420px]">
                  <Link
                    to="/p/$projectId/reports/$reportId"
                    params={{ projectId, reportId: report.id }}
                    className="font-medium hover:underline"
                  >
                    {report.title}
                  </Link>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {formatCreatedBy(report)}
                </TableCell>
                {/* The template the report was written from, else the skill
                    that produced it: what a reader needs to tell two reports
                    on the same site apart. */}
                <TableCell className="text-muted-foreground">
                  {report.templateName ?? report.skill ?? "—"}
                </TableCell>
                <TableCell className="whitespace-nowrap text-muted-foreground">
                  {formatRelativeTime(report.updatedAt)}
                </TableCell>
                <TableCell className="w-10 text-right">
                  <RowActionsMenu label={`Actions for ${report.title}`}>
                    <DropdownMenuItem
                      variant="destructive"
                      onClick={() => onDelete(report)}
                    >
                      <Trash2 />
                      Delete
                    </DropdownMenuItem>
                  </RowActionsMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableCard>
      {reports.length === REPORT_APP_LIST_LIMIT ? (
        <p className="text-xs text-muted-foreground">
          Showing the {REPORT_APP_LIST_LIMIT} most recent reports.
        </p>
      ) : null}
    </div>
  );
}
