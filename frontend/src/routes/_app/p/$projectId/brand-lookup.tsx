import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { BrandLookupPage } from "@/client/features/ai-search/BrandLookupPage";
import { brandLookupSearchSchema } from "@/types/schemas/ai-search";

export const Route = createFileRoute("/_app/p/$projectId/brand-lookup")({
  validateSearch: brandLookupSearchSchema,
  // The project switcher keeps this page; filter drafts must not follow.
  remountDeps: ({ params }) => params.projectId,
  component: BrandLookupRoute,
});

function BrandLookupRoute() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate({ from: Route.fullPath });
  // `c` is already an opaque competitor string array via the schema transform.
  const { q = "", c = [], scope, ai } = Route.useSearch();

  return (
    <BrandLookupPage
      projectId={projectId}
      initialQuery={q}
      initialCompetitors={c}
      initialScope={scope}
      initialBothPlatforms={ai === "both"}
      onSearchChange={(nextQuery, nextCompetitors, nextScope, nextBoth) => {
        void navigate({
          search: (prev) => ({
            ...prev,
            q: nextQuery.trim() || undefined,
            // One serialization site: comma-join the competitor list.
            c:
              nextCompetitors.length > 0
                ? nextCompetitors.join(",")
                : undefined,
            // The page passes a scope only when it differs from the default
            // derived from `q`.
            scope: nextScope,
            ai: nextBoth ? "both" : undefined,
          }),
          replace: true,
        });
      }}
    />
  );
}
