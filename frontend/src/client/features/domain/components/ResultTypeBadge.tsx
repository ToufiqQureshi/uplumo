import { Badge } from "@/client/components/ui/badge";

/**
 * Maps a DataForSEO SERP result type to a short, human label and a Badge
 * variant (theme-aware, so dark mode works). Unknown/long-tail types are shown
 * as their raw provider value rather than guessed as "Organic".
 */
const RESULT_TYPES: Record<
  string,
  { label: string; variant: "success" | "warning" | "info" | "soft" | "secondary" }
> = {
  organic: { label: "Organic", variant: "success" },
  paid: { label: "Paid", variant: "warning" },
  featured_snippet: { label: "Snippet", variant: "info" },
  local_pack: { label: "Local", variant: "soft" },
  people_also_ask: { label: "Q&A", variant: "info" },
  ai_overview: { label: "AI", variant: "info" },
  video: { label: "Video", variant: "secondary" },
  images: { label: "Images", variant: "secondary" },
  twitter: { label: "Twitter", variant: "secondary" },
  knowledge_graph: { label: "Panel", variant: "secondary" },
  related_searches: { label: "Related", variant: "secondary" },
};

export function ResultTypeBadge({ type }: { type: string | null | undefined }) {
  if (!type) return <span className="text-muted-foreground">-</span>;
  const known = RESULT_TYPES[type];
  return (
    <Badge variant={known?.variant ?? "secondary"} size="sm">
      {known?.label ?? type}
    </Badge>
  );
}
