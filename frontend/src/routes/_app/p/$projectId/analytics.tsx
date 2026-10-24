import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AnalyticsWorkspace } from "@/client/features/analytics/AnalyticsWorkspace";

const searchSchema = z.object({
  view: z.enum(["intelligence", "traffic"]).optional().catch(undefined),
  days: z.union([z.literal(7), z.literal(30), z.literal(90)]).catch(30),
});

export const Route = createFileRoute("/_app/p/$projectId/analytics")({
  validateSearch: searchSchema,
  component: AnalyticsRoute,
});

function AnalyticsRoute() {
  const { projectId } = Route.useParams();
  const { days, view } = Route.useSearch();
  const navigate = Route.useNavigate();
  return (
    <AnalyticsWorkspace
      projectId={projectId}
      days={days}
      view={view ?? "traffic"}
      onChange={(update) =>
        void navigate({ search: (previous) => ({ ...previous, ...update }) })
      }
    />
  );
}
