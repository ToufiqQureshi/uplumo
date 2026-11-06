import { createFileRoute } from "@tanstack/react-router";
import { MarketingPage } from "@/client/features/staff-console/MarketingPage";

export const Route = createFileRoute("/staff/marketing")({
  component: MarketingPage,
});
