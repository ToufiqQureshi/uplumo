import { createFileRoute } from "@tanstack/react-router";
import { SubscriptionsPage } from "@/client/features/staff-console/SubscriptionsPage";

export const Route = createFileRoute("/staff/subscriptions")({
  component: SubscriptionsPage,
});
