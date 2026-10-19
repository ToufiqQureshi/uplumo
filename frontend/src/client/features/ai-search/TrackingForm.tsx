import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Textarea } from "@/client/components/ui/textarea";
import {
  LOCATION_OPTIONS,
  SERP_LANGUAGE_OPTIONS,
} from "@/shared/keyword-locations";
import {
  trackingApi,
  trackingKey,
  type TrackingConfig,
  type TrackingEstimate,
} from "./trackingApi";

const ENGINES = [
  ["chat_gpt", "ChatGPT"],
  ["gemini", "Gemini"],
  ["google_ai_overview", "Google AI Overviews"],
  ["claude", "Claude (model API)"],
  ["perplexity", "Perplexity (model API)"],
] as const;
const selectClass =
  "rounded-md border border-input bg-background px-3 py-2 text-sm";

export function TrackingForm({
  projectId,
  saved,
  providerConfigured,
}: {
  projectId: string;
  saved: TrackingConfig | null;
  providerConfigured: boolean;
}) {
  const queryClient = useQueryClient();
  const [config, setConfig] = useState<TrackingConfig>(
    saved ?? {
      brand: "",
      domain: "",
      competitors: [],
      prompts: [{ text: "", topic: "General", paused: false }],
      engines: ["chat_gpt", "gemini"],
      locationCode: 2356,
      languageCode: "en",
      interval: "weekly",
      enabled: false,
      maxCreditsPerRun: 0,
    },
  );
  const [competitorText, setCompetitorText] = useState(
    saved?.competitors.map((c) => `${c.name}, ${c.domain}`).join("\n") ?? "",
  );
  const [review, setReview] = useState<{
    key: string;
    estimate: TrackingEstimate;
  } | null>(null);
  const key = JSON.stringify(config);
  const estimate = useMutation({
    mutationFn: (proposed: TrackingConfig) =>
      trackingApi.estimate(projectId, proposed),
    onSuccess: (data, proposed) =>
      setReview({ key: JSON.stringify(proposed), estimate: data }),
  });
  const save = useMutation({
    mutationFn: () =>
      trackingApi.save(projectId, {
        ...config,
        maxCreditsPerRun: review?.estimate.credits ?? 0,
      }),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: trackingKey(projectId) }),
  });
  const run = useMutation({
    mutationFn: () => trackingApi.run(projectId, approvedCredits),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: trackingKey(projectId) }),
  });
  const reviewed = review?.key === key;
  const unchanged =
    saved !== null &&
    JSON.stringify({ ...config, maxCreditsPerRun: 0 }) ===
      JSON.stringify({ ...saved, maxCreditsPerRun: 0 });
  // Saving remounts this form, which forgets the review; the saved
  // configuration keeps the credits that were approved for it, so an unchanged
  // saved check can run without reviewing again. The server checks the
  // ceiling against the saved questions either way.
  const approvedCredits = reviewed
    ? (review?.estimate.credits ?? 0)
    : unchanged
      ? (saved?.maxCreditsPerRun ?? 0)
      : 0;
  const update = (patch: Partial<TrackingConfig>) =>
    setConfig({ ...config, ...patch });
  const error = estimate.error ?? save.error ?? run.error;
  return (
    <section className="space-y-4 rounded-xl border bg-card p-5">
      <h2 className="text-lg font-semibold">Questions to monitor</h2>
      <p className="text-sm text-muted-foreground">
        Review the credit ceiling before saving a schedule or collecting fresh
        answers. Looking at saved results is free.
      </p>
      <div className="grid gap-4 md:grid-cols-2">
        <label className="space-y-1 text-sm">
          Brand name
          <Input
            value={config.brand}
            maxLength={100}
            onChange={(e) => update({ brand: e.target.value })}
          />
        </label>
        <label className="space-y-1 text-sm">
          Website domain
          <Input
            value={config.domain}
            onChange={(e) => update({ domain: e.target.value })}
            placeholder="example.com"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          Country
          <select
            className={selectClass}
            value={config.locationCode}
            onChange={(e) => update({ locationCode: Number(e.target.value) })}
          >
            {LOCATION_OPTIONS.map((l) => (
              <option key={l.code} value={l.code}>
                {l.label}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm">
          Language
          <select
            className={selectClass}
            value={config.languageCode}
            onChange={(e) => update({ languageCode: e.target.value })}
          >
            {SERP_LANGUAGE_OPTIONS.map((l) => (
              <option key={l.code} value={l.code}>
                {l.label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <fieldset className="flex flex-wrap gap-4">
        <legend className="mb-2 text-sm font-medium">Engines</legend>
        {ENGINES.map(([id, label]) => (
          <label key={id} className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={config.engines.includes(id)}
              onChange={(e) =>
                update({
                  engines: e.target.checked
                    ? [...config.engines, id]
                    : config.engines.filter((x) => x !== id),
                })
              }
            />
            {label}
          </label>
        ))}
      </fieldset>
      <p className="text-xs text-muted-foreground">
        ChatGPT, Gemini and AI Overviews collect consumer-site results through
        the provider queue. Claude and Perplexity use model APIs, can differ
        from their apps, and require a higher credit hold. Questions are your
        chosen sample of AI answers.
      </p>
      {config.prompts.map((p, i) => (
        <div
          key={i}
          className="grid gap-2 rounded-lg border p-3 md:grid-cols-[1fr_3fr_auto]"
        >
          <label className="text-sm">
            Topic
            <Input
              aria-label={`Topic ${i + 1}`}
              value={p.topic}
              maxLength={80}
              onChange={(e) =>
                update({
                  prompts: config.prompts.map((v, n) =>
                    n === i ? { ...v, topic: e.target.value } : v,
                  ),
                })
              }
            />
          </label>
          <label className="text-sm">
            Question
            <Textarea
              aria-label={`Question ${i + 1}`}
              value={p.text}
              maxLength={500}
              onChange={(e) =>
                update({
                  prompts: config.prompts.map((v, n) =>
                    n === i ? { ...v, text: e.target.value } : v,
                  ),
                })
              }
            />
          </label>
          <div className="flex items-center gap-2">
            <label className="text-sm">
              <input
                type="checkbox"
                checked={p.paused}
                onChange={(e) =>
                  update({
                    prompts: config.prompts.map((v, n) =>
                      n === i ? { ...v, paused: e.target.checked } : v,
                    ),
                  })
                }
              />{" "}
              Paused
            </label>
            <Button
              variant="ghost"
              onClick={() =>
                update({ prompts: config.prompts.filter((_, n) => n !== i) })
              }
              aria-label={`Remove question ${i + 1}`}
            >
              Remove
            </Button>
          </div>
        </div>
      ))}
      <Button
        variant="outline"
        disabled={config.prompts.length >= 30}
        onClick={() =>
          update({
            prompts: [
              ...config.prompts,
              { text: "", topic: "General", paused: false },
            ],
          })
        }
      >
        Add question
      </Button>
      <label className="block space-y-1 text-sm">
        Competitors: one name and domain per line, separated by a comma
        <Textarea
          value={competitorText}
          placeholder="Competitor name, competitor.com"
          onChange={(e) => {
            setCompetitorText(e.target.value);
            update({
              competitors: e.target.value
                .split("\n")
                .filter(Boolean)
                .map((line) => {
                  const [name, ...domain] = line.split(",");
                  return { name: name ?? "", domain: domain.join(",").trim() };
                }),
            });
          }}
        />
      </label>
      <div className="flex flex-wrap items-center gap-4">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={config.enabled}
            disabled={!providerConfigured}
            onChange={(e) => update({ enabled: e.target.checked })}
          />{" "}
          Automatic checks
        </label>
        <label className="text-sm">
          Frequency{" "}
          <select
            className={selectClass}
            value={config.interval}
            onChange={(e) =>
              update({
                interval:
                  e.target.value === "daily"
                    ? "daily"
                    : e.target.value === "monthly"
                      ? "monthly"
                      : "weekly",
              })
            }
          >
            <option value="weekly">Weekly</option>
            <option value="daily">Daily</option>
            <option value="monthly">Monthly</option>
          </select>
        </label>
      </div>
      <CostNote
        estimate={reviewed ? review?.estimate : undefined}
        approvedCredits={approvedCredits}
      />
      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error.message}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          disabled={estimate.isPending}
          onClick={() => estimate.mutate(config)}
        >
          Review cost
        </Button>
        <Button
          disabled={!reviewed || save.isPending}
          onClick={() => save.mutate()}
        >
          Save questions & schedule
        </Button>
        <Button
          variant="outline"
          disabled={
            approvedCredits <= 0 ||
            !unchanged ||
            !providerConfigured ||
            run.isPending
          }
          onClick={() => run.mutate()}
        >
          Run approved check
        </Button>
      </div>
      {!providerConfigured ? (
        <p role="status" className="text-sm text-muted-foreground">
          Answer collection is unavailable. You can prepare questions and view
          saved history.
        </p>
      ) : null}
    </section>
  );
}

/** What a check may cost: the fresh review, or the credits already approved. */
function CostNote({
  estimate,
  approvedCredits,
}: {
  estimate: TrackingEstimate | undefined;
  approvedCredits: number;
}) {
  if (estimate) {
    return (
      <p className="rounded-lg bg-muted p-3 text-sm">
        Up to{" "}
        <strong>{estimate.credits.toLocaleString()} credits per check</strong>{" "}
        for {estimate.answers} answers. Estimated scheduled ceiling:{" "}
        {estimate.monthlyCredits.toLocaleString()} credits/month. Actual usage
        settles after collection and is shown after each check. All tools share
        the workspace balance.
      </p>
    );
  }
  if (approvedCredits <= 0) return null;
  return (
    <p className="rounded-lg bg-muted p-3 text-sm">
      Approved: up to{" "}
      <strong>{approvedCredits.toLocaleString()} credits per check</strong>.
    </p>
  );
}
