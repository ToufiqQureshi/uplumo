import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AiPromptsPage } from "@/client/features/ai-search/AiPromptsPage";

const searchSchema = z.object({
  show: z.enum(["all", "gaps", "named"]).optional().catch(undefined),
  /** A question and engine whose saved answer opens on arrival. */
  q: z.string().max(500).optional().catch(undefined),
  e: z.string().max(40).optional().catch(undefined),
});

export const Route = createFileRoute(
  "/_app/p/$projectId/ai-visibility/prompts",
)({
  validateSearch: searchSchema,
  remountDeps: ({ params }) => params.projectId,
  component: function Prompts() {
    const { projectId } = Route.useParams();
    const { show = "all", q, e } = Route.useSearch();
    const navigate = Route.useNavigate();
    return (
      <AiPromptsPage
        projectId={projectId}
        show={show}
        initialOpen={q && e ? { prompt: q, engine: e } : null}
        onShowChange={(next) =>
          void navigate({
            search: (prev) => ({
              ...prev,
              show: next === "all" ? undefined : next,
            }),
            replace: true,
          })
        }
      />
    );
  },
});
