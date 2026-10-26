import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { BarChart3, Database, Share2, Plus } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ReportsList } from "@/client/features/reports/ReportsList";
import {
  DeleteReportModal,
  reportsQueryKey,
  useDeleteReport,
} from "@/client/features/reports/shared";
import { QueryState } from "@/client/components/QueryState";
import {
  listReports,
  type ReportListItem,
} from "@/client/features/reports/reportApi";
import { REPORT_APP_LIST_LIMIT } from "@/types/schemas/reports";
import { z } from "zod";
import { ReportBuilder } from "@/client/features/reports/ReportBuilder";
import { ManualReportForm } from "@/client/features/reports/ManualReportForm";

export const Route = createFileRoute("/_app/p/$projectId/reports/")({
  validateSearch: z.object({
    builder: z.boolean().optional(),
    edit: z.string().optional(),
  }),
  component: ReportsPage,
});

function ReportsPage() {
  const { projectId } = Route.useParams();
  const { builder, edit } = Route.useSearch();
  const navigate = useNavigate();
  const [pendingDelete, setPendingDelete] = useState<ReportListItem | null>(
    null,
  );
  const [manualReportOpen, setManualReportOpen] = useState(false);

  const reportsQuery = useQuery({
    queryKey: reportsQueryKey(projectId),
    queryFn: () =>
      listReports({ data: { projectId, limit: REPORT_APP_LIST_LIMIT } }),
    // This page exists to inspect what an agent just wrote; the app-wide
    // 5-minute staleTime would show a pre-save list as current.
    staleTime: 0,
  });

  const deleteMutation = useDeleteReport(projectId, () =>
    setPendingDelete(null),
  );

  if (builder)
    return (
      <ReportBuilder
        key={`${projectId}-${edit ?? "new"}`}
        projectId={projectId}
        reportId={edit}
      />
    );

  return (
    <div className="overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-7xl space-y-4">
        <PageHeader
          title="Reports"
          description="Turn your data into a report your client can understand and act on."
          actions={
            <div className="flex flex-wrap gap-2">
              <Button
                variant="ghost"
                nativeButton={false}
                render={
                  <Link
                    to="/p/$projectId/reports/templates"
                    params={{ projectId }}
                  />
                }
              >
                AI templates
              </Button>
              <Button
                variant="outline"
                onClick={() => setManualReportOpen(true)}
              >
                <Plus data-icon="inline-start" />
                Write notes
              </Button>
              <Button
                onClick={() =>
                  void navigate({
                    to: "/p/$projectId/reports",
                    params: { projectId },
                    search: { builder: true },
                  })
                }
              >
                <Plus />
                Create report
              </Button>
            </div>
          }
        />

        <section
          className="grid gap-6 rounded-lg border border-border bg-card px-6 py-7 md:grid-cols-[1.2fr_1fr]"
          aria-label="Build a report"
        >
          <div>
            <p className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Your data. Your story.
            </p>
            <h2 className="text-xl font-semibold tracking-tight">
              Build the report you actually need.
            </h2>
            <p className="mt-2 max-w-lg text-sm leading-relaxed text-muted-foreground">
              Choose your metrics, create charts, and explain what happens next.
              Start with Uplumo, Search Console, GA4, or a CSV.
            </p>
            <Button
              className="mt-5"
              onClick={() =>
                void navigate({
                  to: "/p/$projectId/reports",
                  params: { projectId },
                  search: { builder: true },
                })
              }
            >
              <BarChart3 />
              Open report builder
            </Button>
          </div>
          <ol className="grid gap-4 self-center text-sm">
            {[
              {
                icon: Database,
                title: "Bring your data",
                text: "Import a date range or upload your own CSV.",
              },
              {
                icon: BarChart3,
                title: "Make it visual",
                text: "Add KPIs, charts, tables, filters and your findings.",
              },
              {
                icon: Share2,
                title: "Share the result",
                text: "Save privately, then share a link or export a PDF.",
              },
            ].map((step, index) => (
              <li key={step.title} className="flex items-start gap-3">
                <span className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border bg-muted/40">
                  <step.icon className="size-4" />
                </span>
                <div>
                  <p className="font-medium">
                    {index + 1}. {step.title}
                  </p>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {step.text}
                  </p>
                </div>
              </li>
            ))}
          </ol>
        </section>
        <h2 className="pt-3 text-sm font-semibold">Saved reports</h2>
        <QueryState query={reportsQuery} errorFallback="Failed to load reports">
          {(data) => (
            <ReportsList
              projectId={projectId}
              reports={data.reports}
              onDelete={setPendingDelete}
            />
          )}
        </QueryState>
      </div>

      {pendingDelete ? (
        <DeleteReportModal
          title={pendingDelete.title}
          isPending={deleteMutation.isPending}
          onClose={() => setPendingDelete(null)}
          onConfirm={() => deleteMutation.mutate(pendingDelete.id)}
        />
      ) : null}
      {manualReportOpen ? (
        <ManualReportForm
          projectId={projectId}
          onClose={() => setManualReportOpen(false)}
        />
      ) : null}
    </div>
  );
}
