import { useDeferredValue, useState } from "react";
import { useSession } from "@/lib/auth-client";
import { useQuery } from "@tanstack/react-query";
import { Download } from "lucide-react";
import { Button, buttonVariants } from "@/client/components/ui/button";
import { Badge } from "@/client/components/ui/badge";
import { Input } from "@/client/components/ui/input";
import {
  Table,
  TableBody,
  TableCard,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  CUSTOMERS_CSV_URL,
  customersQueryOptions,
  interestsText,
  type Customer,
} from "./customersApi";
import type { UserAction } from "./staffApi";
import { UserActionDialog } from "./UserActionDialog";

const dateFormat = new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" });

function matches(customer: Customer, query: string) {
  if (!query) return true;
  return [
    customer.email,
    customer.name,
    customer.workspace,
    customer.plan,
    customer.domains.join(" "),
    customer.interestedFeatures,
    customer.workFor,
    customer.foundVia,
  ].some((value) => value.toLowerCase().includes(query));
}

export function CustomersPage() {
  const customers = useQuery(customersQueryOptions());
  const ownEmail = useSession().data?.user?.email;
  const [search, setSearch] = useState("");
  const [pending, setPending] = useState<{
    action: UserAction;
    user: Customer;
  }>();
  const query = useDeferredValue(search.trim().toLowerCase());
  const rows = (customers.data?.customers ?? []).filter((c) =>
    matches(c, query),
  );

  return (
    <div className="space-y-4">
      {customers.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {getStandardErrorMessage(customers.error)}
        </p>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-3">
            <Input
              aria-label="Search customers"
              placeholder="Search email, website, plan or answer"
              className="max-w-sm"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
            <p className="text-sm text-muted-foreground" role="status">
              {customers.isPending
                ? "Loading…"
                : `${rows.length} of ${customers.data.customers.length} users`}
            </p>
            <a
              href={CUSTOMERS_CSV_URL}
              download
              className={buttonVariants({
                variant: "outline",
                className: "ml-auto",
              })}
            >
              <Download aria-hidden className="size-4" />
              Download CSV
            </a>
          </div>
          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Signed up</TableHead>
                  <TableHead>Plan</TableHead>
                  <TableHead>Websites</TableHead>
                  <TableHead>Google</TableHead>
                  <TableHead>Wants</TableHead>
                  <TableHead>SEO for</TableHead>
                  <TableHead>Found us via</TableHead>
                  <TableHead>Onboarding</TableHead>
                  <TableHead>
                    <span className="sr-only">Actions</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((c) => (
                  <TableRow key={c.userId}>
                    <TableCell>
                      <div className="font-medium">{c.email}</div>
                      <div className="text-xs text-muted-foreground">
                        {[c.name, c.workspace].filter(Boolean).join(" · ")}
                        {c.emailVerified ? "" : " · email not verified"}
                      </div>
                      {c.blocked ? (
                        <Badge variant="destructive">Blocked</Badge>
                      ) : null}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {dateFormat.format(new Date(c.signedUpAt))}
                    </TableCell>
                    <TableCell className="capitalize">{c.plan}</TableCell>
                    <TableCell>
                      <div>{c.activeSites}</div>
                      <div className="text-xs text-muted-foreground">
                        {c.domains.join(", ")}
                      </div>
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {[c.gscConnected && "GSC", c.ga4Connected && "GA4"]
                        .filter(Boolean)
                        .join(" + ") || "—"}
                    </TableCell>
                    <TableCell>
                      {interestsText(c.interestedFeatures) || "—"}
                    </TableCell>
                    <TableCell>
                      {[c.workFor, c.clientWebsiteCount]
                        .filter(Boolean)
                        .join(" · ") || "—"}
                    </TableCell>
                    <TableCell>{c.foundVia || "—"}</TableCell>
                    <TableCell className="whitespace-nowrap">
                      {c.onboardingCompletedAt ? "Completed" : "Not finished"}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {c.email === ownEmail ? null : (
                        <>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() =>
                              setPending({
                                action: c.blocked ? "unblock" : "block",
                                user: c,
                              })
                            }
                          >
                            {c.blocked ? "Unblock" : "Block"}
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            className="text-destructive"
                            onClick={() =>
                              setPending({ action: "delete", user: c })
                            }
                          >
                            Delete
                          </Button>
                        </>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableCard>
        </>
      )}
      {pending ? (
        <UserActionDialog
          action={pending.action}
          user={pending.user}
          onClose={() => setPending(undefined)}
        />
      ) : null}
    </div>
  );
}
