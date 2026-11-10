import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { ApiError, apiRequest } from "@/client/lib/uplumoApi";
import { Button } from "@/client/components/ui/button";
import {
  billingStatusQueryOptions,
  formatPrice,
  priceIn,
} from "../plan/planApi";
import { ErrorState } from "@/client/components/ErrorState";
import { openTopUpCheckout, paymentFailedMessage } from "../plan/razorpay";

const packsSchema = z.array(
  z.object({
    id: z.string(),
    name: z.string(),
    priceInr: z.number(),
    priceUsd: z.number(),
    credits: z.number(),
    sites: z.number(),
  }),
);
const checkoutSchema = z.object({
  id: z.string(),
  packId: z.string(),
  orderId: z.string(),
  keyId: z.string(),
  amount: z.number(),
  currency: z.string(),
  credits: z.number(),
  sites: z.number(),
  status: z.string(),
});

type Pack = z.infer<typeof packsSchema>[number];

export function lifetime(pack: Pack) {
  if (pack.sites === 1)
    return "Adds 1 website for 30 days from payment. Buy one for each extra website.";
  return pack.sites > 0
    ? `Adds ${pack.sites} websites for 30 days from payment.`
    : "Used after your monthly credits. Lasts while your plan is active and ends with it.";
}

/** The card's heading names only what this plan can buy. */
export function packsHeading(packs: Pack[]) {
  const credits = packs.some((p) => p.credits > 0);
  const sites = packs.some((p) => p.sites > 0);
  if (credits && sites) return "Buy more credits or websites";
  return sites ? "Add more websites" : "Buy more credits";
}

/**
 * One-time credit packs (Pro and Business) and the per-website add-on
 * (every paid plan).
 * The server lists only what the workspace's plan may buy. A payment the
 * customer completed is only waited on, never offered again, so one
 * purchase cannot be paid twice.
 */
export function TopUpsCard({
  canManageBilling,
  enabled,
}: {
  canManageBilling: boolean;
  enabled: boolean;
}) {
  const queryClient = useQueryClient();
  // The purchase whose payment Razorpay reported as completed, waiting for
  // the signed webhook to grant it.
  const [paid, setPaid] = useState<string | null>(null);
  // Packs are charged in the currency a checkout from here is billed in.
  const billing = useQuery({ ...billingStatusQueryOptions, enabled });
  const currency = billing.data?.checkoutCurrency;
  const packs = useQuery({
    queryKey: ["billing", "topups"],
    queryFn: () => apiRequest("/api/v1/billing/topups", packsSchema),
    enabled,
    staleTime: 60000,
  });
  const checkout = useMutation({
    mutationFn: async (packId: string) => {
      const order = await apiRequest(
        "/api/v1/billing/topups/checkout",
        checkoutSchema,
        "POST",
        { packId, currency },
      );
      const pack = packs.data?.find((p) => p.id === packId);
      // A closed window leaves the order unpaid; the next click reuses it.
      const outcome = await openTopUpCheckout(order, pack?.name ?? "Uplumo");
      if (outcome.status === "paid") setPaid(order.id);
      return { ...outcome, packId, packName: pack?.name ?? "Your purchase" };
    },
    retry: false,
    onError: (error) => {
      // The location changed since the price was shown: show the new one.
      if (error instanceof ApiError && error.code === "currency_changed") {
        void queryClient.invalidateQueries({
          queryKey: billingStatusQueryOptions.queryKey,
        });
      }
    },
  });
  const status = useQuery({
    queryKey: ["billing", "topup", paid],
    queryFn: () =>
      apiRequest(
        `/api/v1/billing/topups/${encodeURIComponent(paid!)}`,
        z.object({ status: z.string() }),
      ),
    enabled: Boolean(paid),
    refetchInterval: (q) =>
      q.state.data?.status === "paid" || q.state.dataUpdateCount >= 40
        ? false
        : 3000,
  });
  const granted = status.data?.status === "paid";
  useEffect(() => {
    if (granted) {
      void queryClient.invalidateQueries({ queryKey: ["billing", "usage"] });
      void queryClient.invalidateQueries({ queryKey: ["billing", "credits"] });
      void queryClient.invalidateQueries({ queryKey: ["goBillingStatus"] });
    }
  }, [granted, queryClient]);
  if (!enabled) return null;
  if (packs.isError)
    return (
      <p role="alert" className="text-sm">
        Could not load packs.{" "}
        <Button variant="ghost" onClick={() => void packs.refetch()}>
          Retry
        </Button>
      </p>
    );
  if (!packs.data?.length || !currency) return null;
  const waiting = Boolean(paid) && !granted;
  const failed = checkout.data?.status === "failed" ? checkout.data : null;
  return (
    <section className="space-y-3 rounded-xl border border-border p-5">
      <h2 className="text-lg font-semibold">{packsHeading(packs.data)}</h2>
      <p className="text-sm text-muted-foreground">
        One-time payment, not a subscription.
        {packs.data.some((p) => p.credits > 0)
          ? " Every paid action spends credits at its real cost."
          : ""}{" "}
        Packs are not refundable.
      </p>
      <div className="flex flex-wrap gap-3">
        {packs.data.map((p) => (
          <div
            key={p.id}
            className="max-w-xs space-y-2 rounded-lg border border-border p-4"
          >
            <p className="font-medium">{p.name}</p>
            <p>{formatPrice(priceIn(p, currency), currency)}</p>
            <p className="text-xs text-muted-foreground">{lifetime(p)}</p>
            <Button
              disabled={!canManageBilling || checkout.isPending || waiting}
              onClick={() => checkout.mutate(p.id)}
            >
              {p.sites > 0 ? "Add a website" : "Buy credits"}
            </Button>
          </div>
        ))}
      </div>
      {!canManageBilling ? (
        <p className="text-sm">Ask a workspace owner or admin to purchase.</p>
      ) : null}
      {failed ? (
        <ErrorState
          title="Payment didn't go through"
          message={paymentFailedMessage(failed.reason, failed.packName)}
          onRetry={() => checkout.mutate(failed.packId)}
        />
      ) : null}
      {checkout.error ? (
        <p role="alert" className="text-sm text-destructive">
          {checkout.error.message}
        </p>
      ) : null}
      {paid ? (
        <div role="status" className="space-y-2 text-sm">
          <p>
            {granted
              ? "Payment verified. Your pack is active."
              : "Payment received. Waiting for Razorpay to confirm it; this usually takes under a minute. Do not pay again."}
          </p>
          {granted ? (
            <Button variant="ghost" onClick={() => setPaid(null)}>
              Buy another pack
            </Button>
          ) : (
            <Button
              variant="ghost"
              disabled={status.isFetching}
              onClick={() => void status.refetch()}
            >
              Check payment status
            </Button>
          )}
        </div>
      ) : null}
    </section>
  );
}
