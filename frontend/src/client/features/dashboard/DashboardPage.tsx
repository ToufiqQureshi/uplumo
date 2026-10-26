import type { ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { DashboardOnboarding } from "./DashboardOnboarding";
import {
  AuditHealthCard,
  BacklinkPulseCard,
  GscCard,
} from "@/client/features/dashboard/DashboardCards";
import {
  AiVisibilityCard,
  AnalyticsSummaryCard,
  RankTrackingCard,
} from "@/client/features/dashboard/InsightCards";
import { Ga4Card } from "@/client/features/dashboard/Ga4Card";
import { DashboardShortcuts } from "@/client/features/dashboard/DashboardShortcuts";
import { DashboardCustomize } from "@/client/features/dashboard/DashboardCustomize";
import { PageHeader } from "@/client/components/PageHeader";
import { QueryError } from "@/client/components/QueryState";
import {
  getDashboardActivation,
  getDashboardLayout,
  getDashboardOverview,
  refreshDashboardBacklinkSnapshot,
  saveDashboardLayout,
} from "@/client/features/dashboard/dashboardApi";
import {
  resolveLayout,
  type DashboardCardId,
  type ResolvedCard,
} from "@/client/features/dashboard/dashboardLayout";
import { Skeleton } from "@/client/components/ui/skeleton";

export function DashboardPage({ projectId }: { projectId: string }) {
  const queryClient = useQueryClient();

  const activationQuery = useQuery({
    queryKey: ["dashboardActivation", projectId],
    queryFn: () => getDashboardActivation({ data: { projectId } }),
  });
  const layoutKey = ["dashboardLayout", projectId];
  const layoutQuery = useQuery({
    queryKey: layoutKey,
    queryFn: () => getDashboardLayout({ data: { projectId } }),
  });
  const layoutMutation = useMutation({
    mutationFn: (cards: ResolvedCard[]) =>
      saveDashboardLayout({ data: { projectId, cards } }),
    retry: false,
    meta: { errorToast: false },
    onSuccess: (out) =>
      queryClient.setQueryData(layoutKey, out.cards.length ? out.cards : null),
  });
  const overviewQuery = useQuery({
    queryKey: ["dashboardOverview", projectId],
    queryFn: () => getDashboardOverview({ data: { projectId } }),
    refetchInterval: (query) =>
      query.state.data?.audit?.status === "running" ? 3000 : false,
  });

  const activation = activationQuery.data;
  const overview = overviewQuery.data;

  const refreshMutation = useMutation({
    mutationFn: () => refreshDashboardBacklinkSnapshot({ data: { projectId } }),
    retry: false,
    meta: { errorToast: false },
    onSuccess: () =>
      void queryClient.invalidateQueries({
        queryKey: ["dashboardOverview", projectId],
      }),
  });

  if (activationQuery.isError && !activation) {
    return (
      <div className="px-4 py-4 md:px-6 md:py-6">
        <QueryError
          error={activationQuery.error}
          fallback="Failed to load dashboard"
          onRetry={() => void activationQuery.refetch()}
          isRetrying={activationQuery.isFetching}
        />
      </div>
    );
  }

  // Wait for the overview too: rendering cards from `overview === undefined`
  // flashes their empty states (and reshuffles the data-first sort) once the
  // real data lands. An overview error falls through so the page still loads,
  // with the error in place of the audit and backlink cards.
  if (!activation || overviewQuery.isPending || layoutQuery.isPending) {
    return (
      <div className="px-4 py-4 md:px-6 md:py-6" aria-busy>
        <div className="mx-auto flex max-w-7xl flex-col gap-5">
          <Skeleton className="h-14 w-72" />
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Skeleton className="h-20" />
            <Skeleton className="h-20" />
            <Skeleton className="h-20" />
            <Skeleton className="h-20" />
          </div>
          <Skeleton className="h-36" />
          <div className="grid gap-5 lg:grid-cols-2">
            <Skeleton className="h-44" />
            <Skeleton className="h-44" />
          </div>
        </div>
      </div>
    );
  }

  const showBacklinks = activation.domain !== null;
  const gscConnected = activation.gsc.connected;
  const ga4Connected = activation.ga4.connected;

  // A layout that fails to load falls back to the default; the page still works.
  const layout = resolveLayout(layoutQuery.data ?? null);

  // Each card the dashboard can show, or null when it cannot show here (no
  // overview data, no domain for backlinks, a dismissed GA4 pitch).
  const renderCard: Record<DashboardCardId, () => ReactNode> = {
    gsc: () => <GscCard projectId={projectId} connected={gscConnected} />,
    analytics: () => <AnalyticsSummaryCard projectId={projectId} />,
    ai_visibility: () => <AiVisibilityCard projectId={projectId} />,
    rank_tracking: () => <RankTrackingCard projectId={projectId} />,
    audit: () =>
      overview ? (
        <AuditHealthCard projectId={projectId} audit={overview.audit} />
      ) : null,
    backlinks: () =>
      overview && showBacklinks ? (
        <BacklinkPulseCard
          projectId={projectId}
          backlinks={overview.backlinks}
          refreshing={refreshMutation.isPending}
          canRefresh={overview.backlinkRefreshAvailable}
          onRefresh={() => refreshMutation.mutate()}
          refreshError={refreshMutation.error}
        />
      ) : null,
    ga4: () =>
      ga4Connected || !activation.ga4.cardDismissedAt ? (
        <Ga4Card projectId={projectId} connected={ga4Connected} />
      ) : null,
  };
  const cards = layout
    .filter((card) => !card.hidden)
    .map((card) => ({ id: card.id, node: renderCard[card.id]() }))
    .filter((card) => card.node !== null);

  return (
    <div className="px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto flex max-w-7xl flex-col gap-5">
        <PageHeader
          title={activation.domain ?? "Dashboard"}
          description="Your traffic, AI visibility and site health in one place."
          actions={
            <DashboardCustomize
              cards={layout}
              onSave={(next) => layoutMutation.mutateAsync(next)}
              onReset={() => layoutMutation.mutateAsync([])}
              saving={layoutMutation.isPending}
              error={layoutMutation.error}
            />
          }
        />

        <DashboardShortcuts projectId={projectId} />

        {activationQuery.isError ? (
          <QueryError
            error={activationQuery.error}
            fallback="Failed to refresh dashboard"
            onRetry={() => void activationQuery.refetch()}
            isRetrying={activationQuery.isFetching}
          />
        ) : null}

        {overviewQuery.isError ? (
          <QueryError
            error={overviewQuery.error}
            fallback="Failed to load site audit and backlink summaries"
            onRetry={() => void overviewQuery.refetch()}
            isRetrying={overviewQuery.isFetching}
          />
        ) : null}

        {/* Every card is half width on large screens (only the checklist spans).
          Cards follow the member's saved order. Cards in a row stretch to the
          same height. */}
        <div className="grid gap-5 lg:grid-cols-2">
          {cards.map((card) => (
            <div key={card.id} data-dashboard-card={card.id}>
              {card.node}
            </div>
          ))}
        </div>
        {cards.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            All cards are hidden. Use Customize to show them again.
          </p>
        ) : null}

        {/* Below the cards: the insights a returning customer came for come
          first; the setup checklist stays until its steps are done. */}
        <DashboardOnboarding
          key={projectId}
          projectId={projectId}
          activation={activation}
        />
      </div>
    </div>
  );
}
