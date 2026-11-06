import { queryOptions } from "@tanstack/react-query";
import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";
import type { FilterInput } from "@/client/features/search-performance/searchPerformanceQueries";

const change = z.object({
  clicks: z.number(),
  prevClicks: z.number(),
  impressions: z.number(),
  prevImpressions: z.number(),
  position: z.number(),
  prevPosition: z.number(),
  ctr: z.number(),
  prevCtr: z.number(),
});

const loss = change.extend({
  reason: z.enum(["rank_fell", "fewer_searches", "ctr_fell", "mixed"]),
  severity: z.enum(["urgent", "watch", "minor"]),
});

const segmentKinds = ["all", "branded", "non_branded", "long_tail"] as const;
export type SegmentKind = (typeof segmentKinds)[number];

const insightsSchema = z.discriminatedUnion("connected", [
  z.object({ connected: z.literal(false) }),
  z.object({
    connected: z.literal(true),
    range: z.object({
      startDate: z.string(),
      endDate: z.string(),
      prevStartDate: z.string(),
      prevEndDate: z.string(),
    }),
    coverage: z.object({
      queryPageRows: z.number(),
      truncated: z.boolean(),
      decayAvailable: z.boolean(),
      comparisonAvailable: z.boolean(),
      brandTermsUsed: z.boolean(),
    }),
    segments: z.array(
      z.object({
        kind: z.enum(segmentKinds),
        queries: z.number(),
        clicks: z.number(),
        impressions: z.number(),
        ctr: z.number(),
        position: z.number(),
      }),
    ),
    actions: z.array(
      z.object({
        kind: z.enum([
          "losing_page",
          "decaying_page",
          "ctr_gap",
          "cannibalization",
        ]),
        subject: z.string(),
        page: z.string().optional(),
        impactClicks: z.number(),
        basis: z.enum(["measured", "estimated", "contested"]),
      }),
    ),
    losingPages: z.array(loss.extend({ page: z.string() })),
    queryLosses: z.array(loss.extend({ query: z.string() })),
    queryGains: z.array(change.extend({ query: z.string() })),
    decaying: z.array(
      z.object({
        page: z.string(),
        clicks: z.tuple([z.number(), z.number(), z.number()]),
        lostClicks: z.number(),
      }),
    ),
    ctrGaps: z.array(
      z.object({
        query: z.string(),
        page: z.string(),
        clicks: z.number(),
        impressions: z.number(),
        ctr: z.number(),
        position: z.number(),
        expectedCtr: z.number(),
        possibleClicks: z.number(),
      }),
    ),
    cannibalization: z.array(
      z.object({
        query: z.string(),
        clicks: z.number(),
        impressions: z.number(),
        leader: z.string(),
        pages: z.array(
          z.object({
            page: z.string(),
            clicks: z.number(),
            impressions: z.number(),
            position: z.number(),
            share: z.number(),
          }),
        ),
      }),
    ),
    // How many each list held before the server cut it to its cap.
    totals: z.object({
      decaying: z.number(),
      ctrGaps: z.number(),
      cannibalization: z.number(),
    }),
  }),
]);

export type InsightsResult = z.infer<typeof insightsSchema>;

export type SearchInsights = Extract<
  z.infer<typeof insightsSchema>,
  { connected: true }
>;

function getSearchInsights(
  projectId: string,
  filters: FilterInput,
  brandTerms: string[],
) {
  return apiRequest(
    `/api/v1/projects/${encodeURIComponent(projectId)}/gsc/search-performance/insights`,
    insightsSchema,
    "POST",
    { ...filters, brandTerms },
  );
}

/** Free Search Console reads only. They are kept for five minutes, so moving
 *  between screens does not read Google again. While new numbers load, the
 *  old ones stay (dimmed) only for the same project: another project's actions
 *  and pages must never show under this one. */
export function insightsQueryOptions(
  projectId: string,
  filters: FilterInput,
  brandTerms: string[],
) {
  return queryOptions({
    queryKey: ["searchInsights", projectId, filters, brandTerms],
    queryFn: () => getSearchInsights(projectId, filters, brandTerms),
    staleTime: 5 * 60_000,
    placeholderData: keepForProject(projectId),
  });
}

/** Old numbers may stand in while new ones load only if they are this
 *  project's: the project is the second part of the query key. */
export function keepForProject(projectId: string) {
  return <T>(
    previous: T | undefined,
    previousQuery: { queryKey: readonly unknown[] } | undefined,
  ) => (previousQuery?.queryKey[1] === projectId ? previous : undefined);
}
