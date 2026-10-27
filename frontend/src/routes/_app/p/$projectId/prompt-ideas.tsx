import { createFileRoute } from "@tanstack/react-router";
import { PromptIdeasPage } from "@/client/features/insights/PromptIdeasPage";
export const Route = createFileRoute("/_app/p/$projectId/prompt-ideas")({
  component: InsightRoute,
});
function InsightRoute() {
  return <PromptIdeasPage projectId={Route.useParams().projectId} />;
}
