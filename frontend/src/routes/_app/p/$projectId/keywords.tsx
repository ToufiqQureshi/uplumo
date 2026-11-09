import { createFileRoute, redirect } from "@tanstack/react-router";
import { KeywordResearchPage } from "@/client/features/keywords/research/KeywordResearchPage";
import { normalizeLegacyKeywordSearch } from "@/client/features/keywords/keywordSearchParams";
import { keywordsSearchSchema } from "@/types/schemas/keywords";

export const Route = createFileRoute("/_app/p/$projectId/keywords")({
  validateSearch: keywordsSearchSchema,
  // The project switcher keeps this page; filter drafts must not follow.
  remountDeps: ({ params }) => params.projectId,
  beforeLoad: ({ params, search }) => {
    const { normalized, changed } = normalizeLegacyKeywordSearch(search);
    if (!changed) return;

    throw redirect({
      to: "/p/$projectId/keywords",
      params: { projectId: params.projectId },
      search: normalized,
      replace: true,
    });
  },
  component: KeywordResearchPageRoute,
});

function KeywordResearchPageRoute() {
  const { projectId } = Route.useParams();
  const { q } = Route.useSearch();
  return (
    <KeywordResearchPage
      projectId={projectId}
      initialKeyword={q || undefined}
    />
  );
}
