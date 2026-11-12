import { createFileRoute, redirect } from "@tanstack/react-router";
import { promptExplorerSearchSchema } from "@/types/schemas/ai-search";

// Merged into the AI visibility page. Old links and bookmarks keep working.
export const Route = createFileRoute("/_app/p/$projectId/prompt-explorer")({
  validateSearch: promptExplorerSearchSchema,
  beforeLoad: ({ params, search }) => {
    throw redirect({
      to: "/p/$projectId/ai-visibility/test",
      params: { projectId: params.projectId },
      search,
      replace: true,
    });
  },
});
