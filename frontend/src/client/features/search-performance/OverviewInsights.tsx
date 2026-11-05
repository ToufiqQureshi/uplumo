import { useState, type ReactNode } from "react";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import type { SearchGrowthView } from "@/types/schemas/search-performance";
import type { SearchInsights, SegmentKind } from "./insightsApi";
import { InsightSection, InsightTable, PageLink } from "./InsightParts";
import {
  formatCount,
  formatCtr,
  formatPosition,
} from "./SearchPerformanceColumns";

type Action = SearchInsights["actions"][number];

const ACTION_VIEW: Record<Action["kind"], SearchGrowthView> = {
  losing_page: "fix",
  decaying_page: "fix",
  ctr_gap: "grow",
  cannibalization: "grow",
};

function actionCopy(action: Action): { title: ReactNode; why: string } {
  const clicks = formatCount(action.impactClicks);
  switch (action.kind) {
    case "losing_page":
      return {
        title: <PageLink url={action.subject} />,
        why: `Lost ${clicks} clicks compared with the previous period.`,
      };
    case "decaying_page":
      return {
        title: <PageLink url={action.subject} />,
        why: `Has lost clicks two periods in a row, ${clicks} in total.`,
      };
    case "ctr_gap":
      return {
        title: <>&ldquo;{action.subject}&rdquo;</>,
        why: `Gets fewer clicks than your own pages usually get at this position. About ${clicks} more clicks are possible, estimated from your own numbers.`,
      };
    case "cannibalization":
      return {
        title: <>&ldquo;{action.subject}&rdquo;</>,
        why: `Several of your pages compete for this search. ${clicks} clicks go through it, split between them.`,
      };
  }
}

/**
 * The few things worth doing first, picked from the site's own numbers:
 * clicks really lost come before estimates. Hidden when there is nothing.
 */
export function DoFirst({
  actions,
  onOpen,
}: {
  actions: Action[];
  onOpen: (view: SearchGrowthView) => void;
}) {
  if (actions.length === 0) return null;
  return (
    <InsightSection
      id="do-first"
      title="Do these first"
      note="Real lost clicks come first, then estimates based on your own pages."
    >
      <ol className="grid gap-3 md:grid-cols-3">
        {actions.map((action) => {
          const copy = actionCopy(action);
          return (
            <li
              key={`${action.kind}:${action.subject}`}
              className="flex flex-col gap-2 rounded-lg border p-4"
            >
              <p className="min-w-0 break-words font-medium">{copy.title}</p>
              <p className="text-sm text-muted-foreground">{copy.why}</p>
              <Button
                size="sm"
                variant="outline"
                className="mt-auto self-start"
                onClick={() => onOpen(ACTION_VIEW[action.kind])}
              >
                See the details
              </Button>
            </li>
          );
        })}
      </ol>
    </InsightSection>
  );
}

const SEGMENT_LABEL: Record<SegmentKind, string> = {
  all: "All named searches",
  branded: "Branded",
  non_branded: "Non-branded",
  long_tail: "Long-tail (4+ words)",
};

/** Branded or not decides whether growth is real, so the owner names the brand. */
function BrandNamesForm({
  text,
  onSave,
}: {
  text: string;
  onSave: (text: string) => void;
}) {
  const [draft, setDraft] = useState(text);
  return (
    <form
      className="flex flex-col gap-2 sm:flex-row sm:items-center"
      onSubmit={(event) => {
        event.preventDefault();
        onSave(draft);
      }}
    >
      <Input
        className="sm:max-w-sm"
        aria-label="Your brand names"
        placeholder="Your brand names, separated by commas"
        value={draft}
        maxLength={400}
        onChange={(event) => setDraft(event.target.value)}
      />
      <Button size="sm" type="submit" disabled={draft === text}>
        Split branded searches
      </Button>
    </form>
  );
}

export function SegmentsCard({
  insights,
  brandText,
  onSaveBrands,
}: {
  insights: SearchInsights;
  brandText: string;
  onSaveBrands: (text: string) => void;
}) {
  const { segments, coverage } = insights;
  if (segments.every((segment) => segment.queries === 0)) return null;
  return (
    <InsightSection
      id="segments"
      title="Where your search traffic comes from"
      note={
        coverage.brandTermsUsed
          ? "Branded searches are people who already know you. Growth in non-branded searches is new people finding you."
          : "Tell us your brand names to see how much traffic is people who already know you."
      }
    >
      <BrandNamesForm key={brandText} text={brandText} onSave={onSaveBrands} />
      <InsightTable
        columns={[
          { label: "Searches" },
          { label: "Queries", align: "right" },
          { label: "Clicks", align: "right" },
          { label: "Shown", align: "right" },
          { label: "CTR", align: "right" },
          { label: "Avg. rank", align: "right" },
        ]}
      >
        {segments.map((segment) => (
          <tr key={segment.kind} className="border-t">
            <td className="py-2 pr-3">{SEGMENT_LABEL[segment.kind]}</td>
            <td className="py-2 pr-3 text-right tabular-nums">
              {formatCount(segment.queries)}
            </td>
            <td className="py-2 pr-3 text-right tabular-nums">
              {formatCount(segment.clicks)}
            </td>
            <td className="py-2 pr-3 text-right tabular-nums">
              {formatCount(segment.impressions)}
            </td>
            <td className="py-2 pr-3 text-right tabular-nums">
              {formatCtr(segment.ctr)}
            </td>
            <td className="py-2 pr-3 text-right tabular-nums">
              {formatPosition(segment.position)}
            </td>
          </tr>
        ))}
      </InsightTable>
      <p className="text-xs text-muted-foreground">
        Google leaves out rare searches, so these rows add up to less than your
        total clicks.
        {coverage.truncated
          ? ` Only the first ${formatCount(coverage.queryPageRows)} query and page pairs were read; narrow the date range or add a filter for the rest.`
          : ""}
      </p>
    </InsightSection>
  );
}
