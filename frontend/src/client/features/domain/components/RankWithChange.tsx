/**
 * A keyword's rank with its change since DataForSEO's previous update, which
 * comes with the paid row. Lower is better, so a smaller rank is "up".
 */
export function RankWithChange({
  position,
  previousPosition,
  isNew,
}: {
  position: number | null;
  previousPosition?: number | null;
  isNew?: boolean | null;
}) {
  if (position == null) return <span>-</span>;
  return (
    <span className="inline-flex items-baseline gap-1.5 tabular-nums">
      {position}
      <Change
        position={position}
        previousPosition={previousPosition}
        isNew={isNew}
      />
    </span>
  );
}

function Change({
  position,
  previousPosition,
  isNew,
}: {
  position: number;
  previousPosition?: number | null;
  isNew?: boolean | null;
}) {
  if (isNew) return <span className="text-xs text-info">New</span>;
  if (previousPosition == null || previousPosition === position) return null;
  const gained = previousPosition - position;
  return gained > 0 ? (
    <span className="text-xs text-success" title={`Was ${previousPosition}`}>
      ↑{gained}
    </span>
  ) : (
    <span
      className="text-xs text-destructive"
      title={`Was ${previousPosition}`}
    >
      ↓{-gained}
    </span>
  );
}
