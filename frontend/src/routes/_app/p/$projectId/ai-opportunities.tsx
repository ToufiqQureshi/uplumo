import { createFileRoute, redirect } from "@tanstack/react-router";

// Merged into the AI visibility page. Old links and bookmarks keep working.
export const Route = createFileRoute("/_app/p/$projectId/ai-opportunities")({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: "/p/$projectId/ai-visibility/opportunities",
      params: { projectId: params.projectId },
      replace: true,
    });
  },
});
