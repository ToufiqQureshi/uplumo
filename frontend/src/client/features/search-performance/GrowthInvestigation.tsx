import type { ReactNode } from "react";
import { ChevronDown } from "lucide-react";

export function GrowthInvestigation({
  name,
  title,
  description,
  children,
}: {
  name: string;
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <details
      name={name}
      className="group min-w-0 rounded-xl border border-border bg-card"
    >
      <summary className="flex min-h-14 cursor-pointer list-none items-center gap-3 p-4 [&::-webkit-details-marker]:hidden">
        <div className="min-w-0 flex-1">
          <span className="block font-medium">{title}</span>
          <span className="mt-1 block text-xs leading-relaxed text-muted-foreground">
            {description}
          </span>
        </div>
        <ChevronDown
          aria-hidden
          className="size-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180"
        />
      </summary>
      <div className="min-w-0 space-y-3 border-t border-border p-3 [overflow-wrap:anywhere] sm:p-4">
        {children}
      </div>
    </details>
  );
}
