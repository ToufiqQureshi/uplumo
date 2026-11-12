import { createFileRoute, redirect } from "@tanstack/react-router";

// Merged into the AI visibility page. Old links and bookmarks keep working.
export const Route = createFileRoute("/_app/p/$projectId/prompt-tracking")({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: "/p/$projectId/ai-visibility",
      params: { projectId: params.projectId },
      replace: true,
    });
  },
});
