import { AUTUMN_PAID_PLAN_FEATURE_ID } from "@/shared/billing";

export type PlanStatus = "free" | "paid";

// The slice of the Autumn customer we read. Structural so tests and both
// the SDK's React and server shapes fit without casts.
type CustomerLike = {
  flags?: Record<string, { planId?: string | null } | undefined>;
};

export function getCustomerPlanStatus(
  customer: CustomerLike | undefined,
): PlanStatus {
  return customer?.flags?.[AUTUMN_PAID_PLAN_FEATURE_ID] ? "paid" : "free";
}
