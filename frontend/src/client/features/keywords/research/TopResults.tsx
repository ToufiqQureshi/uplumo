import { useMutation } from "@tanstack/react-query";
import { ErrorState } from "@/client/components/ErrorState";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { Button } from "@/client/components/ui/button";
import { Card, CardContent } from "@/client/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import { getSerpAnalysis } from "@/client/features/keywords/keywordsApi";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { formatCount, formatUsd } from "./researchModel";

type SerpItem = Awaited<ReturnType<typeof getSerpAnalysis>>["items"][number];
type NumericKey =
  | "etv"
  | "estimatedPaidTrafficCost"
  | "referringDomains"
  | "backlinks";

const roundedCount = (value: number | null) =>
  formatCount(value == null ? null : Math.round(value));

const SERP_COLUMNS: Array<{
  key: NumericKey;
  label: string;
  format: (value: SerpItem[NumericKey]) => string;
}> = [
  { key: "etv", label: "Visits / mo", format: roundedCount },
  { key: "estimatedPaidTrafficCost", label: "Ads value", format: formatUsd },
  { key: "referringDomains", label: "Linking sites", format: roundedCount },
  { key: "backlinks", label: "Backlinks", format: roundedCount },
];

/**
 * The pages Google shows for the keyword and how strong each one is. Loaded
 * only when the customer asks: it is a separate paid lookup (cached on the
 * server, so asking again for the same keyword and country is free).
 */
export function TopResults({
  projectId,
  keyword,
  locationCode,
}: {
  projectId: string;
  keyword: string;
  locationCode: number | undefined;
}) {
  const serp = useMutation({
    mutationFn: () =>
      getSerpAnalysis({ projectId, keyword, locationCode, depth: 20 }),
    // A second attempt would buy the same results again.
    retry: false,
  });

  // Only the numbers this lookup returned get a column.
  const columns = SERP_COLUMNS.filter((c) =>
    serp.data?.items.some((item) => item[c.key] != null),
  );

  return (
    <Card>
      <CardContent className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="text-base font-semibold">Who ranks now</h2>
            <p className="text-sm text-muted-foreground">
              The pages Google shows for “{keyword}”, with how strong each one
              is.
            </p>
          </div>
          {serp.data ? null : (
            <Button
              variant="outline"
              pending={serp.isPending}
              onClick={() => serp.mutate()}
            >
              Show top results
            </Button>
          )}
        </div>
        {serp.isError ? (
          <ErrorState
            variant="inline"
            message={getStandardErrorMessage(serp.error)}
          />
        ) : null}
        {serp.data ? (
          serp.data.items.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Google returned no pages for this search.
            </p>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">#</TableHead>
                    <TableHead>Page</TableHead>
                    {columns.map((c) => (
                      <TableHead key={c.key} className="text-right">
                        {c.label}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {serp.data.items.map((item) => (
                    <TableRow key={`${item.rank}-${item.url}`}>
                      <TableCell className="font-mono tabular-nums">
                        {item.rank}
                      </TableCell>
                      <TableCell className="max-w-md">
                        <SafeExternalLink
                          url={item.url}
                          label={item.title || item.url}
                          className="inline-flex max-w-full items-center gap-1 truncate font-medium hover:underline"
                        />
                        <span className="block truncate text-xs text-muted-foreground">
                          {item.domain}
                          {item.description ? ` · ${item.description}` : ""}
                        </span>
                      </TableCell>
                      {columns.map((c) => (
                        <TableCell
                          key={c.key}
                          className="text-right font-mono tabular-nums"
                        >
                          {c.format(item[c.key])}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )
        ) : null}
      </CardContent>
    </Card>
  );
}
