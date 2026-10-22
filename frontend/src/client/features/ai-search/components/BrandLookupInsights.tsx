import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { ArrowRight } from "lucide-react";
import { Badge } from "@/client/components/ui/badge";
import {
  formatCount,
  PLATFORM_SHORT_LABEL,
} from "@/client/features/ai-search/platformLabels";
import type { BrandLookupResult } from "@/types/schemas/ai-search";

type Props = {
  result: BrandLookupResult;
  projectId: string;
};

const MAX_ROWS = 8;

/**
 * The three answers a business owner needs from a lookup: where AI already
 * recommends them to people who did not search their name, which sites AI
 * reads about them (where to be listed and reviewed), and which brands it
 * names beside them. All come with the lookup; showing them buys nothing.
 */
export function BrandLookupInsights({ result, projectId }: Props) {
  const ownHost = result.resolvedTarget.toLowerCase().replace(/^www\./, "");
  const sites = result.sourceSites
    .filter((site) => !isOwnSite(site.domain, ownHost))
    .slice(0, MAX_ROWS);
  const brands = result.brandsAlongside.slice(0, MAX_ROWS);
  // Results cached before the split carry no `branded` flag: skip the section.
  const hasSplit = result.topQueries.some((q) => q.branded !== undefined);
  const unbranded = result.topQueries
    .filter((q) => q.branded === false)
    .slice(0, MAX_ROWS);

  if (!hasSplit && sites.length === 0 && brands.length === 0) return null;

  return (
    <div className="grid gap-3 px-4 pb-4 lg:grid-cols-3">
      {hasSplit ? (
        <Panel
          title="Where AI recommends you"
          lead={unbrandedLead(result)}
          tip={
            unbranded.length > 0
              ? "These people did not search your name, and AI still named you. Keep the pages it cites up to date, and write pages for similar questions."
              : "AI only mentions you when people search your name. Publish pages that answer the questions customers ask before they know you."
          }
        >
          {unbranded.map((q) => (
            <Row
              key={`${q.platform}:${q.question}`}
              label={q.question}
              badge={PLATFORM_SHORT_LABEL[q.platform]}
              value={
                q.aiSearchVolume == null
                  ? "—"
                  : `${formatCount(q.aiSearchVolume)}/mo`
              }
            />
          ))}
        </Panel>
      ) : null}
      {sites.length > 0 ? (
        <Panel
          title="Sites AI reads about you"
          lead="AI quotes these sites in its answers about you."
          tip="Make sure your listing, photos, prices and reviews on these sites are complete and current: AI repeats what they say."
        >
          {sites.map((site) => (
            <Row
              key={site.domain}
              label={site.domain}
              value={`${formatCount(site.mentions)} answers`}
            />
          ))}
        </Panel>
      ) : null}
      {brands.length > 0 ? (
        <Panel
          title="Brands AI names with you"
          lead="AI suggests these alongside you."
          tip="Add the ones you compete with as competitors to see your share of AI answers against them."
        >
          {brands.map((brand) => (
            <Row
              key={brand.name}
              label={brand.name}
              value={`${formatCount(brand.mentions)} answers`}
            />
          ))}
        </Panel>
      ) : null}
      <Link
        to="/p/$projectId/prompt-tracking"
        params={{ projectId }}
        className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline lg:col-span-3"
      >
        Track the questions that matter to you every week
        <ArrowRight className="size-4" aria-hidden />
      </Link>
    </div>
  );
}

/** "12 of 30 questions don't use your name", or a plain lead before counts existed. */
export function unbrandedLead(result: BrandLookupResult): string {
  const { brandedQuestions: branded, unbrandedQuestions: unbranded } = result;
  if (branded === undefined || unbranded === undefined) {
    return "Questions that don't use your name.";
  }
  const total = branded + unbranded;
  return `${unbranded} of ${total} question${total === 1 ? "" : "s"} where AI named you don't use your name.`;
}

function isOwnSite(domain: string, ownHost: string): boolean {
  const host = ownHost.split("/")[0] ?? "";
  return host !== "" && (domain === host || domain.endsWith(`.${host}`));
}

function Panel({
  title,
  lead,
  tip,
  children,
}: {
  title: string;
  lead: string;
  tip: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col rounded-lg border border-border">
      <div className="border-b border-border px-4 py-3">
        <h3 className="text-sm font-medium">{title}</h3>
        <p className="text-xs text-muted-foreground">{lead}</p>
      </div>
      <ul className="flex-1 divide-y divide-border">{children}</ul>
      <p className="border-t border-border bg-muted/40 px-4 py-2.5 text-xs text-muted-foreground">
        {tip}
      </p>
    </section>
  );
}

function Row({
  label,
  value,
  badge,
}: {
  label: string;
  value: string;
  badge?: string;
}) {
  return (
    <li className="flex items-center justify-between gap-2 px-4 py-2 text-sm">
      <span className="min-w-0 truncate" title={label}>
        {label}
      </span>
      <span className="inline-flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground tabular-nums">
        {badge ? <Badge variant="secondary">{badge}</Badge> : null}
        {value}
      </span>
    </li>
  );
}

/** What to do when AI never mentions the brand in the chosen country. */
export function NoMentionsAdvice({
  target,
  countryLabel,
  projectId,
}: {
  target: string;
  countryLabel: string;
  projectId: string;
}) {
  return (
    <section className="rounded-lg border border-border p-4">
      <h3 className="text-sm font-medium">
        AI answers in {countryLabel} don't mention {target} yet
      </h3>
      <p className="mt-1 text-xs text-muted-foreground">
        Here is how businesses start appearing in AI answers:
      </p>
      <ol className="mt-3 list-decimal space-y-1.5 pl-5 text-sm">
        <li>
          Complete your Google Business Profile and get recent reviews: AI
          answers about local businesses lean on them.
        </li>
        <li>
          Get listed on the sites people in your industry compare on (review
          sites, directories, marketplaces) with the same name, address and
          details everywhere.
        </li>
        <li>
          Publish pages that answer your customers' questions directly, in the
          first lines, so AI can quote them.
        </li>
      </ol>
      <Link
        to="/p/$projectId/prompt-tracking"
        params={{ projectId }}
        className="mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
      >
        Track the questions your customers ask, and see when AI starts naming
        you
        <ArrowRight className="size-4" aria-hidden />
      </Link>
    </section>
  );
}
