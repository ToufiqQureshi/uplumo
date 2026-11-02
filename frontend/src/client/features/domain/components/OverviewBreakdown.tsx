import { formatRounded } from "@/client/features/domain/utils";

type Bands = {
  top1: number | null;
  top2To3: number | null;
  top4To10: number | null;
  top11To20: number | null;
};

type Movement = {
  new: number | null;
  up: number | null;
  down: number | null;
  lost: number | null;
};

/**
 * Where the domain's organic keywords rank and how they moved since
 * DataForSEO's previous update. Both come with the overview the customer
 * already paid for; missing values stay "-", never 0.
 */
export function OverviewBreakdown({
  positions,
  movement,
}: {
  positions?: Bands | null;
  movement?: Movement | null;
}) {
  if (!positions && !movement) return null;
  return (
    <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
      {positions ? (
        <Row
          title="Google position"
          items={[
            ["#1", positions.top1],
            ["#2–3", positions.top2To3],
            ["#4–10", positions.top4To10],
            ["#11–20", positions.top11To20],
          ]}
        />
      ) : null}
      {movement ? (
        <Row
          title="Since last update"
          items={[
            ["New", movement.new],
            ["Up", movement.up],
            ["Down", movement.down],
            ["Lost", movement.lost],
          ]}
        />
      ) : null}
    </div>
  );
}

function Row({
  title,
  items,
}: {
  title: string;
  items: [string, number | null][];
}) {
  return (
    <div className="rounded-lg border border-border p-3">
      <p className="label-mono text-muted-foreground">{title}</p>
      <dl className="mt-2 grid grid-cols-4 gap-2">
        {items.map(([label, value]) => (
          <div key={label} className="min-w-0">
            <dt className="text-xs text-muted-foreground">{label}</dt>
            <dd className="font-mono text-lg tabular-nums">
              {formatRounded(value)}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
