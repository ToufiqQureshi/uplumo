import { useEffect, useState } from "react";
import {
  GSC_DEVICES,
  GSC_SEARCH_TYPES,
  type SearchPerformanceDevice,
  type SearchPerformanceSearch,
  type GscSearchType,
  type SEARCH_PERFORMANCE_RANGES,
} from "@/types/schemas/search-performance";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/client/components/ui/select";
import { Input } from "@/client/components/ui/input";
import { Button } from "@/client/components/ui/button";

const DEVICE_LABELS: Record<SearchPerformanceDevice, string> = {
  DESKTOP: "Desktop",
  MOBILE: "Mobile",
  TABLET: "Tablet",
};
const DEVICE_OPTIONS = GSC_DEVICES.map((value) => ({
  value,
  label: DEVICE_LABELS[value],
}));
const SEARCH_TYPE_LABELS: Record<GscSearchType, string> = {
  web: "Web",
  image: "Images",
  video: "Videos",
  news: "News",
};
const SEARCH_TYPE_OPTIONS = GSC_SEARCH_TYPES.map((value) => ({
  value,
  label: SEARCH_TYPE_LABELS[value],
}));

// Sentinel for "no filter" in the selects; never written to the URL.
const ALL = "ALL";
const DEVICE_ITEMS = [{ value: ALL, label: "All devices" }, ...DEVICE_OPTIONS];

function isDevice(value: string): value is SearchPerformanceDevice {
  return GSC_DEVICES.some((option) => option === value);
}
function isSearchType(value: string): value is GscSearchType {
  return GSC_SEARCH_TYPES.some((option) => option === value);
}

function formatDate(value: Date): string {
  return value.toISOString().slice(0, 10);
}

export function subtractUTCMonths(value: Date, months: number): Date {
  const first = new Date(
    Date.UTC(value.getUTCFullYear(), value.getUTCMonth() - months, 1),
  );
  const lastDay = new Date(
    Date.UTC(first.getUTCFullYear(), first.getUTCMonth() + 1, 0),
  ).getUTCDate();
  return new Date(
    Date.UTC(
      first.getUTCFullYear(),
      first.getUTCMonth(),
      Math.min(value.getUTCDate(), lastDay),
    ),
  );
}

/** Rolling ranges Search Console resolves itself (`range` in the URL). */
const ROLLING_PRESETS = [
  { value: "last_7_days", label: "Last 7 days" },
  { value: "last_28_days", label: "Last 28 days" },
  { value: "last_3_months", label: "Last 3 months" },
  { value: "last_6_months", label: "Last 6 months" },
  { value: "last_12_months", label: "Last 12 months" },
  { value: "last_16_months", label: "Last 16 months" },
] as const satisfies readonly {
  value: (typeof SEARCH_PERFORMANCE_RANGES)[number];
  label: string;
}[];

const MONTH_PRESETS = [
  { value: "this_month", label: "This month" },
  { value: "last_month", label: "Last month" },
] as const;
type MonthPreset = (typeof MONTH_PRESETS)[number]["value"];

/** Calendar-month dates for a month preset, by the viewer's `today` (UTC).
 *  `latest` is the newest day Search Console has data for (it lags a few
 *  days), so a range never ends after it. Null when the month has no data
 *  day yet, such as "This month" on the 1st to 3rd. */
export function monthPresetDates(
  preset: MonthPreset,
  today: Date,
  latest: Date,
): { startDate: string; endDate: string } | null {
  const offset = preset === "this_month" ? 0 : 1;
  const year = today.getUTCFullYear();
  const month = today.getUTCMonth() - offset;
  const start = new Date(Date.UTC(year, month, 1));
  const monthEnd = new Date(Date.UTC(year, month + 1, 0));
  const end = monthEnd < latest ? monthEnd : latest;
  if (end < start) return null;
  return { startDate: formatDate(start), endDate: formatDate(end) };
}

function currentPreset(
  search: SearchPerformanceSearch,
  today: Date,
  latest: Date,
): string {
  if (!search.startDate || !search.endDate) {
    return search.range ?? "last_28_days";
  }
  const month = MONTH_PRESETS.find(({ value }) => {
    const dates = monthPresetDates(value, today, latest);
    return (
      dates !== null &&
      dates.startDate === search.startDate &&
      dates.endDate === search.endDate
    );
  });
  return month?.value ?? "custom";
}

function SearchPerformanceDateRange({
  search,
  reportRange,
  onSearchChange,
}: {
  search: SearchPerformanceSearch;
  reportRange: { startDate: string; endDate: string };
  onSearchChange: (update: Partial<SearchPerformanceSearch>) => void;
}) {
  const appliedStart = search.startDate ?? reportRange.startDate;
  const appliedEnd = search.endDate ?? reportRange.endDate;
  const [startDate, setStartDate] = useState(appliedStart);
  const [endDate, setEndDate] = useState(appliedEnd);
  useEffect(() => {
    setStartDate(appliedStart);
    setEndDate(appliedEnd);
  }, [appliedStart, appliedEnd]);

  const now = new Date();
  const today = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()),
  );
  const latest = new Date(today);
  latest.setUTCDate(latest.getUTCDate() - 3);
  const earliest = subtractUTCMonths(today, 16);
  const minDate = formatDate(earliest);
  const maxDate = formatDate(latest);
  const canApply =
    startDate >= minDate &&
    startDate <= endDate &&
    endDate <= maxDate &&
    (startDate !== appliedStart || endDate !== appliedEnd);

  const presetItems = [
    ...ROLLING_PRESETS,
    ...MONTH_PRESETS.filter(
      ({ value }) => monthPresetDates(value, today, latest) !== null,
    ),
    { value: "custom", label: "Custom" },
  ];
  const preset = currentPreset(search, today, latest);

  const choosePreset = (value: string | null) => {
    const rolling = ROLLING_PRESETS.find((item) => item.value === value);
    if (rolling) {
      onSearchChange({
        page: undefined,
        range: rolling.value,
        startDate: undefined,
        endDate: undefined,
      });
      return;
    }
    const month = MONTH_PRESETS.find((item) => item.value === value);
    const dates = month ? monthPresetDates(month.value, today, latest) : null;
    if (dates) onSearchChange({ page: undefined, range: undefined, ...dates });
  };

  return (
    <div className="flex flex-wrap items-end gap-2">
      <Select items={presetItems} value={preset} onValueChange={choosePreset}>
        <SelectTrigger size="sm" className="w-40" aria-label="Date range">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {presetItems.map((item) => (
            <SelectItem
              key={item.value}
              value={item.value}
              disabled={item.value === "custom"}
            >
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <label className="grid gap-1 text-[11px] text-muted-foreground">
        From
        <Input
          type="date"
          aria-label="Start date"
          className="h-8 w-36 px-2 text-xs"
          min={minDate}
          max={endDate || maxDate}
          value={startDate}
          onChange={(event) => setStartDate(event.target.value)}
        />
      </label>
      <label className="grid gap-1 text-[11px] text-muted-foreground">
        To
        <Input
          type="date"
          aria-label="End date"
          className="h-8 w-36 px-2 text-xs"
          min={startDate || minDate}
          max={maxDate}
          value={endDate}
          onChange={(event) => setEndDate(event.target.value)}
        />
      </label>
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={!canApply}
        onClick={() =>
          onSearchChange({
            page: undefined,
            range: undefined,
            startDate,
            endDate,
          })
        }
      >
        Apply
      </Button>
      <span className="pb-1 text-xs text-muted-foreground">
        Up to 16 months; recent data can lag by 3 days.
      </span>
    </div>
  );
}

/** Search type, device, country, and calendar date filters. */
export function SearchPerformanceSelects({
  search,
  countries,
  reportRange,
  onSearchChange,
}: {
  search: SearchPerformanceSearch;
  countries: { key: string }[];
  reportRange: { startDate: string; endDate: string };
  onSearchChange: (update: Partial<SearchPerformanceSearch>) => void;
}) {
  const device = search.device ?? ALL;
  const searchType = search.searchType ?? "web";
  const country = search.country ?? ALL;
  const countryItems = [
    { value: ALL, label: "All countries" },
    ...(country !== ALL && !countries.some((row) => row.key === country)
      ? [{ value: country, label: country.toUpperCase() }]
      : []),
    ...countries.map((row) => ({
      value: row.key,
      label: row.key.toUpperCase(),
    })),
  ];
  return (
    <div className="flex flex-1 flex-wrap items-end gap-2">
      <Select
        items={SEARCH_TYPE_OPTIONS}
        value={searchType}
        onValueChange={(value) => {
          if (value != null && isSearchType(value)) {
            onSearchChange({ page: undefined, searchType: value });
          }
        }}
      >
        <SelectTrigger size="sm" className="w-32" aria-label="Search type">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {SEARCH_TYPE_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        items={DEVICE_ITEMS}
        value={device}
        onValueChange={(value) => {
          if (value == null) return;
          onSearchChange({
            page: undefined,
            device: isDevice(value) ? value : undefined,
          });
        }}
      >
        <SelectTrigger size="sm" className="w-36" aria-label="Device filter">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {DEVICE_ITEMS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        items={countryItems}
        value={country}
        onValueChange={(value) => {
          if (value == null) return;
          onSearchChange({
            page: undefined,
            country: value === ALL ? undefined : value,
          });
        }}
      >
        <SelectTrigger size="sm" className="w-36" aria-label="Country filter">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {countryItems.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <SearchPerformanceDateRange
        search={search}
        reportRange={reportRange}
        onSearchChange={onSearchChange}
      />
    </div>
  );
}
