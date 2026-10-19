export function growthCount(value: number | null) {
  return value === null
    ? "Unavailable"
    : new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(
        value,
      );
}
export function growthRate(value: number | null) {
  return value === null ? "Unavailable" : `${(value * 100).toFixed(1)}%`;
}
