import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { z } from "zod";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Label } from "@/client/components/ui/label";
import { Switch } from "@/client/components/ui/switch";
import { apiRequest } from "@/client/lib/uplumoApi";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { formatCredits } from "@/client/features/billing/credits/format";
import { useBillingUsage } from "@/client/features/billing/usageApi";
import { AiAction, InsightCard } from "./AiAction";
import { insightsApi, nextActionsKey } from "./insightsApi";
import { siteKind } from "./backlinkGap";
import { hashKey } from "./nextActions";
import { useLatestAnswers } from "./useLatestAnswers";

const gapSchema = z.object({
  competitors: z.array(z.string()),
  rows: z.array(
    z.object({
      domain: z.string(),
      rank: z.number(),
      links: z.array(z.number()),
    }),
  ),
  fetchedAt: z.string(),
});

/**
 * Sites that link to competitors but not to the customer. One click buys one
 * backlink lookup of 100 sites, cached for 7 days; outreach emails are a
 * separate AI click; "Contacted" is saved for the team for free.
 */
export function BacklinkGapPage({ projectId }: { projectId: string }) {
  const client = useQueryClient();
  const ai = useLatestAnswers(projectId);
  const usage = useBillingUsage();
  const ownDefault = ai.configuration?.domain ?? "";
  const suggested = (ai.configuration?.competitors ?? [])
    .map((c) => c.domain)
    .slice(0, 3);
  const [own, setOwn] = useState("");
  const [competitorsText, setCompetitorsText] = useState("");
  const [all, setAll] = useState(false);
  const [open, setOpen] = useState<string | null>(null);
  const ownDomain = (own || ownDefault).trim();
  const competitors = (competitorsText || suggested.join(", "))
    .split(/[\s,]+/)
    .map((c) => c.trim())
    .filter(Boolean);
  const cost = usage.data?.costs.find(
    (c) => c.feature === "backlinks" && c.action === "backlink_gap",
  );
  const lookup = useMutation({
    retry: false,
    mutationFn: () =>
      apiRequest(
        `/api/v1/projects/${encodeURIComponent(projectId)}/backlinks/gap`,
        gapSchema,
        "POST",
        { own: ownDomain, competitors, all },
      ),
  });
  const states = useQuery({
    queryKey: nextActionsKey(projectId),
    queryFn: () => insightsApi.actionStates(projectId),
  });
  const contacted = new Set(
    states.data?.states.filter((s) => s.status === "done").map((s) => s.key),
  );
  const mark = useMutation({
    mutationFn: ({ key, on }: { key: string; on: boolean }) =>
      insightsApi.setActionStatus(projectId, key, on ? "done" : "todo"),
    onSettled: () =>
      client.invalidateQueries({ queryKey: nextActionsKey(projectId) }),
  });

  const data = lookup.data;
  const rows = data?.rows ?? [];
  const everyone = rows.filter((r) => r.links.every((l) => l > 0));
  const easy = rows.filter((r) => siteKind(r.domain) === "Directory");
  const first = [...everyone, ...rows].find(
    (r) => siteKind(r.domain) === "Directory",
  );

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="Backlink Opportunity"
        description="Sites that link to your competitors but not to you, and how to get each link."
      />
      <InsightCard
        stamp={
          data
            ? `Backlink index · ${new Date(data.fetchedAt).toLocaleDateString("en-IN")}`
            : "Backlink index"
        }
        headline={
          data ? (
            rows.length ? (
              <>
                <span className="text-primary">{everyone.length}</span> site
                {everyone.length === 1 ? "" : "s"} link to every competitor and
                not to you.
                {first
                  ? ` Start with ${first.domain}: directories usually take a free listing.`
                  : ""}
              </>
            ) : (
              "No site links to these competitors without also linking to you. Try other competitors."
            )
          ) : (
            "Enter up to 3 competitors. We find the sites that link to them and not to you."
          )
        }
        evidence={
          data
            ? [
                {
                  source: "Sites found",
                  text: `${rows.length} (top 100 by authority)`,
                },
                {
                  source: "Link to every competitor",
                  text: `${everyone.length}`,
                },
                { source: "Directories (easiest)", text: `${easy.length}` },
              ]
            : []
        }
      >
        <form
          className="grid gap-3 md:grid-cols-[1fr_2fr_auto] md:items-end"
          onSubmit={(e) => {
            e.preventDefault();
            if (ownDomain && competitors.length) lookup.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="bg-own">Your domain</Label>
            <Input
              id="bg-own"
              value={own || ownDefault}
              placeholder="yoursite.com"
              onChange={(e) => setOwn(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="bg-comp">
              Competitors (up to 3, comma separated)
            </Label>
            <Input
              id="bg-comp"
              value={competitorsText || suggested.join(", ")}
              placeholder="rival1.com, rival2.com"
              onChange={(e) => setCompetitorsText(e.target.value)}
            />
          </div>
          <Button
            type="submit"
            disabled={
              lookup.isPending ||
              !ownDomain ||
              competitors.length === 0 ||
              competitors.length > 3
            }
          >
            {lookup.isPending ? "Finding sites…" : "Find sites"}
          </Button>
        </form>
        <div className="flex flex-wrap items-center gap-4">
          <label className="flex items-center gap-2 text-sm">
            <Switch checked={all} onCheckedChange={setAll} />
            Only sites that link to every competitor
          </label>
          <span className="font-mono text-xs text-muted-foreground">
            1 lookup
            {cost ? ` · about ${formatCredits(cost.credits)} credits` : ""};
            free again for 7 days
          </span>
        </div>
        {lookup.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {getStandardErrorMessage(lookup.error)}
          </p>
        ) : null}
      </InsightCard>

      {rows.length ? (
        <section className="overflow-x-auto rounded-xl border bg-card">
          <table className="w-full min-w-[760px] text-sm">
            <thead className="font-mono text-xs text-muted-foreground">
              <tr className="border-b">
                <th className="p-3 text-left font-normal">Site</th>
                <th className="p-3 text-left font-normal">Authority</th>
                <th className="p-3 text-left font-normal">Type</th>
                {data?.competitors.map((c) => (
                  <th key={c} className="p-3 text-center font-normal">
                    {c}
                  </th>
                ))}
                <th className="p-3 text-left font-normal">Outreach</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const key = `outreach:${hashKey(r.domain)}`;
                const isContacted = contacted.has(key);
                const isOpen = open === r.domain;
                const kind = siteKind(r.domain);
                return (
                  <tr key={r.domain} className="border-b align-top">
                    <td className="p-3">
                      <button
                        type="button"
                        className="text-left font-medium"
                        aria-expanded={isOpen}
                        onClick={() => setOpen(isOpen ? null : r.domain)}
                      >
                        {r.domain}
                      </button>
                      {isOpen ? (
                        <div className="mt-3 flex max-w-xl flex-col gap-3">
                          <p className="text-muted-foreground">
                            {kind === "Directory"
                              ? "A directory: sign up and add your business with the same name, address and phone as your site."
                              : "Pitch one concrete reason this site's readers would care, and keep it short."}
                          </p>
                          <AiAction
                            projectId={projectId}
                            label={
                              kind === "Directory"
                                ? "Write my listing text"
                                : "Write the outreach email"
                            }
                            input={{
                              action:
                                kind === "Directory"
                                  ? "business_post"
                                  : "outreach_email",
                              brief: `Site: ${r.domain} (authority ${Math.round(r.rank)}), a ${kind.toLowerCase()} that links to ${data?.competitors
                                .filter((_, i) => r.links[i] > 0)
                                .join(
                                  ", ",
                                )} but not to ${ownDomain}. Our business: ${ai.configuration?.brand ?? ownDomain}.`,
                            }}
                          />
                        </div>
                      ) : null}
                    </td>
                    <td className="p-3 font-mono">{Math.round(r.rank)}</td>
                    <td className="p-3 text-muted-foreground">{kind}</td>
                    {r.links.map((l, i) => (
                      <td
                        key={data?.competitors[i]}
                        className={`p-3 text-center ${l ? "font-semibold text-primary" : "text-muted-foreground"}`}
                      >
                        {l ? "✓" : "—"}
                      </td>
                    ))}
                    <td className="p-3">
                      <Button
                        size="sm"
                        variant={isContacted ? "default" : "outline"}
                        aria-pressed={isContacted}
                        onClick={() => mark.mutate({ key, on: !isContacted })}
                      >
                        {isContacted ? "Contacted" : "To contact"}
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </section>
      ) : null}
    </div>
  );
}
