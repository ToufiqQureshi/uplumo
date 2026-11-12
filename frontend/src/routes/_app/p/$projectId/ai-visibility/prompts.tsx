import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AiPromptsPage } from "@/client/features/ai-search/AiPromptsPage";

const searchSchema = z.object({
  show: z.enum(["all", "gaps", "named"]).optional().catch(undefined),
});

export const Route = createFileRoute(
  "/_app/p/$projectId/ai-visibility/prompts",
)({
  validateSearch: searchSchema,
  remountDeps: ({ params }) => params.projectId,
  component: function Prompts() {
    const { projectId } = Route.useParams();
    const { show = "all" } = Route.useSearch();
    const navigate = Route.useNavigate();
    return (
      <AiPromptsPage
        projectId={projectId}
        show={show}
        onShowChange={(next) =>
          void navigate({
            search: { show: next === "all" ? undefined : next },
            replace: true,
          })
        }
      />
    );
  },
});
