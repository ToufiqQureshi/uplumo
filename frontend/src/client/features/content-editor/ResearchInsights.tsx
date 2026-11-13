import { LOCATION_OPTIONS } from "@/shared/keyword-locations";
import { Button } from "@/client/components/ui/button";
import { useFreeSeoEditor } from "./FreeSeoEditorContext";
import { MAX_TARGET_TOPICS, parseTargetTopics } from "./target-topics";
import type { SearchBrief } from "./contentEditorApi";

const intentGuidance: Record<string, string> = {
  informational:
    "Explain the question clearly, then cover useful steps, examples and limitations.",
  commercial:
    "Help readers choose: compare options, selection criteria, verified prices and trade-offs.",
  transactional:
    "Help readers act: make the offer, availability, cost and next step clear. Check whether a product or service page fits better than an article.",
  navigational:
    "Readers may be looking for a particular website or page. Check whether an article can actually satisfy this query.",
};

export function ResearchInsights({
  brief,
  targetTopics,
}: {
  brief: SearchBrief;
  targetTopics: string;
}) {
  const { setDraftField } = useFreeSeoEditor();
  const topics = parseTargetTopics(targetTopics);
  const keywords = brief.keywords ?? [];
  const seed = keywords.find(
    (item) =>
      item.keyword.toLocaleLowerCase() === brief.keyword.toLocaleLowerCase(),
  );
  const country =
    LOCATION_OPTIONS.find((item) => item.code === brief.locationCode)?.label ??
    String(brief.locationCode);
  return (
    <div className="space-y-3 text-sm">
      <p className="text-xs text-muted-foreground">
        Research market: {country} · {brief.languageCode}. Keyword metrics are
        estimates, not promised traffic.
      </p>
      {seed?.intent && intentGuidance[seed.intent] ? (
        <div className="rounded-lg border bg-muted/30 p-3">
          <h3 className="font-medium">Search intent: {seed.intent}</h3>
          <p className="mt-1 text-xs leading-5">
            {intentGuidance[seed.intent]}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            Keyword-level evidence. Analyze full draft to review whether your
            content satisfies it.
          </p>
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">
          Search intent is not available for the primary topic. Do not infer it
          from the writing score.
        </p>
      )}
      {keywords.length ? (
        <details open>
          <summary className="cursor-pointer font-medium">
            Keyword opportunities ({keywords.length})
          </summary>
          <p className="my-2 text-xs text-muted-foreground">
            Choose only relevant supporting topics. Adding a topic does not
            insert text or buy another brief.
          </p>
          <ul className="max-h-72 space-y-2 overflow-y-auto">
            {keywords.map((item) => {
              const added = topics.some(
                (topic) =>
                  topic.toLocaleLowerCase() ===
                  item.keyword.toLocaleLowerCase(),
              );
              const canAdd =
                !added &&
                topics.length < MAX_TARGET_TOPICS &&
                [...topics, item.keyword].join(", ").length <= 120;
              return (
                <li key={item.keyword} className="rounded-md border p-2">
                  <div className="flex items-start justify-between gap-2">
                    <span className="break-words font-medium">
                      {item.keyword}
                    </span>
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={!canAdd}
                      onClick={() =>
                        setDraftField(
                          "targetKeyword",
                          [...topics, item.keyword].join(", "),
                        )
                      }
                    >
                      {added ? "Added" : "Add topic"}
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Monthly searches:{" "}
                    {item.searchVolume?.toLocaleString() ?? "Unknown"} ·
                    Difficulty:{" "}
                    {item.difficulty === null
                      ? "Unknown"
                      : `${item.difficulty}/100`}{" "}
                    · {item.intent || "Intent unknown"}
                  </p>
                  {item.updatedAt ? (
                    <p className="text-xs text-muted-foreground">
                      Metric updated: {item.updatedAt}
                    </p>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </details>
      ) : (
        <p role="status" className="text-xs text-muted-foreground">
          {brief.researchStatus === "unsupported_market"
            ? "Keyword metrics are not supported in this market; the competitor brief remains usable."
            : brief.researchStatus === "unavailable"
              ? "Keyword research could not be loaded. Competitor research is preserved; there is no automatic paid retry."
              : "No keyword metrics returned. This does not prove there is no demand."}
        </p>
      )}
    </div>
  );
}
