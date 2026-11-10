import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
  type UseMutationResult,
} from "@tanstack/react-query";
import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const placeSchema = z.object({
  rank: z.number().int(),
  cid: z.string(),
  placeId: z.string(),
  title: z.string(),
  address: z.string(),
  category: z.string(),
  categories: z.array(z.string()),
  rating: z.number().nullable(),
  reviews: z.number().int(),
  latitude: z.number(),
  longitude: z.number(),
  domain: z.string(),
});
export type Place = z.infer<typeof placeSchema>;

const businessSchema = z.object({
  cid: z.string(),
  placeId: z.string(),
  title: z.string(),
  address: z.string(),
  category: z.string(),
  latitude: z.number(),
  longitude: z.number(),
  searchTerm: z.string(),
  languageCode: z.string(),
  updatedAt: z.string(),
});
export type Business = z.infer<typeof businessSchema>;

const profileSchema = z.object({
  cid: z.string(),
  title: z.string(),
  categories: z.array(z.string()),
  hasDescription: z.boolean(),
  hasPhone: z.boolean(),
  hasWebsite: z.boolean(),
  hasHours: z.boolean(),
  claimed: z.boolean(),
  photos: z.number().int(),
  rating: z.number().nullable(),
  reviews: z.number().int(),
  attributes: z.array(z.string()),
});

const contenderSchema = z.object({
  place: placeSchema,
  profile: profileSchema.nullable(),
  postsLast30: z.number().int().nullable(),
  lastPostAt: z.string().nullable(),
});
export type Contender = z.infer<typeof contenderSchema>;

const fixSchema = z.object({
  status: z.enum(["missing", "behind", "fine"]),
  title: z.string(),
  detail: z.string(),
  you: z.string(),
  others: z.string(),
});

export const checkResultSchema = z.object({
  searchTerm: z.string(),
  yourRank: z.number().int(),
  you: contenderSchema,
  rivals: z.array(contenderSchema),
  fixes: z.array(fixSchema),
  summary: z.string(),
  missing: z.array(z.string()),
});
export type CheckResult = z.infer<typeof checkResultSchema>;

export const gridResultSchema = z.object({
  searchTerm: z.string(),
  size: z.number().int(),
  spacingKm: z.number(),
  points: z.array(
    z.object({
      row: z.number().int(),
      col: z.number().int(),
      latitude: z.number(),
      longitude: z.number(),
      rank: z.number().int(),
      top: z.array(z.string()).nullable(),
      missing: z.boolean(),
    }),
  ),
  summary: z.string(),
});
export type GridResult = z.infer<typeof gridResultSchema>;

const themeSchema = z.object({
  name: z.string(),
  positive: z.number().int(),
  negative: z.number().int(),
});
export type Theme = z.infer<typeof themeSchema>;

const reviewSchema = z.object({
  id: z.string(),
  author: z.string(),
  rating: z.number().int(),
  text: z.string(),
  at: z.string(),
  answered: z.boolean(),
  url: z.string(),
  replyDraft: z.string().optional(),
});
export type Review = z.infer<typeof reviewSchema>;

export const reviewsResultSchema = z.object({
  total: z.number().int(),
  rating: z.number().nullable(),
  fetched: z.number().int(),
  replyRate: z.number().int().nullable(),
  distribution: z.array(z.number().int()).length(5),
  unanswered: z.array(reviewSchema),
  themes: z.array(themeSchema),
  rivals: z.array(
    z.object({
      cid: z.string(),
      title: z.string(),
      fetched: z.number().int(),
      replyRate: z.number().int().nullable(),
      themes: z.array(themeSchema),
    }),
  ),
  summary: z.string(),
  missing: z.array(z.string()),
});
export type ReviewsResult = z.infer<typeof reviewsResultSchema>;

export const watchResultSchema = z.object({
  changes: z.array(
    z.object({
      cid: z.string(),
      title: z.string(),
      rating: z.number().nullable(),
      ratingDelta: z.number().nullable(),
      newReviews: z.number().int().nullable(),
      newPosts: z.number().int(),
      photosDelta: z.number().int().nullable(),
      addedCategories: z.array(z.string()),
      removedCategories: z.array(z.string()),
      hoursChanged: z.boolean(),
    }),
  ),
  since: z.string().nullable(),
  summary: z.string(),
});
export type WatchResult = z.infer<typeof watchResultSchema>;

type RunKind = "check" | "grid" | "reviews" | "watch";

const runSchema = z.object({
  id: z.string(),
  kind: z.enum(["check", "grid", "reviews", "watch"]),
  status: z.enum(["running", "done", "failed"]),
  result: z.unknown(),
  error: z.string(),
  createdAt: z.string(),
  finishedAt: z.string().nullable(),
  cached: z.boolean(),
});
export type Run = z.infer<typeof runSchema>;

const pricesSchema = z.object({
  find: z.number().int(),
  check: z.number().int(),
  grid3: z.number().int(),
  grid5: z.number().int(),
  reviews: z.number().int(),
  watchPerCompetitor: z.number().int(),
  reply: z.number().int(),
});

const watchSchema = z.object({
  enabled: z.boolean(),
  nextRunAt: z.string().nullable(),
  competitors: z.array(z.object({ cid: z.string(), title: z.string() })),
  creditsPerWeek: z.number().int(),
});

const overviewSchema = z.object({
  business: businessSchema.nullable(),
  runs: z.object({
    check: runSchema.optional(),
    grid: runSchema.optional(),
    reviews: runSchema.optional(),
    watch: runSchema.optional(),
  }),
  watch: watchSchema,
  prices: pricesSchema,
  aiReady: z.boolean(),
});
export type Overview = z.infer<typeof overviewSchema>;

/**
 * Wraps a paid mutation so a second click in the same tick, before React
 * has re-rendered the disabled button, sends nothing.
 */
function once<TData, TVars>(
  client: QueryClient,
  mutationKey: readonly unknown[],
  mutation: UseMutationResult<TData, Error, TVars>,
): UseMutationResult<TData, Error, TVars> {
  const mutate: typeof mutation.mutate = (...args) => {
    if (client.isMutating({ mutationKey: [...mutationKey] }) === 0) {
      mutation.mutate(...args);
    }
  };
  return { ...mutation, mutate };
}

const base = (projectId: string) =>
  `/api/v1/projects/${encodeURIComponent(projectId)}/local`;

const overviewKey = (projectId: string) => ["local-visibility", projectId];

// The overview is free (stored data only), so it may refetch; every paid
// call below is a mutation that runs only on a click and never retries.
export function useOverview(projectId: string) {
  return useQuery({
    queryKey: overviewKey(projectId),
    queryFn: () =>
      apiRequest(`${base(projectId)}/overview`, overviewSchema, "POST", {}),
    retry: false,
  });
}

function patchOverview(
  client: QueryClient,
  projectId: string,
  update: (overview: Overview) => Overview,
) {
  client.setQueryData<Overview>(overviewKey(projectId), (old) =>
    old ? update(old) : old,
  );
}

export function useFindBusiness(projectId: string) {
  const client = useQueryClient();
  const mutationKey = ["local-find", projectId];
  const mutation = useMutation({
    mutationKey,
    mutationFn: (query: string) =>
      apiRequest(
        `${base(projectId)}/find`,
        z.object({ places: z.array(placeSchema), cached: z.boolean() }),
        "POST",
        { query },
      ),
    retry: false,
  });
  return once(client, mutationKey, mutation);
}

export function useSaveBusiness(projectId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { cid: string; searchTerm: string }) =>
      apiRequest(`${base(projectId)}/business`, businessSchema, "POST", input),
    retry: false,
    onSuccess: () =>
      client.invalidateQueries({ queryKey: overviewKey(projectId) }),
  });
}

export function useStartRun(projectId: string) {
  const client = useQueryClient();
  const mutationKey = ["local-start", projectId];
  const mutation = useMutation({
    mutationKey,
    mutationFn: (input: { kind: RunKind; size?: number; spacingKm?: number }) =>
      apiRequest(`${base(projectId)}/runs/start`, runSchema, "POST", {
        kind: input.kind,
        size: input.size ?? 0,
        spacingKm: input.spacingKm ?? 0,
      }),
    retry: false,
    onSuccess: (run) =>
      patchOverview(client, projectId, (overview) => ({
        ...overview,
        runs: { ...overview.runs, [run.kind]: run },
      })),
  });
  return once(client, mutationKey, mutation);
}

const POLL_MS = 4000;

/** Polls a running run (a free read) and writes it into the overview. */
export function useRunPolling(projectId: string, run: Run | undefined) {
  const client = useQueryClient();
  const running = run?.status === "running";
  return useQuery({
    queryKey: ["local-visibility-run", projectId, run?.id],
    queryFn: async () => {
      const next = await apiRequest(
        `${base(projectId)}/runs/get`,
        runSchema,
        "POST",
        { runId: run?.id },
      );
      if (next.status !== "running") {
        patchOverview(client, projectId, (overview) => ({
          ...overview,
          runs: { ...overview.runs, [next.kind]: next },
        }));
        if (next.kind === "check" || next.kind === "watch") {
          void client.invalidateQueries({ queryKey: overviewKey(projectId) });
        }
      }
      return next;
    },
    enabled: running,
    refetchInterval: (query) =>
      query.state.data?.status === "running" || !query.state.data
        ? POLL_MS
        : false,
    retry: false,
  });
}

export function useSetWatch(projectId: string) {
  const client = useQueryClient();
  const mutationKey = ["local-watch", projectId];
  const mutation = useMutation({
    mutationKey,
    mutationFn: (enabled: boolean) =>
      apiRequest(`${base(projectId)}/watch`, watchSchema, "POST", { enabled }),
    retry: false,
    onSuccess: () =>
      client.invalidateQueries({ queryKey: overviewKey(projectId) }),
  });
  return once(client, mutationKey, mutation);
}

export function useDraftReply(projectId: string, reviewId: string) {
  const client = useQueryClient();
  const mutationKey = ["local-reply", projectId, reviewId];
  const mutation = useMutation({
    mutationKey,
    mutationFn: (input: { reviewId: string; rewrite: boolean }) =>
      apiRequest(
        `${base(projectId)}/reply`,
        z.object({ reply: z.string() }),
        "POST",
        {
          reviewId: input.reviewId,
          rewrite: input.rewrite,
          idempotencyKey: crypto.randomUUID().replaceAll("-", ""),
        },
      ),
    retry: false,
  });
  return once(client, mutationKey, mutation);
}

/** Parses a finished run's result, or undefined while it runs or failed. */
export function runResult<T>(run: Run | undefined, schema: z.ZodType<T>) {
  if (run?.status !== "done") return undefined;
  const parsed = schema.safeParse(run.result);
  return parsed.success ? parsed.data : undefined;
}
