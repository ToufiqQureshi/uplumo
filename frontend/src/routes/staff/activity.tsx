import { createFileRoute } from "@tanstack/react-router";
import { ActivityPage } from "@/client/features/staff-console/ActivityPage";

export const Route = createFileRoute("/staff/activity")({
  component: ActivityPage,
});
