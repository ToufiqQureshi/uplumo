import { createFileRoute } from "@tanstack/react-router";
import { CiteFixPage } from "@/client/features/cite-fix/CiteFixPage";

export const Route = createFileRoute("/_app/p/$projectId/cite-fix")({
  component: CiteFixRoute,
});

function CiteFixRoute() {
  return <CiteFixPage projectId={Route.useParams().projectId} />;
}
