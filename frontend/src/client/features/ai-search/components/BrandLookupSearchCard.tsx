import type { FormEvent } from "react";
import { ResearchScopeSelect } from "@/client/components/ResearchScopeSelect";
import { SearchCard, SearchInput } from "@/client/components/SearchCard";
import { Field, FieldDescription } from "@/client/components/ui/field";
import { Input } from "@/client/components/ui/input";
import type { ResearchScope } from "@/shared/researchScope";
import { BRAND_LOOKUP_MAX_INPUT_LENGTH } from "@/types/schemas/ai-search";
import { CreditCostHint } from "@/client/features/billing/credits/CreditCostHint";
import { LocationSelect } from "@/client/components/LocationSelect";

/** ChatGPT's mentions data covers this market only (US). */
const CHATGPT_LOCATION_CODE = 2840;

type Props = {
  query: string;
  onQueryChange: (next: string) => void;
  scope: ResearchScope;
  onScopeChange: (next: ResearchScope) => void;
  scopeDisabledReason: string | undefined;
  locationCode: number;
  onLocationChange: (next: number) => void;
  competitors: string;
  onCompetitorsChange: (next: string) => void;
  onSubmit: (event: FormEvent) => void;
  isLoading: boolean;
  validationError: { field: "query" | "competitors"; message: string } | null;
};

/**
 * Adding competitors adds one cross_aggregated_metrics call per platform
 * bought, measured live at $0.101 each: about 130 credits (1 credit is
 * $1/1280 of provider cost), 260 in the US, where ChatGPT is compared too.
 * The base lookup's credits come from the usage costs, as for every paid
 * action, so the screen shows one number for it.
 */
const COMPETITOR_CREDITS_PER_PLATFORM = 130;

export function BrandLookupSearchCard({
  query,
  onQueryChange,
  scope,
  onScopeChange,
  scopeDisabledReason,
  locationCode,
  onLocationChange,
  competitors,
  onCompetitorsChange,
  onSubmit,
  isLoading,
  validationError,
}: Props) {
  const hasCompetitors = competitors.trim().length > 0;
  const queryError = validationError?.field === "query";
  const competitorsError = validationError?.field === "competitors";

  return (
    <div className="space-y-2">
      <SearchCard
        onSubmit={onSubmit}
        pending={isLoading}
        error={validationError?.message}
        errorId="brand-lookup-input-error"
        secondRow={
          <>
            <Field>
              <Input
                placeholder="Add competitors (comma-separated)"
                value={competitors}
                onChange={(event) => onCompetitorsChange(event.target.value)}
                autoComplete="off"
                spellCheck={false}
                aria-label="Competitors"
                aria-invalid={competitorsError || undefined}
                aria-describedby={
                  competitorsError ? "brand-lookup-input-error" : undefined
                }
              />
              <FieldDescription>
                Add up to 5 competitor brands or domains to see your Share of
                Voice.
              </FieldDescription>
            </Field>
            {hasCompetitors ? (
              <p className="text-xs text-muted-foreground tabular-nums">
                Comparing competitors adds about{" "}
                {COMPETITOR_CREDITS_PER_PLATFORM *
                  (locationCode === CHATGPT_LOCATION_CODE ? 2 : 1)}{" "}
                credits.
              </p>
            ) : null}
          </>
        }
      >
        <SearchInput
          placeholder="Enter a brand name or domain"
          aria-label="Brand name or domain"
          value={query}
          maxLength={BRAND_LOOKUP_MAX_INPUT_LENGTH}
          onChange={(event) => onQueryChange(event.target.value)}
          aria-invalid={queryError || undefined}
          aria-describedby={queryError ? "brand-lookup-input-error" : undefined}
          autoComplete="off"
          spellCheck={false}
        />

        <LocationSelect
          value={locationCode}
          onChange={onLocationChange}
          className="w-full lg:w-48"
        />
        <ResearchScopeSelect
          value={scope}
          className="w-full lg:w-40"
          onChange={onScopeChange}
          disabledReason={scopeDisabledReason}
        />
      </SearchCard>
      <CreditCostHint feature="ai_visibility" action="brand_lookup" />
    </div>
  );
}
