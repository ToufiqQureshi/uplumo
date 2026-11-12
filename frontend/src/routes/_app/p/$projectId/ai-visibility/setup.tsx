import { createFileRoute } from "@tanstack/react-router";
import { PromptTrackingPage } from "@/client/features/ai-search/PromptTrackingPage";

export const Route = createFileRoute("/_app/p/$projectId/ai-visibility/setup")({
  component: function Section() {
    return <PromptTrackingPage projectId={Route.useParams().projectId} />;
  },
});
