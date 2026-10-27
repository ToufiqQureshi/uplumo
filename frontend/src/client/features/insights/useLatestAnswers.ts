import { useQuery } from "@tanstack/react-query";
import {
  trackingApi,
  trackingKey,
} from "@/client/features/ai-search/trackingApi";

/**
 * The AI tracker's configuration and the answers of its latest finished
 * check. Reads saved evidence only: no provider call, no credits.
 */
export function useLatestAnswers(projectId: string) {
  const state = useQuery({
    queryKey: trackingKey(projectId),
    queryFn: () => trackingApi.state(projectId),
    staleTime: 60_000,
  });
  const latest = state.data?.runs.find(
    (r) => r.status === "completed" || r.status === "partial",
  );
  const answers = useQuery({
    queryKey: [...trackingKey(projectId), "answers", latest?.id],
    queryFn: () => trackingApi.answers(projectId, latest!.id),
    enabled: Boolean(latest),
    staleTime: 60_000,
  });
  return {
    isPending: state.isPending || (Boolean(latest) && answers.isPending),
    isError: state.isError || answers.isError,
    configuration: state.data?.configuration ?? null,
    run: latest ?? null,
    answers: answers.data ?? [],
  };
}
