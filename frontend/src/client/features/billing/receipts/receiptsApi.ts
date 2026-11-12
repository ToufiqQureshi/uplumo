import { queryOptions } from "@tanstack/react-query";
import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const paymentSchema = z.object({
  paymentId: z.string(),
  receiptNo: z.string(),
  kind: z.enum(["subscription", "topup"]),
  item: z.string(),
  /** In the currency's subunits (paise, cents). */
  amount: z.number(),
  refunded: z.number(),
  currency: z.enum(["INR", "USD"]),
  paidAt: z.string(),
});

export const billingDetailsSchema = z.object({
  name: z.string(),
  address: z.string(),
  country: z.string(),
  taxId: z.string(),
});

const receiptSchema = paymentSchema.extend({ billedTo: billingDetailsSchema });

type Payment = z.infer<typeof paymentSchema>;
export type BillingDetails = z.infer<typeof billingDetailsSchema>;

// Read from Uplumo's own payment ledger: free, nothing paid is called.
export const paymentsQueryOptions = queryOptions({
  queryKey: ["billing", "payments"],
  queryFn: async () =>
    (
      await apiRequest(
        "/api/v1/billing/payments",
        z.object({ payments: z.array(paymentSchema) }),
      )
    ).payments,
  staleTime: 60_000,
});

export const billingDetailsQueryOptions = queryOptions({
  queryKey: ["billing", "details"],
  queryFn: () => apiRequest("/api/v1/billing/details", billingDetailsSchema),
  staleTime: 60_000,
});

export function receiptQueryOptions(paymentId: string) {
  return queryOptions({
    queryKey: ["billing", "receipt", paymentId],
    queryFn: () =>
      apiRequest(
        `/api/v1/billing/payments/${encodeURIComponent(paymentId)}/receipt`,
        receiptSchema,
      ),
    retry: false,
  });
}

export function saveBillingDetails(details: BillingDetails) {
  return apiRequest(
    "/api/v1/billing/details",
    billingDetailsSchema,
    "POST",
    details,
  );
}

/** Formats an amount in subunits: 249900 INR is ₹2,499, 1950 USD is $19.50. */
export function formatAmount(subunits: number, currency: Payment["currency"]) {
  return new Intl.NumberFormat(currency === "INR" ? "en-IN" : "en-US", {
    style: "currency",
    currency,
    minimumFractionDigits: subunits % 100 === 0 ? 0 : 2,
  }).format(subunits / 100);
}

export function formatPaidOn(paidAt: string) {
  return new Date(paidAt).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/** The payment's status as a receipt shows it. */
export function refundStatus(p: Pick<Payment, "amount" | "refunded">) {
  if (p.refunded <= 0) return "Paid";
  return p.refunded >= p.amount ? "Refunded" : "Part refunded";
}
