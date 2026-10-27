import { useMutation } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Label } from "@/client/components/ui/label";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { formatCredits } from "@/client/features/billing/credits/format";
import { useBillingUsage } from "@/client/features/billing/usageApi";
import {
  getDomainKeywordSuggestions,
  getDomainKeywordsPage,
} from "@/client/features/domain/domainApi";
import { AiAction, InsightCard } from "./AiAction";
import { contentIdeas, type IdeaPage } from "./contentIdeas";
import { useLatestAnswers } from "./useLatestAnswers";

const num = new Intl.NumberFormat("en-IN");

/**
 * A competitor's pages that bring it search traffic, compared with the
 * customer's own rankings. One click buys two domain report lookups (the
 * competitor's top 100 keywords with their pages, and the customer's own top
 * 100), each cached like Domain Overview; the AI brief is a separate click.
 */
export function ContentIdeasPage({ projectId }: { projectId: string }) {
  const ai = useLatestAnswers(projectId);
  const usage = useBillingUsage();
  const ownDomain = ai.configuration?.domain ?? "";
  const suggested = ai.configuration?.competitors.map((c) => c.domain) ?? [];
  const [competitor, setCompetitor] = useState(suggested[0] ?? "");
  const [own, setOwn] = useState(ownDomain);
  const [tab, setTab] = useState<IdeaPage["kind"]>("gap");
  const [open, setOpen] = useState<string | null>(null);
  const lookupCredits =
    usage.data?.costs.find((c) => c.feature === "domain_research")?.credits ??
    0;
  const analyze = useMutation({
    retry: false,
    mutationFn: async () => {
      const [theirs, mine] = await Promise.all([
        getDomainKeywordsPage({
          data: {
            projectId,
            domain: competitor.trim(),
            page: 1,
            pageSize: 100,
            sortMode: "traffic",
            sortOrder: "desc",
            filters: {},
          },
        }),
        getDomainKeywordSuggestions({
          data: { projectId, domain: (own || ownDomain).trim() },
        }),
      ]);
      return {
        domain: theirs.domain,
        pages: contentIdeas(theirs.keywords, mine),
      };
    },
  });
  const pages = analyze.data?.pages ?? [];
  const list = pages.filter((p) => p.kind === tab);
  const gapTraffic = pages
    .filter((p) => p.kind === "gap")
    .reduce((s, p) => s + p.traffic, 0);
  const top = pages.find((p) => p.kind === "gap");

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="Content Ideas"
        description="The pages bringing your competitor search traffic, and which ones you are missing."
      />
      <InsightCard
        stamp="Estimated traffic · DataForSEO Labs · your project's market"
        headline={
          analyze.data ? (
            pages.length ? (
              <>
                {analyze.data.domain} gets about{" "}
                <span className="text-primary">
                  {num.format(gapTraffic)} visits a month
                </span>{" "}
                from pages you don&rsquo;t have.
                {top ? ` Start with ${top.path}.` : ""}
              </>
            ) : (
              "You already rank in the top 10 for every page this competitor gets traffic from."
            )
          ) : (
            "Enter a competitor. We compare its top pages with what you rank for and show what to write next."
          )
        }
        evidence={
          analyze.data
            ? [
                {
                  source: "Pages you don't have",
                  text: `${pages.filter((p) => p.kind === "gap").length} pages`,
                },
                {
                  source: "Pages where you rank 11-30",
                  text: `${pages.filter((p) => p.kind === "improve").length} pages`,
                },
              ]
            : []
        }
      >
        <form
          className="grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end"
          onSubmit={(e) => {
            e.preventDefault();
            if (competitor.trim() && (own || ownDomain).trim())
              analyze.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ci-competitor">Competitor domain</Label>
            <Input
              id="ci-competitor"
              value={competitor}
              placeholder="competitor.com"
              list="ci-suggested"
              onChange={(e) => setCompetitor(e.target.value)}
            />
            <datalist id="ci-suggested">
              {suggested.map((d) => (
                <option key={d} value={d} />
              ))}
            </datalist>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ci-own">Your domain</Label>
            <Input
              id="ci-own"
              value={own || ownDomain}
              placeholder="yoursite.com"
              onChange={(e) => setOwn(e.target.value)}
            />
          </div>
          <Button type="submit" disabled={analyze.isPending}>
            {analyze.isPending ? "Comparing…" : "Compare"}
          </Button>
        </form>
        <p className="font-mono text-xs text-muted-foreground">
          Uses 2 domain lookups
          {lookupCredits
            ? ` (about ${formatCredits(lookupCredits * 2)} credits)`
            : ""}
          , free again for the same domains while cached.
        </p>
        {analyze.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {getStandardErrorMessage(analyze.error)}
          </p>
        ) : null}
      </InsightCard>

      {analyze.data && pages.length ? (
        <>
          <div role="tablist" aria-label="View" className="flex gap-2">
            {(
              [
                ["gap", "They have it, you don't"],
                ["improve", "You rank 11-30"],
              ] as const
            ).map(([id, label]) => (
              <Button
                key={id}
                role="tab"
                aria-selected={tab === id}
                variant={tab === id ? "default" : "outline"}
                onClick={() => setTab(id)}
              >
                {label}
              </Button>
            ))}
          </div>
          <ol className="flex flex-col divide-y rounded-xl border bg-card">
            {list.length === 0 ? (
              <li className="p-8 text-center text-sm text-muted-foreground">
                Nothing in this view.
              </li>
            ) : null}
            {list.map((p) => {
              const isOpen = open === p.url;
              return (
                <li key={p.url} className="flex flex-col gap-3 p-4">
                  <button
                    type="button"
                    aria-expanded={isOpen}
                    onClick={() => setOpen(isOpen ? null : p.url)}
                    className="flex flex-wrap items-center justify-between gap-2 text-left"
                  >
                    <span className="min-w-0 truncate font-medium">
                      {p.path}
                    </span>
                    <span className="flex gap-2 text-xs">
                      <span className="rounded-full border px-2 py-0.5">
                        ≈ {num.format(p.traffic)} visits/mo
                      </span>
                      <span className="rounded-full border px-2 py-0.5 text-muted-foreground">
                        {p.keywords.length} keywords
                      </span>
                    </span>
                  </button>
                  {isOpen ? (
                    <>
                      <div className="overflow-x-auto rounded-lg bg-muted p-3">
                        <table className="w-full min-w-[420px] text-sm">
                          <thead className="font-mono text-xs text-muted-foreground">
                            <tr>
                              <th className="text-left font-normal">Keyword</th>
                              <th className="text-right font-normal">
                                Searches/mo
                              </th>
                              <th className="text-right font-normal">Them</th>
                              <th className="text-right font-normal">You</th>
                            </tr>
                          </thead>
                          <tbody>
                            {p.keywords.slice(0, 8).map((k) => (
                              <tr key={k.keyword} className="border-t">
                                <td className="py-1.5">{k.keyword}</td>
                                <td className="text-right font-mono">
                                  {num.format(k.volume)}
                                </td>
                                <td className="text-right font-mono text-destructive">
                                  {k.theirs ? `#${k.theirs}` : "—"}
                                </td>
                                <td className="text-right font-mono text-muted-foreground">
                                  {k.yours ? `#${k.yours}` : "—"}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      <AiAction
                        projectId={projectId}
                        label="Write a brief to beat this page"
                        input={{
                          action: "outline",
                          title: p.keywords[0]?.keyword ?? p.path,
                          targetKeyword: p.keywords[0]?.keyword ?? "",
                          brief: `Competitor page ${p.url} gets about ${p.traffic} visits a month. Its keywords: ${p.keywords
                            .slice(0, 15)
                            .map(
                              (k) =>
                                `${k.keyword} (${k.volume}/mo, them #${k.theirs ?? "-"}, us ${k.yours ? `#${k.yours}` : "not ranking"})`,
                            )
                            .join("; ")}. Cover every keyword, answer first.`,
                        }}
                      >
                        {() => (
                          <Button
                            size="xs"
                            variant="outline"
                            nativeButton={false}
                            render={
                              <Link
                                to="/p/$projectId/content-editor"
                                params={{ projectId }}
                              />
                            }
                          >
                            Open Content editor
                          </Button>
                        )}
                      </AiAction>
                    </>
                  ) : null}
                </li>
              );
            })}
          </ol>
        </>
      ) : null}
    </div>
  );
}
