import { createFileRoute, redirect } from "@tanstack/react-router";

// Merged into the AI visibility page. Old links and bookmarks keep working.
export const Route = createFileRoute("/_app/p/$projectId/ai-sentiment")({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: "/p/$projectId/ai-visibility/sentiment",
      params: { projectId: params.projectId },
      replace: true,
    });
  },
});
