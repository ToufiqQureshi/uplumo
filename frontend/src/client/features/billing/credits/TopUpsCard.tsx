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
    expiresAfterDays: z.number(),
  }),
);
const checkoutSchema = z.object({
  id: z.string(),
  orderId: z.string(),
  keyId: z.string(),
  amount: z.number(),
  currency: z.string(),
  credits: z.number(),
  expiresAfterDays: z.number(),
  status: z.string(),
});
export function TopUpsCard({
  canManageBilling,
  enabled,
}: {
  canManageBilling: boolean;
  enabled: boolean;
}) {
  const queryClient = useQueryClient();
  const [payment, setPayment] = useState<string | null>(null);
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
      setPayment(order.id);
      await openTopUpCheckout(order);
    },
  });
  const status = useQuery({
    queryKey: ["billing", "topup", payment],
    queryFn: () =>
      apiRequest(
        `/api/v1/billing/topups/${encodeURIComponent(payment!)}`,
        z.object({ status: z.string() }),
      ),
    enabled: Boolean(payment),
    refetchInterval: (q) =>
      q.state.data?.status === "paid" ||
      q.state.data?.status === "failed" ||
      q.state.dataUpdateCount >= 40
        ? false
        : 3000,
  });
  useEffect(() => {
    if (status.data?.status === "paid") {
      void queryClient.invalidateQueries({ queryKey: ["billing", "usage"] });
      void queryClient.invalidateQueries({ queryKey: ["billing", "credits"] });
    }
  }, [status.data?.status, queryClient]);
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
  return (
    <section className="space-y-3 rounded-xl border p-5">
      <h2 className="text-lg font-semibold">Add usage credits</h2>
      <p className="text-sm text-muted-foreground">
        One-time purchase for an active paid plan. Your current feature and
        daily spending limits still apply. Credits appear after payment is
        verified.
      </p>
      <div className="flex flex-wrap gap-3">
        {packs.data.map((p) => (
          <div key={p.id} className="space-y-2 rounded-lg border p-4">
            <p className="font-medium">{p.name}</p>
            <p>
              {p.credits.toLocaleString()} credits · ₹
              {p.priceInr.toLocaleString("en-IN")}
            </p>
            <p className="text-xs text-muted-foreground">
              Valid for {p.expiresAfterDays} days after purchase.
            </p>
            <Button
              disabled={
                !canManageBilling ||
                checkout.isPending ||
                (Boolean(payment) &&
                  status.data?.status !== "paid" &&
                  status.data?.status !== "failed")
              }
              onClick={() => checkout.mutate(p.id)}
            >
              Buy credits
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
      {payment ? (
        <p role="status" className="text-sm">
          {status.data?.status === "paid"
            ? "Payment verified. Credits have been added."
            : "Waiting for payment verification. If you closed checkout, reopen it to finish."}
          {status.data?.status !== "paid" ? (
            <Button
              variant="ghost"
              disabled={status.isFetching}
              onClick={() => void status.refetch()}
            >
              Check payment status
            </Button>
          ) : null}
          <Button
            variant="ghost"
            disabled={checkout.isPending || status.data?.status === "paid"}
            onClick={() => {
              setPayment(null);
              checkout.reset();
            }}
          >
            Choose pack again
          </Button>
        </p>
      ) : null}
    </section>
  );
}
