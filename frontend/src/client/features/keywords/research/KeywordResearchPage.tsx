import { useMutation, useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Bot, Search } from "lucide-react";
import { useRef, useState, type FormEvent } from "react";
import { toast } from "sonner";
import { ErrorState } from "@/client/components/ErrorState";
import { LocationSelect } from "@/client/components/LocationSelect";
import { PageHeader } from "@/client/components/PageHeader";
import { SearchCard, SearchInput } from "@/client/components/SearchCard";
import { SegmentedToggle } from "@/client/components/SegmentedToggle";
import { SkeletonCard } from "@/client/components/SkeletonPresets";
import { Button } from "@/client/components/ui/button";
import { Card, CardContent } from "@/client/components/ui/card";
import { billingStatusQueryOptions } from "@/client/features/billing/plan/planApi";
import { CreditCostHint } from "@/client/features/billing/credits/CreditCostHint";
import { KeywordAgentPage } from "@/client/features/keywords/agent/KeywordAgentPage";
import {
  researchKeywords,
  saveKeywords,
} from "@/client/features/keywords/keywordsApi";
import { useProjectMarket } from "@/client/features/projects/useProjectMarket";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { LOCATION_OPTIONS } from "@/shared/keyword-locations";
import type { KeywordResearchRow } from "@/types/keywords";
import { KeywordIdeasTable } from "./KeywordIdeasTable";
import { KeywordMetrics } from "./KeywordMetrics";
import { findSeedRow } from "./researchModel";
import { TopResults } from "./TopResults";

type Mode = "research" | "agent";
type Request = { keyword: string; locationCode: number };

// Same keyword and country within a day reuses the answer: the server caches
// it for 24 hours, so the browser never asks again sooner.
const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * Keyword Research: type a keyword, pick a country, and see every number the
 * research bought. Nothing paid runs until Research is pressed; filtering,
 * sorting and opening a keyword use rows already paid for.
 */
export function KeywordResearchPage({
  projectId,
  initialKeyword,
}: {
  projectId: string;
  initialKeyword?: string;
}) {
  const market = useProjectMarket(projectId);
  const status = useQuery(billingStatusQueryOptions);
  const [mode, setMode] = useState<Mode>("research");
  const [keyword, setKeyword] = useState(initialKeyword ?? "");
  const [pickedLocation, setPickedLocation] = useState<number | null>(null);
  const [request, setRequest] = useState<Request | null>(null);
  const [inputError, setInputError] = useState<string | null>(null);
  const [opened, setOpened] = useState<KeywordResearchRow | null>(null);
  const [saved, setSaved] = useState<ReadonlySet<string>>(new Set());
  const [saving, setSaving] = useState<ReadonlySet<string>>(new Set());
  const detailsRef = useRef<HTMLDivElement>(null);
  const locationCode = pickedLocation ?? market?.locationCode ?? null;

  const research = useQuery({
    queryKey: ["keywordResearch", projectId, request],
    queryFn: () =>
      researchKeywords({
        projectId,
        keywords: [request!.keyword],
        locationCode: request!.locationCode,
        resultLimit: 150,
        mode: "auto",
      }),
    enabled: request !== null,
    staleTime: DAY_MS,
    gcTime: DAY_MS,
    // A second attempt would buy the same research again.
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  const save = useMutation({
    mutationFn: (rows: KeywordResearchRow[]) =>
      saveKeywords({
        projectId,
        keywords: rows.map((r) => r.keyword),
        locationCode: request?.locationCode,
        metrics: rows.map((r) => ({
          keyword: r.keyword,
          searchVolume: r.searchVolume,
          cpc: r.cpc,
          competition: r.competition,
          keywordDifficulty: r.keywordDifficulty,
          intent: r.intent,
          monthlySearches: r.trend,
        })),
      }),
    onMutate: (rows) =>
      setSaving((s) => new Set([...s, ...rows.map((r) => r.keyword)])),
    onSuccess: (_, rows) => {
      setSaved((s) => new Set([...s, ...rows.map((r) => r.keyword)]));
      toast.success(
        rows.length === 1
          ? `Saved “${rows[0].keyword}”.`
          : `Saved ${rows.length} keywords.`,
      );
    },
    onError: (error) => toast.error(getStandardErrorMessage(error)),
    onSettled: (_, __, rows) =>
      setSaving((s) => {
        const next = new Set(s);
        for (const r of rows) next.delete(r.keyword);
        return next;
      }),
  });

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = keyword.trim();
    if (!trimmed) {
      setInputError("Type a keyword to research.");
      return;
    }
    if (Array.from(trimmed).length > 80) {
      setInputError("Keep the keyword under 80 characters.");
      return;
    }
    if (locationCode === null) {
      setInputError("Pick a country.");
      return;
    }
    setInputError(null);
    setOpened(null);
    setRequest({ keyword: trimmed, locationCode });
  };

  const isFree = status.data?.plan === "free" && !status.data.unlimited;
  const rows = research.data?.rows ?? [];
  const seed = request ? findSeedRow(rows, request.keyword) : null;
  const focus = opened ?? seed;
  const open = (row: KeywordResearchRow) => {
    setOpened(row);
    detailsRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-5 px-4 py-6 md:px-6">
      <PageHeader
        title="Keyword Research"
        description="Type a keyword and pick a country. You get every number we have on it, the ideas around it, and who ranks now."
        actions={
          isFree ? null : (
            <SegmentedToggle<Mode>
              showLabels
              value={mode}
              onChange={setMode}
              items={[
                {
                  value: "research",
                  label: "Research",
                  icon: <Search aria-hidden />,
                },
                {
                  value: "agent",
                  label: "Ask the AI planner",
                  icon: <Bot aria-hidden />,
                },
              ]}
            />
          )
        }
      />

      {isFree ? (
        <Card>
          <CardContent className="space-y-3">
            <h2 className="text-base font-semibold">
              Keyword research is part of every paid plan
            </h2>
            <p className="text-sm text-muted-foreground">
              Upgrade to see search volumes, difficulty, ideas and who ranks on
              Google for any keyword.
            </p>
            <Button render={<Link to="/billing/plan" />}>See plans</Button>
          </CardContent>
        </Card>
      ) : mode === "agent" ? (
        <KeywordAgentPage projectId={projectId} />
      ) : (
        <>
          <SearchCard
            onSubmit={submit}
            pending={research.isFetching}
            error={inputError}
            errorId="keyword-error"
            secondRow={<CreditCostHint feature="keyword_research" action="" />}
          >
            <SearchInput
              aria-label="Keyword"
              aria-invalid={inputError ? true : undefined}
              aria-describedby={inputError ? "keyword-error" : undefined}
              placeholder="e.g. seo audit tool"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
            {locationCode === null ? (
              <SkeletonCard />
            ) : (
              <LocationSelect
                value={locationCode}
                onChange={setPickedLocation}
                className="w-full lg:w-56"
              />
            )}
          </SearchCard>

          {research.isError ? (
            <ErrorState
              title="Research didn't finish"
              message={getStandardErrorMessage(research.error)}
            />
          ) : research.isFetching && !research.data ? (
            <SkeletonCard />
          ) : request && research.data ? (
            <>
              <div ref={detailsRef} className="scroll-mt-4">
                <Card>
                  <CardContent className="space-y-4">
                    <div className="flex flex-wrap items-baseline justify-between gap-2">
                      <h2 className="font-serif text-3xl">
                        {focus?.keyword ?? request.keyword}
                      </h2>
                      <p className="text-sm text-muted-foreground">
                        {LOCATION_OPTIONS.find(
                          (o) => o.code === request.locationCode,
                        )?.label ?? "Chosen country"}
                        {research.data.placeFilter
                          ? ` · ideas limited to “${research.data.placeFilter}”`
                          : ""}
                      </p>
                    </div>
                    {focus ? (
                      <>
                        <KeywordMetrics row={focus} />
                        {opened && seed && opened.keyword !== seed.keyword ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setOpened(null)}
                          >
                            Back to “{seed.keyword}”
                          </Button>
                        ) : null}
                      </>
                    ) : (
                      <p className="text-sm text-muted-foreground">
                        Google has no numbers for this exact keyword. Pick one
                        of the ideas below to see its details.
                      </p>
                    )}
                  </CardContent>
                </Card>
              </div>

              <section className="space-y-2">
                <h2 className="text-base font-semibold">
                  Ideas around “{request.keyword}”
                </h2>
                <p className="text-sm text-muted-foreground">
                  Click a keyword to see all its numbers above. Filtering and
                  sorting are free.
                </p>
                <KeywordIdeasTable
                  rows={rows}
                  activeKeyword={opened?.keyword ?? null}
                  onOpen={open}
                  onSave={(picked) => {
                    if (picked.length > 0) save.mutate(picked);
                  }}
                  savedKeywords={saved}
                  savingKeywords={saving}
                />
              </section>

              <TopResults
                key={`${focus?.keyword ?? request.keyword}-${request.locationCode}`}
                projectId={projectId}
                keyword={focus?.keyword ?? request.keyword}
                locationCode={request.locationCode}
              />
            </>
          ) : null}
        </>
      )}
    </div>
  );
}
