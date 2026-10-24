import { Link } from "@tanstack/react-router";
import { Sparkles, Activity } from "lucide-react";
import { SegmentedToggle } from "@/client/components/SegmentedToggle";
import { SearchGrowthPanel } from "@/client/features/search-performance/SearchGrowthPanel";
import { AnalyticsPage } from "./AnalyticsPage";
import type { RangeDays } from "./analyticsApi";

const VIEWS = [
  { value: "intelligence", label: "SEO insights", icon: <Sparkles /> },
  { value: "traffic", label: "Tracking", icon: <Activity /> },
] as const;

export function AnalyticsWorkspace({
  projectId,
  days,
  view,
  onChange,
}: {
  projectId: string;
  days: RangeDays;
  view: "intelligence" | "traffic";
  onChange: (update: {
    days?: RangeDays;
    view?: "intelligence" | "traffic";
  }) => void;
}) {
  return (
    <div className="flex h-full min-w-0 flex-col">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-card px-4 py-3 md:px-6">
        <SegmentedToggle
          items={[...VIEWS]}
          value={view}
          onChange={(value) =>
            onChange({ view: value === "traffic" ? "traffic" : "intelligence" })
          }
          showLabels
        />
        <Link
          className="text-xs font-medium text-muted-foreground underline underline-offset-4"
          to="/p/$projectId/search-performance"
          params={{ projectId }}
        >
          Explore Search Console queries and pages
        </Link>
      </div>
      {view === "traffic" ? (
        <AnalyticsPage
          projectId={projectId}
          days={days}
          onDaysChange={(nextDays) => onChange({ days: nextDays })}
        />
      ) : (
        <div className="min-h-0 flex-1 overflow-auto bg-violet-500/[0.025] px-4 py-5 pb-24 md:px-6">
          <div className="mx-auto max-w-7xl">
            <SearchGrowthPanel key={projectId} projectId={projectId} />
          </div>
        </div>
      )}
    </div>
  );
}
