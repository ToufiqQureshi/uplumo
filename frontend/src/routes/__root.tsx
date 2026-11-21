/// <reference types="vite/client" />
import { Outlet, createRootRoute } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { DefaultCatchBoundary } from "@/client/components/DefaultCatchBoundary";
import { captureGoogleLinkError } from "@/client/features/integrations/googleLinkError";
import { ExportToSheetsModal } from "@/client/components/table/ExportToSheetsModal";
import {
  identifyAnalyticsUser,
  logAnalyticsLifecycle,
  resetAnalyticsUser,
  setAnalyticsOrganization,
  startAnalyticsCapture,
  stopAnalyticsCapture,
} from "@/client/lib/posthog";
import { NotFound } from "@/client/components/NotFound";
import { TooltipProvider } from "@/client/components/ui/tooltip";
import { useSession } from "@/lib/auth-client";
import { Toaster } from "sonner";
import { queryClient } from "@/client/tanstack-db";
import { showCreditsUsed } from "@/client/features/billing/credits/creditsUsedToast";
import { getActiveOrganizationId } from "@/lib/auth-session";

// Capture Google link error params before the router starts — a route loader
// redirect would otherwise replace the URL and lose them. See googleLinkError.ts.
captureGoogleLinkError();

export const Route = createRootRoute({
  component: AppLayout,
  errorComponent: DefaultCatchBoundary,
  notFoundComponent: () => <NotFound />,
});

function PostHogBootstrap() {
  const { data: session, isPending: isSessionPending } = useSession();
  const userId = session?.user?.id ?? null;
  const email = session?.user?.email ?? null;
  const name = session?.user?.name ?? null;
  const optedOut = session?.user?.analyticsOptedOut === true;
  const organizationId = getActiveOrganizationId(session);
  const previousUserIdRef = React.useRef<string | null>(null);

  React.useEffect(() => {
    if (isSessionPending) {
      return;
    }

    if (userId && !optedOut) {
      const isNewAnalyticsIdentity = previousUserIdRef.current !== userId;
      if (previousUserIdRef.current && isNewAnalyticsIdentity) {
        resetAnalyticsUser();
      }
      startAnalyticsCapture();
      if (isNewAnalyticsIdentity) {
        identifyAnalyticsUser({ userId, email, name, organizationId });
        logAnalyticsLifecycle("analytics session identified", {
          event: "analytics_session_identified",
          organization_context: Boolean(organizationId),
        });
      } else {
        setAnalyticsOrganization(organizationId);
        logAnalyticsLifecycle("analytics organization context refreshed", {
          event: "analytics_organization_context_refreshed",
          organization_context: Boolean(organizationId),
        });
      }
      previousUserIdRef.current = userId;
    } else if (userId && optedOut) {
      stopAnalyticsCapture();
    } else if (previousUserIdRef.current) {
      previousUserIdRef.current = null;
      resetAnalyticsUser();
    }
  }, [email, isSessionPending, name, optedOut, organizationId, userId]);

  return null;
}

function AppLayout() {
  const showDevtools =
    import.meta.env.DEV && import.meta.env.VITE_SHOW_DEVTOOLS !== "false";
  React.useEffect(() => showCreditsUsed(), []);

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <PostHogBootstrap />
        <Outlet />
        <ExportToSheetsModal />
        <Toaster position="bottom-right" mobileOffset={{ bottom: 100 }} />
        {showDevtools ? (
          <TanStackDevtools
            config={{ position: "bottom-right" }}
            eventBusConfig={{ connectToServerBus: true }}
            plugins={[
              {
                name: "TanStack Router",
                render: <TanStackRouterDevtoolsPanel />,
                defaultOpen: true,
              },
            ]}
          />
        ) : null}
      </TooltipProvider>
    </QueryClientProvider>
  );
}
