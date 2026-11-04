import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ConfirmDialog } from "@/client/components/ConfirmDialog";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { Button } from "@/client/components/ui/button";
import { useCopy } from "@/client/hooks/useCopy";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { billingUsageQueryKey } from "@/client/features/billing/usageApi";
import { engineLabel } from "@/client/features/ai-search/platformLabels";
import {
  draftCiteFixPitch,
  type CiteFixPitch,
  type CiteFixProof,
  type CiteFixReport,
} from "./citeFixApi";
import { formatDate } from "./DecisionSections";

/** The lines of saved AI answers that name the customer, in their words. */
export function WhatAISaysCard({ said }: { said: CiteFixProof["said"] }) {
  if (said.length === 0) return null;
  return (
    <section
      aria-labelledby="cite-fix-said"
      className="space-y-3 rounded-2xl border bg-card p-5"
    >
      <div>
        <h2 id="cite-fix-said" className="font-serif text-2xl">
          What AI says about you
        </h2>
        <p className="text-sm text-muted-foreground">
          From the answers that name you in the latest check. Check the facts: a
          wrong one is repeated in every answer. Put the right facts on your
          site so other AIs say them too.
        </p>
      </div>
      <ul className="grid gap-3 md:grid-cols-2">
        {said.map((quote) => (
          <li
            key={`${quote.prompt} ${quote.engine}`}
            className="space-y-2 rounded-xl border p-4"
          >
            <p className="font-mono text-xs text-muted-foreground">
              {engineLabel(quote.engine)} · “{quote.prompt}”
            </p>
            <blockquote className="font-serif text-lg leading-snug">
              “{quote.excerpt}”
            </blockquote>
          </li>
        ))}
      </ul>
    </section>
  );
}

function change(before: number, after: number) {
  return `${before} → ${after}`;
}

/**
 * Each page the agent fixed, and what followed: whether AI now cites it, and
 * AI visitors and their leads for the same number of days before and after.
 */
export function FixProofCard({
  proofs,
}: {
  proofs: CiteFixReport["fixProofs"];
}) {
  if (proofs.length === 0) return null;
  return (
    <section
      aria-labelledby="cite-fix-proof"
      className="space-y-3 rounded-2xl border bg-card p-5"
    >
      <div>
        <h2 id="cite-fix-proof" className="font-serif text-2xl">
          Did your fixes work?
        </h2>
        <p className="text-sm text-muted-foreground">
          Pages the AI agent fixed, compared over the same number of days before
          and after. Likely linked, not proven: AI assistants do not say who
          clicked.
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[36rem] text-sm">
          <thead>
            <tr className="text-left text-xs text-muted-foreground">
              <th className="py-2 font-normal">Page fixed</th>
              <th className="py-2 font-normal">AI cites it now</th>
              <th className="py-2 text-right font-normal">AI visitors</th>
              <th className="py-2 text-right font-normal">Leads</th>
              <th className="py-2 text-right font-normal">Days each side</th>
            </tr>
          </thead>
          <tbody>
            {proofs.map((proof) => (
              <tr key={proof.url} className="border-t">
                <td className="max-w-64 py-2">
                  <SafeExternalLink
                    url={proof.url}
                    label={new URL(proof.url).pathname}
                    className="truncate hover:underline"
                  />
                  <span className="block text-xs text-muted-foreground">
                    fixed {formatDate(proof.fixedAt)}
                  </span>
                </td>
                <td className="py-2">
                  {proof.citedNow ? (
                    <span className="text-success">Yes</span>
                  ) : (
                    <span className="text-muted-foreground">Not yet</span>
                  )}
                </td>
                <td className="py-2 text-right font-mono">
                  {change(proof.aiVisitorsBefore, proof.aiVisitorsAfter)}
                </td>
                <td className="py-2 text-right font-mono">
                  {proof.leadsTracked
                    ? change(proof.leadsBefore, proof.leadsAfter)
                    : "–"}
                </td>
                <td className="py-2 text-right font-mono">{proof.days}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {proofs.some((proof) => !proof.leadsTracked) ? (
        <p className="text-xs text-muted-foreground">
          Leads show once you choose your lead actions in Traffic.
        </p>
      ) : null}
    </section>
  );
}

/**
 * Paid: drafts an email asking a list page's owner to add the customer. The
 * cost cap comes from the server; one click sends one request, never
 * retried, and a page drafted before is shown again for free.
 */
export function PitchButton({
  projectId,
  url,
  maxCredits,
}: {
  projectId: string;
  url: string;
  maxCredits: number;
}) {
  const client = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const { copied, copy } = useCopy();
  const draft = useMutation({
    retry: false,
    meta: { errorToast: false },
    mutationFn: () =>
      draftCiteFixPitch(projectId, {
        url,
        idempotencyKey: crypto.randomUUID(),
      }),
    // Close the dialog either way, so a failure message is not hidden
    // behind it; the balance may have changed even on a failure.
    onSettled: () => {
      setConfirming(false);
      void client.invalidateQueries({ queryKey: billingUsageQueryKey });
    },
  });
  const pitch: CiteFixPitch | undefined = draft.data;
  return (
    <div className="space-y-2">
      {pitch ? null : (
        <Button
          size="sm"
          variant="outline"
          disabled={draft.isPending}
          onClick={() => setConfirming(true)}
        >
          Draft a pitch
          <span className="font-mono text-xs opacity-70">
            ≤{maxCredits} credits
          </span>
        </Button>
      )}
      {draft.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {getStandardErrorMessage(
            draft.error,
            "Could not write the draft. You were not charged.",
          )}
        </p>
      ) : null}
      {pitch ? (
        <div className="space-y-2 rounded-xl border p-4 text-sm">
          <p className="font-medium">Subject: {pitch.subject}</p>
          <p className="whitespace-pre-wrap text-muted-foreground">
            {pitch.body}
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void copy(
                  `Subject: ${pitch.subject}\n\n${pitch.body}`,
                  "Draft copied",
                )
              }
            >
              {copied ? "Copied" : "Copy"}
            </Button>
            <span className="text-xs text-muted-foreground">
              {pitch.cached
                ? "Your earlier draft, shown again for free."
                : `${pitch.credits} credits used. Fill the [brackets] before sending.`}
            </span>
          </div>
        </div>
      ) : null}
      {confirming ? (
        <ConfirmDialog
          title="Draft an outreach email?"
          confirmLabel={`Continue · up to ${maxCredits} credits`}
          pending={draft.isPending}
          onClose={() => setConfirming(false)}
          onConfirm={() => draft.mutate()}
        >
          The AI writes a short email asking this page's owner to add your
          business. It costs the AI's real price, at most {maxCredits} credits.
          If you drafted this page before, you get that draft again for free.
        </ConfirmDialog>
      ) : null}
    </div>
  );
}
