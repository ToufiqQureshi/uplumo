import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";
import { researchScopeSchema } from "@/shared/researchScope";
import type { DomainKeywordsFilters } from "@/types/schemas/domain";

const nullableNumber = z.number().nullable();

const overviewSchema = z.object({
  domain: z.string(),
  scope: researchScopeSchema,
  displayTarget: z.string(),
  organicTraffic: nullableNumber,
  organicKeywords: nullableNumber,
  backlinks: nullableNumber,
  referringDomains: nullableNumber,
  hasData: z.boolean(),
  fetchedAt: z.string(),
});

const keywordSuggestionsSchema = z.array(
  z.object({
    keyword: z.string(),
    position: nullableNumber,
    searchVolume: nullableNumber,
    traffic: nullableNumber,
    cpc: nullableNumber,
    keywordDifficulty: nullableNumber,
  }),
);

const keywordsPageSchema = z.object({
  domain: z.string(),
  page: z.number(),
  pageSize: z.number(),
  totalCount: nullableNumber,
  hasMore: z.boolean(),
  keywords: z.array(
    z.object({
      keyword: z.string(),
      position: nullableNumber,
      searchVolume: nullableNumber,
      traffic: nullableNumber,
      cpc: nullableNumber,
      url: z.string().nullable(),
      relativeUrl: z.string().nullable(),
      keywordDifficulty: nullableNumber,
      // Present only when the provider reports them.
      intent: z.string().nullable().optional(),
      type: z.string().nullable().optional(),
    }),
  ),
  fetchedAt: z.string(),
});

const pagesPageSchema = z.object({
  domain: z.string(),
  page: z.number(),
  pageSize: z.number(),
  totalCount: nullableNumber,
  hasMore: z.boolean(),
  pages: z.array(
    z.object({
      page: z.string(),
      relativePath: z.string().nullable(),
      organicTraffic: nullableNumber,
      keywords: nullableNumber,
    }),
  ),
  fetchedAt: z.string(),
});

type DomainTarget = {
  projectId: string;
  domain: string;
  scope?: string;
  locationCode?: number;
  languageCode?: string;
};
type DomainPageRequest = DomainTarget & {
  page: number;
  pageSize: number;
  sortMode: string;
  sortOrder: string;
  filters: DomainKeywordsFilters | Record<string, unknown>;
  search?: string;
};

function post<T>(
  endpoint: string,
  schema: z.ZodType<T>,
  { projectId, ...body }: { projectId: string } & Record<string, unknown>,
) {
  return apiRequest(
    `/api/v1/projects/${encodeURIComponent(projectId)}/domain/${endpoint}`,
    schema,
    "POST",
    body,
  );
}

export function getDomainOverview({ data }: { data: DomainTarget }) {
  return post("overview", overviewSchema, data);
}

export function getDomainKeywordSuggestions({ data }: { data: DomainTarget }) {
  return post("keyword-suggestions", keywordSuggestionsSchema, {
    ...data,
    userInitiated: true,
  });
}

export function getDomainKeywordsPage({ data }: { data: DomainPageRequest }) {
  return post("keywords", keywordsPageSchema, data);
}

export function getDomainPagesPage({ data }: { data: DomainPageRequest }) {
  return post("pages", pagesPageSchema, data);
}
