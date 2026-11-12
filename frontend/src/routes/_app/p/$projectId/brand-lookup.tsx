import { createFileRoute, redirect } from "@tanstack/react-router";
import { brandLookupSearchSchema } from "@/types/schemas/ai-search";

// Merged into the AI visibility page. Old links and bookmarks keep working.
export const Route = createFileRoute("/_app/p/$projectId/brand-lookup")({
  validateSearch: brandLookupSearchSchema,
  beforeLoad: ({ params, search }) => {
    throw redirect({
      to: "/p/$projectId/ai-visibility/research",
      params: { projectId: params.projectId },
      search,
      replace: true,
    });
  },
});
