import { createFileRoute } from "@tanstack/react-router";
import { BacklinkGapPage } from "@/client/features/insights/BacklinkGapPage";
export const Route = createFileRoute("/_app/p/$projectId/backlink-opportunity")(
  {
    component: InsightRoute,
  },
);
function InsightRoute() {
  return <BacklinkGapPage projectId={Route.useParams().projectId} />;
}
