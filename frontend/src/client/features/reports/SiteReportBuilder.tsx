import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import {
  Activity,
  Check,
  Gauge,
  Link2,
  Search,
  TrendingUp,
  type LucideIcon,
} from "lucide-react";
import { useId, useState, type FormEvent } from "react";
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
import { Switch } from "@/client/components/ui/switch";
import { Textarea } from "@/client/components/ui/textarea";
import { captureClientEvent } from "@/client/lib/posthog";
import {
  generateSiteReport,
  SITE_REPORT_SECTIONS,
  type SiteReportResult,
  type SiteReportSection,
} from "@/client/features/reports/reportApi";
import { reportsQueryKey } from "@/client/features/reports/shared";
import { sharePath } from "@/shared/report-share";

const SECTIONS: Record<
  SiteReportSection,
  { label: string; detail: string; icon: LucideIcon }
> = {
  search: {
    label: "Google Search",
    detail: "Clicks, impressions, top searches and pages, rising and falling",
    icon: Search,
  },
  traffic: {
    label: "Organic traffic",
    detail: "Visits, visitors, engagement and key events from Analytics",
    icon: Activity,
  },
  rankings: {
    label: "Keyword rankings",
    detail: "Tracked positions, movers and ranking pages",
    icon: TrendingUp,
  },
  audit: {
    label: "Site health",
    detail: "Audit issues with pages, speed and quality scores",
    icon: Gauge,
  },
  backlinks: {
    label: "Backlinks",
    detail: "Linking websites gained and lost",
    icon: Link2,
  },
};

const PERIODS = [7, 28, 90] as const;
type PeriodDays = (typeof PERIODS)[number];

function formatDay(date: Date, withYear: boolean) {
  return date.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    ...(withYear ? { year: "numeric" } : {}),
    timeZone: "UTC",
  });
}

// Mirrors the server: the window ends on Search Console's last final day,
// three days before today (UTC).
export function reportPeriod(days: PeriodDays, now = new Date()) {
  const end = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - 3),
  );
  const start = new Date(end);
  start.setUTCDate(end.getUTCDate() - (days - 1));
  return `${formatDay(start, start.getUTCFullYear() !== end.getUTCFullYear())} – ${formatDay(end, true)}`;
}

export function SiteReportBuilder({
  projectId,
  onClose,
}: {
  projectId: string;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const shareId = useId();
  const [periodDays, setPeriodDays] = useState<PeriodDays>(28);
  const [title, setTitle] = useState(() => `SEO report · ${reportPeriod(28)}`);
  const [titleEdited, setTitleEdited] = useState(false);
  const [sections, setSections] = useState<SiteReportSection[]>([
    ...SITE_REPORT_SECTIONS,
  ]);
  const [note, setNote] = useState("");
  const [nextSteps, setNextSteps] = useState("");
  const [share, setShare] = useState(true);
  const [formError, setFormError] = useState<string | null>(null);

  const mutation = useMutation({
    // Free, but never repeated behind the user's back: a retry would save a
    // second copy or fail on the duplicate title.
    retry: false,
    meta: { errorToast: false },
    mutationFn: () =>
      generateSiteReport({
        data: {
          projectId,
          title: title.trim(),
          periodDays,
          sections,
          note,
          nextSteps,
          share,
        },
      }),
    onSuccess: async (result) => {
      captureClientEvent("report:created", {
        project_id: projectId,
        report_id: result.reportId,
        source: "site_report",
      });
      await announce(result);
      void queryClient.invalidateQueries({
        queryKey: reportsQueryKey(projectId),
      });
      void navigate({
        to: "/p/$projectId/reports/$reportId",
        params: { projectId, reportId: result.reportId },
      });
    },
  });

  const choosePeriod = (days: PeriodDays) => {
    setPeriodDays(days);
    if (!titleEdited) setTitle(`SEO report · ${reportPeriod(days)}`);
  };

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
    setFormError(null);
    mutation.mutate();
  };

  const error =
    formError ??
    (mutation.isError
      ? mutation.error instanceof Error
        ? mutation.error.message
        : "Could not create the report."
      : null);

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
            you can send to your client. Uses no credits.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-6">
          <fieldset
            disabled={mutation.isPending}
            className="m-0 min-w-0 space-y-6 border-0 p-0"
          >
            <legend className="sr-only">Report settings</legend>

            <div className="space-y-2">
              <p className="text-sm font-medium" id="period-label">
                Period
              </p>
              <div
                role="group"
                aria-labelledby="period-label"
                className="inline-flex rounded-lg border border-border bg-muted/40 p-1"
              >
                {PERIODS.map((days) => (
                  <button
                    key={days}
                    type="button"
                    aria-pressed={periodDays === days}
                    onClick={() => choosePeriod(days)}
                    className={cn(
                      "rounded-md px-3.5 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground",
                      periodDays === days &&
                        "bg-background text-foreground shadow-sm",
                    )}
                  >
                    {days === 90 ? "3 months" : `${days} days`}
                  </button>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                {reportPeriod(periodDays)}, compared with the {periodDays} days
                before.
              </p>
            </div>

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
                  const { label, detail, icon: Icon } = SECTIONS[section];
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
                onChange={(event) => {
                  setTitle(event.target.value);
                  setTitleEdited(true);
                }}
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
                placeholder="What you worked on this period and what the numbers mean."
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
                placeholder="What you will do next. Separate items with a blank line."
              />
            </div>

            <div className="flex items-start justify-between gap-4 rounded-xl border border-border p-4">
              <div>
                <label htmlFor={shareId} className="text-sm font-medium">
                  Create a client link
                </label>
                <p className="text-xs text-muted-foreground">
                  Anyone with the link can view the report without signing in.
                  You can turn it off any time.
                </p>
              </div>
              <Switch id={shareId} checked={share} onCheckedChange={setShare} />
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
              {mutation.isPending ? "Building report…" : "Create report"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

async function announce(result: SiteReportResult) {
  const left = result.skipped.map((part) => part.reason).join(" ");
  if (result.shareToken) {
    const url = `${window.location.origin}${sharePath(result.shareToken)}`;
    let copied = false;
    try {
      await navigator.clipboard.writeText(url);
      copied = true;
    } catch {
      // The link is on the report page's Share button either way.
    }
    toast.success(
      copied ? "Report ready. Client link copied." : "Report ready.",
      { description: left || undefined, duration: left ? 10_000 : 4_000 },
    );
    return;
  }
  toast.success("Report ready.", {
    description: left || undefined,
    duration: left ? 10_000 : 4_000,
  });
}
