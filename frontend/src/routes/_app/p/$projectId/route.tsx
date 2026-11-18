import {
  Outlet,
  createFileRoute,
  useMatch,
  useNavigate,
} from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { ErrorState } from "@/client/components/ErrorState";
import { ApiError } from "@/client/lib/uplumoApi";
import { setLastProjectId } from "@/client/lib/active-project";
import { getErrorCode } from "@/client/lib/error-messages";
import {
  getCurrentAuthRedirectFromHref,
  getSignInSearch,
} from "@/lib/auth-redirect";
import { getProjectAccess } from "@/client/features/projects/projectApi";

export const Route = createFileRoute("/_app/p/$projectId")({
  // Everything under this subtree fetches its data client-side with
  // react-query, so SSR would only render empty chrome.
  ssr: false,
  component: ProjectLayout,
});

// Redirect-only guard, deliberately NOT a blocking beforeLoad: the shell
// renders immediately while the access check runs in the background, and the
// browser only gets bounced if it lands on a project it can't see (stale
// last-project id, foreign URL). Real authorization is enforced on every data
// call; nothing sensitive renders from this check.
function useProjectAccessRedirect(projectId: string) {
  const navigate = useNavigate();
  const access = useQuery({
    queryKey: ["projectAccess", projectId],
    queryFn: () => getProjectAccess({ data: { projectId } }),
    meta: { errorToast: false },
    retry: false,
    // One check per project per tab; a revoked project still dead-ends at
    // every data call, so there's nothing to re-validate here.
    staleTime: Infinity,
  });
  const error = access.error;
  const signedOut = getErrorCode(error) === "UNAUTHENTICATED";
  const projectMissing =
    error instanceof ApiError &&
    error.status === 404 &&
    ["PROJECT_NOT_FOUND", "NOT_FOUND"].includes(error.code.toUpperCase());
  useEffect(() => {
    if (signedOut) {
      void navigate({
        to: "/sign-in",
        search: getSignInSearch(
          getCurrentAuthRedirectFromHref(window.location.href),
        ),
        replace: true,
      });
      return;
    }
    if (projectMissing) void navigate({ to: "/", replace: true });
  }, [signedOut, projectMissing, navigate]);
  return {
    access,
    retryableError: Boolean(error) && !signedOut && !projectMissing,
  };
}

function ProjectLayout() {
  const { projectId } = Route.useParams();
  const { access, retryableError } = useProjectAccessRedirect(projectId);

  // Remember this as the last-visited project for the landing redirect.
  // Settings and its sub-pages are excluded: editing another project's
  // settings is administration, not a context switch, so it shouldn't change
  // which project the app opens next time. (An explicit choice still counts:
  // the switcher and project creation set it themselves, settings page or not.)
  const isSettingsPage =
    useMatch({
      from: "/_app/p/$projectId/settings",
      shouldThrow: false,
      select: () => true,
    }) ?? false;
  useEffect(() => {
    if (isSettingsPage || !access.isSuccess) return;
    setLastProjectId(projectId);
  }, [projectId, isSettingsPage, access.isSuccess]);

  if (retryableError) {
    return (
      <ErrorState
        variant="page"
        title="Could not open this project"
        message="We couldn't check project access. Try again to stay on this page."
        onRetry={() => void access.refetch()}
        isRetrying={access.isFetching}
      />
    );
  }
  return <Outlet />;
}
