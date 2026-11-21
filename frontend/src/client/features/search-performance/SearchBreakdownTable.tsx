import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import {
  formatCount,
  formatCtr,
  formatPosition,
  type Report,
} from "@/client/features/search-performance/SearchPerformanceColumns";
import { countryName } from "@/client/lib/countryNames";

type BreakdownRow = Report["countries"][number];
type Breakdown = "country" | "device";

const DEVICE_LABEL: Record<string, string> = {
  DESKTOP: "Desktop",
  MOBILE: "Mobile",
  TABLET: "Tablet",
};

function rowLabel(breakdown: Breakdown, key: string): string {
  return breakdown === "country"
    ? countryName(key)
    : (DEVICE_LABEL[key.toUpperCase()] ?? key);
}

/** Clicks by country or device for the period, with each row's share of all
 *  clicks. Choosing a row filters the whole page to it. */
export function SearchBreakdownTable({
  breakdown,
  rows,
  totalClicks,
  onSelect,
}: {
  breakdown: Breakdown;
  rows: BreakdownRow[];
  totalClicks: number;
  onSelect: (key: string) => void;
}) {
  if (rows.length === 0) {
    return (
      <p className="rounded-xl border px-4 py-6 text-sm text-muted-foreground">
        Google has no {breakdown === "country" ? "country" : "device"} data for
        this period yet.
      </p>
    );
  }
  return (
    <div className="overflow-hidden rounded-xl border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>
              {breakdown === "country" ? "Country" : "Device"}
            </TableHead>
            <TableHead className="text-right">Clicks</TableHead>
            <TableHead className="text-right">Share of clicks</TableHead>
            <TableHead className="text-right">Impressions</TableHead>
            <TableHead className="text-right">CTR</TableHead>
            <TableHead className="text-right">Position</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.key}>
              <TableCell>
                <button
                  type="button"
                  className="text-left font-medium underline-offset-4 hover:underline"
                  title="Show only this in every number and table"
                  onClick={() => onSelect(row.key)}
                >
                  {rowLabel(breakdown, row.key)}
                </button>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatCount(row.clicks)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {totalClicks > 0
                  ? `${Math.round((row.clicks / totalClicks) * 100)}%`
                  : "—"}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatCount(row.impressions)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatCtr(row.ctr)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {formatPosition(row.position)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
