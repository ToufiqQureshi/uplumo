import { Input } from "@/client/components/ui/input";
import { cn } from "cn";

const periods = [
  { key: 7, label: "7 days" },
  { key: 28, label: "28 days" },
  { key: 90, label: "3 months" },
  { key: "last-month", label: "Last month" },
  { key: "custom", label: "Custom" },
] as const;

type PeriodDays = 7 | 28 | 90;
type PeriodKey = (typeof periods)[number]["key"];
type Dates = { startDate: string; endDate: string };
export type ReportPeriod =
  | { key: PeriodDays }
  | ({ key: "last-month" | "custom" } & Dates);

function formatDay(date: Date, withYear: boolean) {
  return date.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    ...(withYear ? { year: "numeric" } : {}),
    timeZone: "UTC",
  });
}

const dayISO = (date: Date) => date.toISOString().slice(0, 10);

function latestDate(now: Date) {
  return new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - 3),
  );
}

function rollingDates(days: PeriodDays, now: Date): Dates {
  const end = latestDate(now);
  const start = new Date(end);
  start.setUTCDate(end.getUTCDate() - (days - 1));
  return { startDate: dayISO(start), endDate: dayISO(end) };
}

export function chooseReportPeriod(
  key: PeriodKey,
  current: ReportPeriod,
  now = new Date(),
): ReportPeriod {
  if (typeof key === "number") return { key };
  if (key === "last-month") {
    const year = now.getUTCFullYear();
    const month = now.getUTCMonth();
    return {
      key,
      startDate: dayISO(new Date(Date.UTC(year, month - 1, 1))),
      endDate: dayISO(new Date(Date.UTC(year, month, 0))),
    };
  }
  const dates =
    "startDate" in current ? current : rollingDates(current.key, now);
  return { ...dates, key };
}

export function reportPeriodLabel(period: ReportPeriod, now = new Date()) {
  const { startDate, endDate } =
    "startDate" in period ? period : rollingDates(period.key, now);
  if (!startDate || !endDate) return "Custom period";
  const start = new Date(startDate);
  const end = new Date(endDate);
  return `${formatDay(start, start.getUTCFullYear() !== end.getUTCFullYear())} – ${formatDay(end, true)}`;
}

export function reportPeriodError(period: ReportPeriod, now = new Date()) {
  if (!("startDate" in period)) return null;
  if (
    !period.startDate ||
    !period.endDate ||
    period.startDate > period.endDate
  ) {
    return "Enter valid From and To dates in order.";
  }
  if (period.endDate > dayISO(latestDate(now))) {
    return `Choose a To date on or before ${formatDay(latestDate(now), true)}.`;
  }
  const days =
    (Date.parse(period.endDate) - Date.parse(period.startDate)) / 86_400_000 +
    1;
  return days > 90 ? "Choose a date range of up to 90 days." : null;
}

export function SiteReportPeriod({
  period,
  onChange,
}: {
  period: ReportPeriod;
  onChange: (value: ReportPeriod) => void;
}) {
  const now = new Date();
  const latest = latestDate(now);
  return (
    <div className="space-y-2">
      <p className="text-sm font-medium" id="period-label">
        Period
      </p>
      <div
        role="group"
        aria-labelledby="period-label"
        className="inline-flex flex-wrap rounded-lg border border-border bg-muted/40 p-1"
      >
        {periods.map(({ key, label }) => (
          <button
            key={key}
            type="button"
            aria-pressed={period.key === key}
            onClick={() => onChange(chooseReportPeriod(key, period, now))}
            className={cn(
              "rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground",
              period.key === key && "bg-background text-foreground shadow-sm",
            )}
          >
            {label}
          </button>
        ))}
      </div>
      {"startDate" in period ? (
        <>
          <div className="grid grid-cols-2 gap-3">
            <label className="min-w-0 space-y-1 text-sm font-medium">
              <span>From</span>
              <Input
                type="date"
                max={dayISO(latest)}
                value={period.startDate}
                onChange={(event) =>
                  onChange({
                    ...period,
                    key: "custom",
                    startDate: event.target.value,
                  })
                }
                required
              />
            </label>
            <label className="min-w-0 space-y-1 text-sm font-medium">
              <span>To</span>
              <Input
                type="date"
                min={period.startDate || undefined}
                max={dayISO(latest)}
                value={period.endDate}
                onChange={(event) =>
                  onChange({
                    ...period,
                    key: "custom",
                    endDate: event.target.value,
                  })
                }
                required
              />
            </label>
          </div>
          <p className="text-xs text-muted-foreground">
            Up to 90 days. Search data is available through{" "}
            {formatDay(latest, true)}.
          </p>
        </>
      ) : (
        <p className="text-xs text-muted-foreground">
          {reportPeriodLabel(period, now)}, compared with the {period.key} days
          before.
        </p>
      )}
    </div>
  );
}
