import { createFileRoute } from "@tanstack/react-router";
import { AiOpportunitiesPage } from "@/client/features/insights/AiOpportunitiesPage";

export const Route = createFileRoute(
  "/_app/p/$projectId/ai-visibility/opportunities",
)({
  component: function Section() {
    return <AiOpportunitiesPage projectId={Route.useParams().projectId} />;
  },
});
