import { createFileRoute } from "@tanstack/react-router";
import { AiOverviewPage } from "@/client/features/ai-search/AiOverviewPage";

export const Route = createFileRoute("/_app/p/$projectId/ai-visibility/")({
  component: function Overview() {
    return <AiOverviewPage projectId={Route.useParams().projectId} />;
  },
});
