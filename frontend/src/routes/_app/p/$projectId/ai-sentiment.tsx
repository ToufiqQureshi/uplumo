import { createFileRoute } from "@tanstack/react-router";
import { AiSentimentPage } from "@/client/features/insights/AiSentimentPage";
export const Route = createFileRoute("/_app/p/$projectId/ai-sentiment")({
  component: InsightRoute,
});
function InsightRoute() {
  return <AiSentimentPage projectId={Route.useParams().projectId} />;
}
