import type { ReactNode } from "react";
import { Badge } from "@/client/components/ui/badge";
import { HelpLabel } from "@/client/components/HelpLabel";
import { IntentBadge } from "@/client/features/keywords/components/IntentBadge";
import type { KeywordResearchRow } from "@/types/keywords";
import {
  formatChange,
  formatCount,
  formatUpdated,
  formatUsd,
  serpFeatureName,
  trendOldestFirst,
} from "./researchModel";

function Fact({
  label,
  help,
  children,
}: {
  label: string;
  help?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt className="text-xs text-muted-foreground">
        {help ? <HelpLabel label={label} helpText={help} /> : label}
      </dt>
      <dd className="font-mono text-sm tabular-nums">{children}</dd>
    </div>
  );
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-2 rounded-lg border border-border p-3">
      <h4 className="label-mono text-muted-foreground">{title}</h4>
      <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">{children}</dl>
    </section>
  );
}

function changeTone(value: number | null | undefined) {
  if (value == null || value === 0) return undefined;
  return value > 0 ? "text-success" : "text-destructive";
}

/** The 12-month search trend as labelled bars. */
function TrendBars({ row }: { row: KeywordResearchRow }) {
  const months = trendOldestFirst(row);
  if (months.length === 0)
    return <p className="text-sm text-muted-foreground">No monthly data.</p>;
  const max = Math.max(1, ...months.map((m) => m.volume));
  return (
    <div
      className="flex h-20 items-end gap-1"
      role="img"
      aria-label={`Monthly searches: ${months
        .map((m) => `${m.label} ${m.volume}`)
        .join(", ")}`}
    >
      {months.map((m) => (
        <div
          key={m.label}
          title={`${m.label}: ${formatCount(m.volume)}`}
          className="max-w-8 flex-1 rounded-[2px] bg-muted-foreground/50"
          style={{ height: `${Math.max(4, (m.volume / max) * 100)}%` }}
        />
      ))}
    </div>
  );
}

/**
 * Every number DataForSEO returned for one keyword, grouped by the question
 * it answers. Missing numbers show a dash rather than hiding the field, so the
 * customer sees what the data has and what it does not.
 */
export function KeywordMetrics({ row }: { row: KeywordResearchRow }) {
  const features = row.serpFeatures ?? [];
  const secondary = row.secondaryIntents ?? [];
  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <section className="space-y-2 rounded-lg border border-border p-3 lg:col-span-2">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <h4 className="label-mono text-muted-foreground">
            Searches per month, last 12 months
          </h4>
          <p className="flex gap-3 font-mono text-xs tabular-nums">
            <span className={changeTone(row.monthlyChange)}>
              1 mo {formatChange(row.monthlyChange)}
            </span>
            <span className={changeTone(row.quarterlyChange)}>
              3 mo {formatChange(row.quarterlyChange)}
            </span>
            <span className={changeTone(row.yearlyChange)}>
              1 yr {formatChange(row.yearlyChange)}
            </span>
          </p>
        </div>
        <TrendBars row={row} />
      </section>

      <Group title="Demand and value">
        <Fact label="Searches / month">{formatCount(row.searchVolume)}</Fact>
        <Fact
          label="Cost per click"
          help="What advertisers pay Google per click, in US dollars."
        >
          {formatUsd(row.cpc)}
        </Fact>
        <Fact
          label="Ad bids"
          help="The low and high bids advertisers pay to show at the top of the page."
        >
          {row.topOfPageBidLow == null && row.topOfPageBidHigh == null
            ? "—"
            : `${formatUsd(row.topOfPageBidLow)}–${formatUsd(row.topOfPageBidHigh)}`}
        </Fact>
        <Fact
          label="Ad competition"
          help="How many advertisers bid on it, from 0 to 1."
        >
          {row.competition == null ? "—" : row.competition.toFixed(2)}
          {row.competitionLevel
            ? ` · ${row.competitionLevel.toLowerCase()}`
            : ""}
        </Fact>
      </Group>

      <Group title="How hard to rank">
        <Fact
          label="Difficulty"
          help="0 to 100. Higher means the pages on Google's first page are harder to beat."
        >
          {row.keywordDifficulty ?? "—"}
        </Fact>
        <Fact
          label="Top pages' site strength"
          help="Average domain rank of the sites on page one, from 0 to 1,000."
        >
          {row.topPagesDomainRank ?? "—"}
        </Fact>
        <Fact
          label="Linking sites"
          help="Average number of websites linking to each page-one result."
        >
          {formatCount(row.referringDomains)}
        </Fact>
        <Fact label="Backlinks" help="Average backlinks per page-one result.">
          {formatCount(row.backlinks)}
        </Fact>
        <Fact
          label="Dofollow backlinks"
          help="Of those backlinks, the ones that pass ranking strength."
        >
          {formatCount(row.dofollowBacklinks)}
        </Fact>
        <Fact
          label="Linking pages"
          help="Average number of pages linking to each page-one result."
        >
          {formatCount(row.referringPages)}
        </Fact>
      </Group>

      <Group title="What searchers want">
        <Fact label="Main intent">
          <IntentBadge intent={row.intent} />
        </Fact>
        <Fact label="Also">
          {secondary.length
            ? secondary
                .map((i) => i.charAt(0).toUpperCase() + i.slice(1))
                .join(", ")
            : "—"}
        </Fact>
        <Fact
          label="Shares volume with"
          help="Close variants share one Google number; this is the keyword it belongs to."
        >
          {row.coreKeyword ?? "—"}
        </Fact>
      </Group>

      <Group title="About the data">
        <Fact label="Language">{row.detectedLanguage ?? "—"}</Fact>
        <Fact label="Words">{row.wordsCount ?? "—"}</Fact>
        <Fact label="Numbers from">{formatUpdated(row.lastUpdatedTime)}</Fact>
      </Group>

      <section className="space-y-2 rounded-lg border border-border p-3 lg:col-span-2">
        <h4 className="label-mono text-muted-foreground">
          What Google shows for this search
        </h4>
        {features.length ? (
          <ul className="flex flex-wrap gap-1.5">
            {features.map((type) => (
              <li key={type}>
                <Badge variant="secondary">{serpFeatureName(type)}</Badge>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-muted-foreground">
            No result types reported.
          </p>
        )}
      </section>
    </div>
  );
}
