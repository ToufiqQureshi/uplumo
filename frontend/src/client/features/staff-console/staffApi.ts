import { queryOptions } from "@tanstack/react-query";
import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const money = z.object({ currency: z.string(), amount: z.number() });
const count = z.object({ label: z.string(), count: z.number() });
export type Money = z.infer<typeof money>;
export type Count = z.infer<typeof count>;

const overviewSchema = z.object({
  users: z.number(),
  verifiedUsers: z.number(),
  blockedUsers: z.number(),
  newToday: z.number(),
  new7Days: z.number(),
  new30Days: z.number(),
  payingWorkspaces: z.number(),
  planCounts: z.array(count),
  mrr: z.array(money),
  collectedThisMonth: z.array(money),
  collectedLastMonth: z.array(money),
  cancelledLast30Days: z.number(),
  endingNext7Days: z.number(),
  paymentFailing: z.number(),
  signupsByDay: z.array(z.object({ day: z.string(), count: z.number() })),
});

const subscriptionSchema = z.object({
  organizationId: z.string(),
  workspace: z.string(),
  ownerEmail: z.string(),
  plan: z.string(),
  currency: z.string(),
  status: z.string(),
  currentPeriodEnd: z.string().nullable(),
  cancelAtPeriodEnd: z.boolean(),
  updatedAt: z.string(),
  paidTotal: z.number(),
});
export type StaffSubscription = z.infer<typeof subscriptionSchema>;

const marketingSchema = z.object({
  foundVia: z.array(count),
  interests: z.array(count),
  workFor: z.array(count),
  funnel: z.array(count),
});

const financeSchema = z.object({
  revenue: z.array(
    z.object({
      month: z.string(),
      currency: z.string(),
      gross: z.number(),
      refunded: z.number(),
      payments: z.number(),
    }),
  ),
  providerCost: z.array(z.object({ month: z.string(), costUsd: z.number() })),
  costByFeature30Days: z.array(
    z.object({ feature: z.string(), costUsd: z.number() }),
  ),
  topCostWorkspaces30Days: z.array(
    z.object({
      organizationId: z.string(),
      workspace: z.string(),
      ownerEmail: z.string(),
      plan: z.string(),
      costUsd: z.number(),
    }),
  ),
});

const actionSchema = z.object({
  actorEmail: z.string(),
  action: z.string(),
  targetEmail: z.string(),
  targetId: z.string(),
  detail: z.string(),
  createdAt: z.string(),
});

// Staff numbers are read-only and free; a failed load is shown, not retried.
const staffQuery = <T>(key: string, path: string, schema: z.ZodType<T>) =>
  queryOptions({
    queryKey: ["staff", key],
    queryFn: () => apiRequest(path, schema),
    retry: false,
  });

export const overviewQueryOptions = () =>
  staffQuery("overview", "/api/v1/staff/overview", overviewSchema);
export const subscriptionsQueryOptions = () =>
  staffQuery(
    "subscriptions",
    "/api/v1/staff/subscriptions",
    z.object({ subscriptions: z.array(subscriptionSchema) }),
  );
export const marketingQueryOptions = () =>
  staffQuery("marketing", "/api/v1/staff/marketing", marketingSchema);
export const financeQueryOptions = () =>
  staffQuery("finance", "/api/v1/staff/finance", financeSchema);
export const actionsQueryOptions = () =>
  staffQuery(
    "actions",
    "/api/v1/staff/actions",
    z.object({ actions: z.array(actionSchema) }),
  );

const failedPaymentSchema = z.object({
  id: z.string(),
  email: z.string(),
  method: z.string(),
  amount: z.number(),
  currency: z.string(),
  reason: z.string(),
  code: z.string(),
  at: z.string(),
  paidLater: z.boolean(),
});

/** Read live from Razorpay when the page opens; Razorpay API reads are free. */
export const failedPaymentsQueryOptions = () =>
  staffQuery(
    "failed-payments",
    "/api/v1/staff/payments/failed",
    z.object({
      since: z.string(),
      payments: z.array(failedPaymentSchema),
      truncated: z.boolean(),
    }),
  );

export type UserAction = "block" | "unblock" | "delete";

export function runUserAction(
  action: UserAction,
  input: { userId: string; reason?: string; confirmEmail?: string },
) {
  return apiRequest(
    `/api/v1/staff/users/${action}`,
    z.object({ ok: z.boolean() }),
    "POST",
    input,
  );
}

export function syncPayments() {
  return apiRequest(
    "/api/v1/staff/payments/sync",
    z.object({
      subscriptions: z.number(),
      added: z.number(),
      failed: z.number(),
      refundsUpdated: z.number(),
    }),
    "POST",
    {},
  );
}

/** Formats subunits (paise or cents) as whole currency, e.g. ₹2,499. */
export function formatMoney(amount: number, currency: string) {
  return new Intl.NumberFormat(currency === "INR" ? "en-IN" : "en-US", {
    style: "currency",
    currency,
    maximumFractionDigits: 0,
  }).format(amount / 100);
}

export function formatUSD(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 2,
  }).format(value);
}

export const dateFormat = new Intl.DateTimeFormat("en-IN", {
  dateStyle: "medium",
});

export const dateTimeFormat = new Intl.DateTimeFormat("en-IN", {
  dateStyle: "medium",
  timeStyle: "short",
});
