import { createFileRoute } from "@tanstack/react-router";
import { NextActionsPage } from "@/client/features/insights/NextActionsPage";
export const Route = createFileRoute("/_app/p/$projectId/next-actions")({
  component: InsightRoute,
});
function InsightRoute() {
  return <NextActionsPage projectId={Route.useParams().projectId} />;
}
