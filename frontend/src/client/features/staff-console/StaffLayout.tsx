import type { ReactNode } from "react";
import { Link, linkOptions } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { NotFound } from "@/client/components/NotFound";
import { PageLoading } from "@/client/components/Spinner";
import { useHostedAuthRouteGuard } from "@/client/features/auth/useHostedAuthRouteGuard";
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

/**
 * The staff console's own shell: no product navigation. Anyone the server
 * does not confirm as staff sees the ordinary not-found page.
 */
export function StaffLayout({ children }: { children: ReactNode }) {
  const authGate = useHostedAuthRouteGuard();
  const access = useQuery({
    ...staffAccessQueryOptions(),
    enabled: authGate.canRenderAuthenticatedContent,
  });
  if (!authGate.canRenderAuthenticatedContent || access.isPending) {
    return <PageLoading />;
  }
  if (!access.data?.staff) {
    return <NotFound />;
  }
  return (
    <div className="min-h-dvh bg-background">
      <header className="border-b border-border">
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
      <main className="mx-auto max-w-[1600px] px-4 py-6 md:px-6">
        {children}
      </main>
    </div>
  );
}
