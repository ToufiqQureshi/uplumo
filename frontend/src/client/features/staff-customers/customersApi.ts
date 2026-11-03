import { queryOptions } from "@tanstack/react-query";
import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const customerSchema = z.object({
  userId: z.string(),
  email: z.string(),
  name: z.string(),
  emailVerified: z.boolean(),
  signedUpAt: z.string(),
  workspace: z.string(),
  role: z.string(),
  plan: z.string(),
  subscriptionStatus: z.string(),
  activeSites: z.number(),
  domains: z.array(z.string()),
  gscConnected: z.boolean(),
  ga4Connected: z.boolean(),
  interestedFeatures: z.string(),
  workFor: z.string(),
  clientWebsiteCount: z.string(),
  foundVia: z.string(),
  mcpSetupIntent: z.string(),
  onboardingCompletedAt: z.string(),
});

export type Customer = z.infer<typeof customerSchema>;

export const CUSTOMERS_CSV_URL = "/api/v1/admin/customers.csv";

export const staffAccessQueryOptions = () =>
  queryOptions({
    queryKey: ["staffAccess"],
    queryFn: () =>
      apiRequest("/api/v1/admin/access", z.object({ staff: z.boolean() })),
    staleTime: Infinity,
    retry: false,
  });

export const customersQueryOptions = () =>
  queryOptions({
    queryKey: ["staffCustomers"],
    queryFn: () =>
      apiRequest(
        "/api/v1/admin/customers",
        z.object({ customers: z.array(customerSchema) }),
      ),
    retry: false,
  });

/** Saved interests are a JSON array; show them as a readable list. */
export function interestsText(value: string): string {
  try {
    const parsed: unknown = JSON.parse(value);
    if (Array.isArray(parsed)) return parsed.map(String).join(", ");
  } catch {
    // Older rows may hold plain text.
  }
  return value;
}
