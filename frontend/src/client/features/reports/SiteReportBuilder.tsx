import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Check } from "lucide-react";
import { useState, type FormEvent } from "react";
import { toast } from "sonner";
import { cn } from "cn";
import { Button } from "@/client/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/client/components/ui/dialog";
import { Input } from "@/client/components/ui/input";
import { Textarea } from "@/client/components/ui/textarea";
import { captureClientEvent } from "@/client/lib/posthog";
import {
  previewSiteReport,
  SITE_REPORT_SECTIONS,
  type SiteReportSection,
} from "@/client/features/reports/reportApi";
import {
  SiteReportPeriod,
  reportPeriodError,
  reportPeriodLabel,
  type ReportPeriod,
} from "@/client/features/reports/SiteReportPeriod";
import { SiteReportPreviewDialog } from "@/client/features/reports/SiteReportPreviewDialog";
import { reportsQueryKey } from "@/client/features/reports/shared";
import { SITE_REPORT_SECTION_DETAILS } from "@/client/features/reports/siteReportSections";
import { sharePath } from "@/shared/report-share";

export function SiteReportBuilder({
  projectId,
  onClose,
}: {
  projectId: string;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [period, setPeriod] = useState<ReportPeriod>({ key: 28 });
  const [customTitle, setCustomTitle] = useState<string>();
  const title = customTitle ?? `SEO report ? ${reportPeriodLabel(period)}`;
  const [sections, setSections] = useState<SiteReportSection[]>([
    ...SITE_REPORT_SECTIONS,
  ]);
  const [note, setNote] = useState("");
  const [workCompleted, setWorkCompleted] = useState("");
  const [nextSteps, setNextSteps] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const mutation = useMutation({
    retry: false,
    meta: { errorToast: false },
    mutationFn: () =>
      previewSiteReport({
        data: {
          projectId,
          title: title.trim(),
          ...("startDate" in period
            ? { startDate: period.startDate, endDate: period.endDate }
            : { periodDays: period.key }),
          sections,
          note,
          workCompleted,
          nextSteps,
        },
      }),
  });

  const toggle = (section: SiteReportSection) =>
    setSections((current) =>
      current.includes(section)
        ? current.filter((item) => item !== section)
        : SITE_REPORT_SECTIONS.filter(
            (item) => item === section || current.includes(item),
          ),
    );

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!title.trim()) return setFormError("Add a report title.");
    if (sections.length === 0)
      return setFormError("Choose at least one section.");
    const dateError = reportPeriodError(period);
    if (dateError) return setFormError(dateError);
    setFormError(null);
    mutation.mutate();
  };

  const error =
    formError ??
    (mutation.isError
      ? mutation.error instanceof Error
        ? mutation.error.message
        : "Could not preview the report."
      : null);

  if (mutation.data) {
    return (
      <SiteReportPreviewDialog
        projectId={projectId}
        title={title.trim()}
        draft={mutation.data}
        onBack={mutation.reset}
        onClose={onClose}
        onSaved={(result) => {
          captureClientEvent("report:created", {
            project_id: projectId,
            report_id: result.reportId,
            source: "site_report",
          });
          if (result.shareFailed) {
            toast.warning(
              "Report saved privately. Client link creation failed.",
            );
          } else if (result.shareToken) {
            const link = `${window.location.origin}${sharePath(result.shareToken)}`;
            void navigator.clipboard.writeText(link).then(
              () => toast.success("Report saved. Client link copied."),
              () => toast.success("Report saved. Open Share to copy the link."),
            );
          } else {
            toast.success("Report saved privately.");
          }
          void queryClient.invalidateQueries({
            queryKey: reportsQueryKey(projectId),
          });
          void navigate({
            to: "/p/$projectId/reports/$reportId",
            params: { projectId, reportId: result.reportId },
          });
        }}
      />
    );
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) onClose();
      }}
    >
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Create a client report</DialogTitle>
          <DialogDescription>
            Built from this project's connected data into an interactive page
            you can send to your client.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-6">
          <fieldset
            disabled={mutation.isPending}
            className="m-0 min-w-0 space-y-6 border-0 p-0"
          >
            <legend className="sr-only">Report settings</legend>

            <SiteReportPeriod
              period={period}
              onChange={(value) => {
                setPeriod(value);
                setFormError(null);
              }}
            />

            <div className="space-y-2">
              <p className="text-sm font-medium" id="sections-label">
                Sections
              </p>
              <div
                role="group"
                aria-labelledby="sections-label"
                className="grid gap-2 sm:grid-cols-2"
              >
                {SITE_REPORT_SECTIONS.map((section) => {
                  const {
                    label,
                    detail,
                    icon: Icon,
                  } = SITE_REPORT_SECTION_DETAILS[section];
                  const on = sections.includes(section);
                  return (
                    <button
                      key={section}
                      type="button"
                      aria-pressed={on}
                      onClick={() => toggle(section)}
                      className={cn(
                        "flex items-start gap-3 rounded-xl border p-3 text-left transition-colors",
                        on
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted/50",
                      )}
                    >
                      <span
                        className={cn(
                          "grid size-8 shrink-0 place-items-center rounded-lg",
                          on
                            ? "bg-primary text-primary-foreground"
                            : "bg-muted text-muted-foreground",
                        )}
                      >
                        <Icon className="size-4" aria-hidden />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block text-sm font-medium">
                          {label}
                        </span>
                        <span className="block text-xs text-muted-foreground">
                          {detail}
                        </span>
                      </span>
                      <span
                        aria-hidden
                        className={cn(
                          "mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border",
                          on
                            ? "border-primary bg-primary text-primary-foreground"
                            : "border-border",
                        )}
                      >
                        {on ? <Check className="size-3" /> : null}
                      </span>
                    </button>
                  );
                })}
              </div>
              <p className="text-xs text-muted-foreground">
                Sections without data yet are left out, and we tell you why.
              </p>
            </div>

            <div className="space-y-2">
              <label
                htmlFor="site-report-title"
                className="text-sm font-medium"
              >
                Title
              </label>
              <Input
                id="site-report-title"
                value={title}
                maxLength={120}
                onChange={(event) => setCustomTitle(event.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="site-report-note" className="text-sm font-medium">
                Message to your client{" "}
                <span className="font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
              <Textarea
                id="site-report-note"
                value={note}
                maxLength={4000}
                rows={3}
                onChange={(event) => setNote(event.target.value)}
                placeholder="Add context the client needs to understand the results."
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="site-report-work" className="text-sm font-medium">
                Work completed{" "}
                <span className="font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
              <Textarea
                id="site-report-work"
                value={workCompleted}
                maxLength={4000}
                rows={3}
                onChange={(event) => setWorkCompleted(event.target.value)}
                placeholder="List what changed and when. Put evidence links on their own lines."
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="site-report-next" className="text-sm font-medium">
                Next steps{" "}
                <span className="font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
              <Textarea
                id="site-report-next"
                value={nextSteps}
                maxLength={4000}
                rows={3}
                onChange={(event) => setNextSteps(event.target.value)}
                placeholder="List the next priorities and why they matter."
              />
            </div>
          </fieldset>

          {error ? (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={onClose}
              disabled={mutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Building preview…" : "Preview report"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
