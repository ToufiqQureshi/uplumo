import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
  userRequestedRun,
  useUserRequestedRun,
} from "@/client/features/ai-search/userRequestedRun";
import { PromptExplorerPage } from "@/client/features/ai-search/PromptExplorerPage";
import {
  PROMPT_EXPLORER_MODELS,
  promptExplorerSearchSchema,
} from "@/types/schemas/ai-search";

export const Route = createFileRoute("/_app/p/$projectId/ai-visibility/test")({
  validateSearch: promptExplorerSearchSchema,
  component: PromptExplorerRoute,
});

function PromptExplorerRoute() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate({ from: Route.fullPath });
  const requested = useUserRequestedRun();
  const search = Route.useSearch();

  return (
    <PromptExplorerPage
      projectId={projectId}
      requested={requested}
      urlState={{
        prompt: search.q ?? "",
        highlightBrand: search.hb ?? "",
        models:
          search.models && search.models.length > 0
            ? search.models
            : [...PROMPT_EXPLORER_MODELS],
        webSearch: search.web ?? true,
        webSearchCountryCode: search.cc,
      }}
      onSubmit={(values) => {
        void navigate({
          search: {
            q: values.prompt,
            models: values.models,
            web: values.webSearch ? undefined : false,
            cc: values.webSearchCountryCode,
            hb: values.highlightBrand || undefined,
          },
          replace: true,
          state: userRequestedRun,
        });
      }}
    />
  );
}
