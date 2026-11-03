import { createFileRoute } from "@tanstack/react-router";
import { OverviewPage } from "@/client/features/staff-console/OverviewPage";

export const Route = createFileRoute("/staff/")({
  component: OverviewPage,
});
