import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { EnquiriesPage } from "@/client/features/analytics/EnquiriesPage";
export const Route = createFileRoute("/_app/p/$projectId/enquiries")({
  validateSearch: z.object({
    path: z.string().max(2048).catch("").default(""),
  }),
  component: EnquiriesRoute,
});
function EnquiriesRoute() {
  const { projectId } = Route.useParams();
  const { path } = Route.useSearch();
  return (
    <EnquiriesPage
      key={`${projectId}:${path}`}
      projectId={projectId}
      initialPath={path}
    />
  );
}
