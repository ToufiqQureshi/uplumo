import { useState, type ReactNode } from "react";
import { ConfirmDialog } from "@/client/components/ConfirmDialog";
import { formatCredits } from "@/client/features/billing/credits/format";
import {
  creditLevel,
  useBillingUsage,
  type CreditFeature,
} from "@/client/features/billing/usageApi";

/** An action that typically costs at least this many credits asks first. */
const CONFIRM_FROM_CREDITS = 300;

/**
 * Asks the customer before an action that is expensive in credits (a brand
 * lookup is about 370). Cheaper actions, and servers where credits are not
 * charged, run at once. Call \`request(run)\` instead of running the action,
 * and render \`dialog\` anywhere on the page.
 */
export function useConfirmedCost(
  feature: CreditFeature,
  action: string,
  what: string,
): { request: (run: () => void) => void; dialog: ReactNode } {
  const { data: usage } = useBillingUsage();
  const [pending, setPending] = useState<(() => void) | null>(null);
  const credits =
    usage?.costs.find((c) => c.feature === feature && c.action === action)
      ?.credits ?? 0;
  const asks =
    usage !== undefined &&
    creditLevel(usage) !== "not_charged" &&
    credits >= CONFIRM_FROM_CREDITS;

  const request = (run: () => void) => {
    if (asks) setPending(() => run);
    else run();
  };

  const dialog = pending ? (
    <ConfirmDialog
      title={`Use about ${formatCredits(credits)} credits?`}
      confirmLabel="Continue"
      onClose={() => setPending(null)}
      onConfirm={() => {
        const run = pending;
        setPending(null);
        run();
      }}
    >
      This {what} uses about {formatCredits(credits)} credits
      {usage ? ` of your ${formatCredits(usage.balanceCredits)} left` : ""}. You
      are charged the real cost of the data, which can be a little more or less.
    </ConfirmDialog>
  ) : null;

  return { request, dialog };
}
