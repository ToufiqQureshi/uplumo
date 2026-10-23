import { useQuery } from "@tanstack/react-query";
import { getDomainOverview } from "@/client/features/domain/domainApi";
import type { ResearchScope } from "@/shared/researchScope";

type Input = {
  projectId: string;
  domain: string;
  scope: ResearchScope;
  locationCode: number | undefined;
};

export function useDomainOverviewQuery(input: Input) {
  const trimmedDomain = input.domain.trim();

  return useQuery({
    enabled: trimmedDomain !== "",
    retry: false,
    retryOnMount: false, // The paid POST may already have completed when a 5xx reaches us.
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    queryKey: [
      "domain-overview",
      input.projectId,
      trimmedDomain,
      input.scope,
      input.locationCode,
    ],
    queryFn: () =>
      getDomainOverview({
        data: {
          projectId: input.projectId,
          domain: trimmedDomain,
          scope: input.scope,
          locationCode: input.locationCode,
        },
      }),
    staleTime: (query) =>
      query.state.data?.hasData ? 12 * 60 * 60_000 : 5 * 60_000,
    gcTime: 12 * 60 * 60_000,
  });
}
