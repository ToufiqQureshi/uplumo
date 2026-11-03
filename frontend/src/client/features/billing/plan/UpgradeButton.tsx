import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { ErrorState } from "@/client/components/ErrorState";
import { Button } from "@/client/components/ui/button";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { ApiError } from "@/client/lib/uplumoApi";
import {
  billingPlansQueryOptions,
  billingStatusQueryOptions,
  createCheckout,
  formatPrice,
  waitForPaidPlan,
  type Currency,
} from "./planApi";
import { openRazorpayCheckout } from "./razorpay";

/**
 * Pays for a plan through Razorpay at the price shown, in the currency shown.
 */
export function UpgradeButton({
  plan,
  name,
  price,
  currency,
}: {
  plan: "basic" | "pro" | "business";
  name: string;
  price: number;
  currency: Currency;
}) {
  const queryClient = useQueryClient();
  const [activating, setActivating] = useState(false);
  const upgrade = useMutation({
    mutationFn: async () => {
      const paid = await openRazorpayCheckout(
        await createCheckout(plan, currency),
        name,
      );
      if (!paid) return null;
      setActivating(true);
      try {
        return await waitForPaidPlan();
      } finally {
        setActivating(false);
      }
    },
    meta: { errorToast: false },
    // Never retried: a second attempt could start a second payment.
    retry: false,
    onError: async (error) => {
      // The price shown is no longer what checkout would charge (the
      // customer's location changed): show the new price before any payment.
      if (error instanceof ApiError && error.code === "currency_changed") {
        await queryClient.invalidateQueries({
          queryKey: billingStatusQueryOptions.queryKey,
        });
        await queryClient.invalidateQueries({
          queryKey: billingPlansQueryOptions.queryKey,
        });
      }
    },
    onSuccess: async (status) => {
      if (!status) return;
      if (status.plan === "free") {
        toast.success(
          `Payment received. Uplumo ${name} turns on within a few minutes.`,
        );
      } else {
        toast.success(`Payment received. Welcome to Uplumo ${name}!`);
      }
      queryClient.setQueryData(billingStatusQueryOptions.queryKey, status);
      // Every screen reads the plan and credits; refresh them all.
      await queryClient.invalidateQueries();
    },
  });

  return (
    <div className="w-full space-y-3">
      <Button
        data-uplumo-event={`subscription_checkout_${plan}`}
        onClick={() => upgrade.mutate()}
        disabled={upgrade.isPending}
      >
        {activating
          ? "Activating your plan…"
          : upgrade.isPending
            ? "Opening checkout…"
            : `Choose ${name} · ${formatPrice(price, currency)}/month`}
      </Button>
      {upgrade.isError ? (
        <ErrorState
          variant="inline"
          message={getStandardErrorMessage(upgrade.error)}
        />
      ) : null}
    </div>
  );
}
