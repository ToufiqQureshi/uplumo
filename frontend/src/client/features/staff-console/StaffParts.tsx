import type { ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { Card, CardContent } from "@/client/components/ui/card";
import { Skeleton } from "@/client/components/ui/skeleton";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import type { Count } from "./staffApi";

/** One headline number with a short label and optional context line. */
export function StatTile({
  label,
  value,
  hint,
}: {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
}) {
  return (
    <Card>
      <CardContent className="space-y-1">
        <p className="text-sm text-muted-foreground">{label}</p>
        <p className="text-2xl font-semibold tabular-nums">{value}</p>
        {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      </CardContent>
    </Card>
  );
}

export function Section({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-lg font-semibold">{title}</h2>
        {description ? (
          <p className="text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {children}
    </section>
  );
}

/** Renders a query's loading and error states, then its data. */
export function QueryView<T>({
  query,
  children,
}: {
  query: UseQueryResult<T>;
  children: (data: T) => ReactNode;
}) {
  if (query.isPending) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    );
  }
  if (query.isError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        {getStandardErrorMessage(query.error)}
      </p>
    );
  }
  return <>{children(query.data)}</>;
}

/** A ranked list with counts and each row's share of the total. */
export function CountTable({ rows, total }: { rows: Count[]; total?: number }) {
  const sum = total ?? rows.reduce((acc, r) => acc + r.count, 0);
  if (rows.length === 0) {
    return <p className="text-sm text-muted-foreground">No data yet.</p>;
  }
  return (
    <ul className="space-y-2">
      {rows.map((row) => {
        const share = sum > 0 ? Math.round((row.count / sum) * 100) : 0;
        return (
          <li key={row.label} className="space-y-1">
            <div className="flex justify-between gap-3 text-sm">
              <span className="truncate">{row.label}</span>
              <span className="shrink-0 tabular-nums text-muted-foreground">
                {row.count} · {share}%
              </span>
            </div>
            <div className="h-1.5 rounded-full bg-muted" aria-hidden>
              <div
                className="h-1.5 rounded-full bg-primary"
                style={{ width: `${share}%` }}
              />
            </div>
          </li>
        );
      })}
    </ul>
  );
}
