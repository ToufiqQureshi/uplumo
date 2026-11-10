/**
 * The other crawled addresses of one page (the same path with ?from=home or
 * a #fragment). The server scores them as one page; this says so.
 */
export function AlsoAt({ urls }: { urls: readonly string[] | undefined }) {
  if (!urls?.length) return null;
  return (
    <p className="text-xs text-muted-foreground" title={urls.join("\n")}>
      Same page also at {urls.length} other address
      {urls.length === 1 ? "" : "es"}
    </p>
  );
}
