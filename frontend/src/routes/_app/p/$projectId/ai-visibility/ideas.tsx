import { createFileRoute } from "@tanstack/react-router";
import { PromptIdeasPage } from "@/client/features/insights/PromptIdeasPage";

export const Route = createFileRoute("/_app/p/$projectId/ai-visibility/ideas")({
  component: function Section() {
    return <PromptIdeasPage projectId={Route.useParams().projectId} />;
  },
});
