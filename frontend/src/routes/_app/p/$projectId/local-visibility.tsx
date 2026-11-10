import { createFileRoute } from "@tanstack/react-router";
import { LocalVisibilityPage } from "@/client/features/local-visibility/LocalVisibilityPage";
export const Route = createFileRoute("/_app/p/$projectId/local-visibility")({
  component: LocalVisibilityRoute,
});
function LocalVisibilityRoute() {
  return <LocalVisibilityPage projectId={Route.useParams().projectId} />;
}
