export function GrowthMobileResults({
  label,
  rows,
}: {
  label: string;
  rows: {
    key: string;
    title: string;
    subtitle?: string;
    metrics: { label: string; value: string }[];
  }[];
}) {
  return (
    <div aria-label={label} className="space-y-3 sm:hidden">
      {rows.map((row) => (
        <article
          key={row.key}
          className="min-w-0 rounded-xl border border-border bg-muted/20 p-3"
        >
          <h5 className="break-words text-sm font-medium [overflow-wrap:anywhere]">
            {row.title}
          </h5>
          {row.subtitle ? (
            <p className="mt-1 break-words text-xs text-muted-foreground">
              {row.subtitle}
            </p>
          ) : null}
          <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-3">
            {row.metrics.map((metric) => (
              <div key={metric.label} className="min-w-0">
                <dt className="text-xs leading-relaxed text-muted-foreground">
                  {metric.label}
                </dt>
                <dd className="mt-1 break-words text-sm font-medium tabular-nums">
                  {metric.value}
                </dd>
              </div>
            ))}
          </dl>
        </article>
      ))}
    </div>
  );
}
