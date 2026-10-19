import { createFileRoute } from "@tanstack/react-router";
import { PromptTrackingPage } from "@/client/features/ai-search/PromptTrackingPage";
export const Route = createFileRoute("/_app/p/$projectId/prompt-tracking")({
  component: TrackingRoute,
});
function TrackingRoute() {
  return <PromptTrackingPage projectId={Route.useParams().projectId} />;
}
