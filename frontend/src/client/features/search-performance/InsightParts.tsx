import { useState, type ReactNode } from "react";
import { Button } from "@/client/components/ui/button";
import { safeHttpUrl } from "@/shared/safe-url";
import { pathOf } from "./SearchVerdictCards";

/** One titled block of the Search Growth screens. */
export function InsightSection({
  id,
  title,
  note,
  children,
}: {
  id: string;
  title: string;
  /** A line of context: where the numbers come from, or what they mean. */
  note?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section
      id={id}
      aria-labelledby={`${id}-title`}
      className="space-y-3 rounded-xl border bg-card p-5"
    >
      <div className="space-y-1">
        <h2 id={`${id}-title`} className="text-lg font-semibold">
          {title}
        </h2>
        {note ? <p className="text-sm text-muted-foreground">{note}</p> : null}
      </div>
      {children}
    </section>
  );
}

/** A page's path as a link to the page, with the full address on hover. */
export function PageLink({ url }: { url: string }) {
  const href = safeHttpUrl(url);
  const label = pathOf(url);
  if (!href) return <span title={url}>{label}</span>;
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="underline-offset-4 hover:underline"
      title={url}
    >
      {label}
    </a>
  );
}

const STEP = 25;

/**
 * Every row, in steps of 25: the list is never cut to a fixed top few. The
 * server's own cap, when it applies, is stated by the caller.
 */
export function ShowMore<T>({
  items,
  children,
}: {
  items: T[];
  children: (visible: T[]) => ReactNode;
}) {
  const [shown, setShown] = useState(STEP);
  const left = items.length - shown;
  return (
    <>
      {children(items.slice(0, shown))}
      {left > 0 ? (
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShown((current) => current + STEP)}
        >
          Show {Math.min(STEP, left)} more ({left} left)
        </Button>
      ) : null}
    </>
  );
}

/** A table that scrolls sideways on a phone instead of squeezing. */
export function InsightTable({
  columns,
  children,
}: {
  columns: { label: string; align?: "right" }[];
  children: ReactNode;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr>
            {columns.map((column) => (
              <th
                key={column.label}
                className={`py-2 pr-3 font-medium ${column.align === "right" ? "text-right" : ""}`}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  );
}
