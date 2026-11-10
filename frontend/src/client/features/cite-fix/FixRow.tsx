import { ChevronDown, ChevronRight } from "lucide-react";
import { useState } from "react";
import { cn } from "cn";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { AlsoAt } from "./AlsoAt";
import type { CiteFix } from "./citeFixApi";

const impactStyle = {
  high: "border-destructive/40 text-destructive",
  medium: "border-warning/40 text-warning",
  low: "text-muted-foreground",
} as const;

export function FixRow({ fix, position }: { fix: CiteFix; position: number }) {
  const [open, setOpen] = useState(position === 1);
  const Chevron = open ? ChevronDown : ChevronRight;
  return (
    <li className="flex flex-col gap-3 p-4">
      <button
        type="button"
        className="flex items-start gap-2 text-left"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        <Chevron aria-hidden className="mt-0.5 size-4 shrink-0" />
        <span className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
          <span className="font-medium">
            {position}. {fix.title}
          </span>
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-xs",
              impactStyle[fix.impact],
            )}
          >
            {fix.impact} impact
          </span>
          <span className="font-mono text-xs text-primary">
            +{fix.gain} points
          </span>
          <span className="font-mono text-xs text-muted-foreground">
            {fix.pagesAffected
              ? `${fix.pagesAffected} page${fix.pagesAffected === 1 ? "" : "s"}`
              : "Whole site"}
          </span>
        </span>
      </button>
      {open ? (
        <div className="flex flex-col gap-3 pl-6 text-sm">
          <p>
            <span className="font-medium">Why it matters: </span>
            <span className="text-muted-foreground">{fix.why}</span>
          </p>
          <p>
            <span className="font-medium">How to fix: </span>
            <span className="text-muted-foreground">{fix.how}</span>
          </p>
          {fix.pages.length ? (
            <div className="flex flex-col gap-1">
              <p className="label-mono text-muted-foreground">
                {fix.pagesAffected > fix.pages.length
                  ? `First ${fix.pages.length} of ${fix.pagesAffected} pages`
                  : "Pages"}
              </p>
              <ul className="flex flex-col gap-1">
                {fix.pages.map((page) => (
                  <li key={page.url} className="min-w-0">
                    <SafeExternalLink
                      url={page.url}
                      label={page.title || page.url}
                      className="inline-flex max-w-full items-center gap-1 truncate hover:underline"
                    />
                    <AlsoAt urls={page.alsoAt} />
                    {page.evidence?.map((item) => (
                      <p
                        key={item}
                        className="break-words text-muted-foreground"
                      >
                        {item}
                      </p>
                    ))}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}
