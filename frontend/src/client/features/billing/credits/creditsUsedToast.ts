import { toast } from "sonner";
import { onCreditsUsed } from "@/client/lib/uplumoApi";
import { formatCredits } from "@/client/features/billing/credits/format";
import { billingUsageQueryKey } from "@/client/features/billing/usageApi";
import { queryClient } from "@/client/tanstack-db/queryClient";

/**
 * Tells the customer how many credits an action just used, and refreshes the
 * balance shown next to paid actions. Returns the function that stops it.
 */
export function showCreditsUsed() {
  return onCreditsUsed((credits) => {
    toast(`${formatCredits(credits)} credits used`, {
      description: "Charged at the real cost of the data you asked for.",
    });
    void queryClient.invalidateQueries({ queryKey: billingUsageQueryKey });
  });
}
