import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AiVisibilityTabs } from "@/client/features/ai-search/AiVisibilityTabs";
import { PageHeader } from "@/client/components/PageHeader";

export const Route = createFileRoute("/_app/p/$projectId/ai-visibility")({
  component: AiVisibilityLayout,
});

function AiVisibilityLayout() {
  const { projectId } = Route.useParams();
  return (
    <div className="overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-7xl space-y-5">
        <PageHeader
          title="AI visibility"
          description="What ChatGPT, Gemini, Google AI Overview, Perplexity and Claude say when your customers ask, with the answer behind every number."
        />
        <AiVisibilityTabs projectId={projectId} />
        <Outlet />
      </div>
    </div>
  );
}
