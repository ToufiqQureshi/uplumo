import type { ReactNode } from "react";
import { Link, linkOptions } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { NotFound } from "@/client/components/NotFound";
import { PageLoading } from "@/client/components/Spinner";
import { Button } from "@/client/components/ui/button";
import { useHostedAuthRouteGuard } from "@/client/features/auth/useHostedAuthRouteGuard";
import { ApiError } from "@/client/lib/uplumoApi";
import { staffAccessQueryOptions } from "./customersApi";

const tabs = [
  {
    label: "Overview",
    link: linkOptions({ to: "/staff", activeOptions: { exact: true } }),
  },
  { label: "Subscriptions", link: linkOptions({ to: "/staff/subscriptions" }) },
  { label: "Customers", link: linkOptions({ to: "/staff/customers" }) },
  { label: "Marketing", link: linkOptions({ to: "/staff/marketing" }) },
  { label: "Finance", link: linkOptions({ to: "/staff/finance" }) },
  { label: "Activity", link: linkOptions({ to: "/staff/activity" }) },
];

/** Staff-only shell with its own navigation and content viewport. */
export function StaffLayout({ children }: { children: ReactNode }) {
  const authGate = useHostedAuthRouteGuard();
  const access = useQuery({
    ...staffAccessQueryOptions(),
    enabled: authGate.canRenderAuthenticatedContent,
  });
  if (!authGate.canRenderAuthenticatedContent || access.isPending) {
    return <PageLoading />;
  }
  if (access.isError) {
    if (
      access.error instanceof ApiError &&
      [401, 403, 404].includes(access.error.status)
    ) {
      return <NotFound />;
    }
    return (
      <div className="mx-auto max-w-md space-y-3 px-4 py-6">
        <p role="alert" className="text-sm text-destructive">
          Could not load Staff Console. Please try again.
        </p>
        <Button
          variant="outline"
          disabled={access.isFetching}
          onClick={() => void access.refetch()}
        >
          Try again
        </Button>
      </div>
    );
  }
  if (!access.data?.staff) {
    return <NotFound />;
  }
  return (
    <div className="flex h-dvh min-h-0 flex-col bg-background">
      <header className="shrink-0 border-b border-border">
        <div className="mx-auto flex max-w-[1600px] flex-wrap items-center gap-x-6 gap-y-2 px-4 py-3 md:px-6">
          <span className="font-semibold">Uplumo Staff</span>
          <nav
            aria-label="Staff console"
            className="-mx-1 flex flex-wrap gap-1"
          >
            {tabs.map((tab) => (
              <Link
                key={tab.label}
                {...tab.link}
                className="rounded-md px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                activeProps={{
                  className: "bg-muted text-foreground font-medium",
                }}
              >
                {tab.label}
              </Link>
            ))}
          </nav>
          <Link
            to="/"
            className="ml-auto inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft aria-hidden className="size-4" />
            Back to app
          </Link>
        </div>
      </header>
      <main
        aria-label="Staff console content"
        tabIndex={0}
        className="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
      >
        <div className="mx-auto max-w-[1600px] px-4 py-6 md:px-6">
          {children}
        </div>
      </main>
    </div>
  );
}
