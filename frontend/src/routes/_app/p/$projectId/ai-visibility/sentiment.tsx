import { createFileRoute } from "@tanstack/react-router";
import { AiSentimentPage } from "@/client/features/insights/AiSentimentPage";

export const Route = createFileRoute(
  "/_app/p/$projectId/ai-visibility/sentiment",
)({
  component: function Section() {
    return <AiSentimentPage projectId={Route.useParams().projectId} />;
  },
});
