import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { SearchContextPanel } from "@/client/features/content-editor/SearchContextPanel";
import { PageHeader } from "@/client/components/PageHeader";
import { ContentEditorPage } from "@/client/features/content-editor/ContentEditorPage";
import { EditorSessionProvider } from "@/client/features/content-editor/editor-session";

export const Route = createFileRoute("/_app/p/$projectId/content-editor")({
  validateSearch: z.object({
    growth: z.literal("1").optional().catch(undefined),
    keyword: z.string().trim().min(1).max(200).optional().catch(undefined),
  }),
  component: ContentEditorRoute,
});

function ContentEditorRoute() {
  const { projectId } = Route.useParams();
  const { keyword, growth } = Route.useSearch();
  return (
    <div className="h-full overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-[1500px] space-y-5">
        <PageHeader
          title="Content Editor"
          description="Write for people first. Switch between search visibility and answer readiness as you shape your draft."
        />
        {growth ? (
          <SearchContextPanel key={projectId} projectId={projectId} />
        ) : null}
        <EditorSessionProvider>
          <ContentEditorPage
            key={projectId}
            projectId={projectId}
            initialKeyword={keyword}
          />
        </EditorSessionProvider>
      </div>
    </div>
  );
}
