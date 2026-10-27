import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";
import { Button } from "@/client/components/ui/button";
import { openTopUpCheckout } from "../plan/razorpay";

const packsSchema = z.array(
  z.object({
    id: z.string(),
    name: z.string(),
    priceInr: z.number(),
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

function lifetime(pack: Pack) {
  return pack.sites > 0
    ? `Adds ${pack.sites} websites for 30 days from payment.`
    : "Used after your monthly credits. Lasts while your plan is active and ends with it.";
}

/**
 * One-time credit packs (Pro and Business) and the site add-on (Business).
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
        { packId },
      );
      // A closed window leaves the order unpaid; the next click reuses it.
      if (await openTopUpCheckout(order)) setPaid(order.id);
    },
    retry: false,
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
        Could not load credit packs.{" "}
        <Button variant="ghost" onClick={() => void packs.refetch()}>
          Retry
        </Button>
      </p>
    );
  if (!packs.data?.length) return null;
  const waiting = Boolean(paid) && !granted;
  return (
    <section className="space-y-3 rounded-xl border border-border p-5">
      <h2 className="text-lg font-semibold">Buy more credits</h2>
      <p className="text-sm text-muted-foreground">
        One-time payment, not a subscription. Every paid action spends credits
        at its real cost. Packs are not refundable.
      </p>
      <div className="flex flex-wrap gap-3">
        {packs.data.map((p) => (
          <div
            key={p.id}
            className="max-w-xs space-y-2 rounded-lg border border-border p-4"
          >
            <p className="font-medium">{p.name}</p>
            <p>₹{p.priceInr.toLocaleString("en-IN")}</p>
            <p className="text-xs text-muted-foreground">{lifetime(p)}</p>
            <Button
              disabled={!canManageBilling || checkout.isPending || waiting}
              onClick={() => checkout.mutate(p.id)}
            >
              {p.sites > 0 ? "Add websites" : "Buy credits"}
            </Button>
          </div>
        ))}
      </div>
      {!canManageBilling ? (
        <p className="text-sm">Ask a workspace owner or admin to purchase.</p>
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
