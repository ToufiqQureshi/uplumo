import { createFileRoute } from "@tanstack/react-router";
import { AiOpportunitiesPage } from "@/client/features/insights/AiOpportunitiesPage";
export const Route = createFileRoute("/_app/p/$projectId/ai-opportunities")({
  component: InsightRoute,
});
function InsightRoute() {
  return <AiOpportunitiesPage projectId={Route.useParams().projectId} />;
}
