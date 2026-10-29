import { getKeywordResearch } from "@/client/features/keywords/keywordsApi";
import type {
  KeywordMode,
  ResultLimit,
} from "@/client/features/keywords/keywordResearchTypes";

type KeywordResearchRequestInput = {
  projectId: string;
  keywordInput: string;
  locationCode: number | undefined;
  locationName: string | undefined;
  resultLimit: ResultLimit;
  mode: KeywordMode;
  clickstream: boolean;
  groupKeywords: boolean;
};

type KeywordResearchRequest = {
  projectId: string;
  keywords: string[];
  seedKeyword: string;
  locationCode: number | undefined;
  locationName: string | undefined;
  resultLimit: ResultLimit;
  mode: KeywordMode;
  clickstream: boolean;
  groupKeywords: boolean;
};

function parseKeywordInput(value: string) {
  return value
    .split(/[\n,]/)
    .map((keyword) => keyword.trim())
    .filter(Boolean);
}

export const KEYWORD_RESEARCH_STALE_TIME_MS = 24 * 60 * 60 * 1000;

export function buildKeywordResearchRequest(
  input: KeywordResearchRequestInput,
): KeywordResearchRequest | null {
  const keywords = parseKeywordInput(input.keywordInput);
  const seedKeyword = keywords[0] ?? "";
  if (!seedKeyword) return null;

  return {
    projectId: input.projectId,
    keywords,
    seedKeyword,
    locationCode: input.locationCode,
    locationName: input.locationName,
    resultLimit: input.resultLimit,
    mode: input.mode,
    clickstream: input.clickstream,
    groupKeywords: input.groupKeywords,
  };
}

export function buildKeywordResearchQueryKey(
  request: KeywordResearchRequest | null,
) {
  return request
    ? [
        "keywordResearch",
        request.projectId,
        request.keywords,
        request.locationCode,
        request.locationName,
        request.resultLimit,
        request.mode,
        request.clickstream,
        request.groupKeywords,
      ]
    : ["keywordResearch", "idle"];
}

export function keywordResearchQueryFn(request: KeywordResearchRequest) {
  return getKeywordResearch({
    data: {
      projectId: request.projectId,
      keywords: request.keywords,
      locationCode: request.locationCode,
      locationName: request.locationName,
      resultLimit: request.resultLimit,
      mode: request.mode,
      clickstream: request.clickstream,
      groupKeywords: request.groupKeywords,
    },
  });
}
