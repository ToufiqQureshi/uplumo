import { createFileRoute } from "@tanstack/react-router";
import { ContentIdeasPage } from "@/client/features/insights/ContentIdeasPage";
export const Route = createFileRoute("/_app/p/$projectId/content-ideas")({
  component: InsightRoute,
});
function InsightRoute() {
  return <ContentIdeasPage projectId={Route.useParams().projectId} />;
}
