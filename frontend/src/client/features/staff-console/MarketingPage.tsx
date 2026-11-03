import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "@/client/components/ui/card";
import { CountTable, QueryView } from "./StaffParts";
import { marketingQueryOptions } from "./staffApi";

function Panel({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <CardContent className="space-y-3">
        <h2 className="text-lg font-semibold">{title}</h2>
        {children}
      </CardContent>
    </Card>
  );
}

export function MarketingPage() {
  const marketing = useQuery(marketingQueryOptions());
  return (
    <QueryView query={marketing}>
      {(m) => {
        const signedUp = m.funnel[0]?.count ?? 0;
        return (
          <div className="grid gap-6 lg:grid-cols-2">
            <Panel title="Funnel: signup to paying">
              <CountTable rows={m.funnel} total={signedUp} />
              <p className="text-xs text-muted-foreground">
                Share is of everyone who signed up.
              </p>
            </Panel>
            <Panel title="How they found Uplumo">
              <CountTable rows={m.foundVia} />
            </Panel>
            <Panel title="What they want to do">
              <CountTable rows={m.interests} total={signedUp} />
              <p className="text-xs text-muted-foreground">
                Users can pick more than one.
              </p>
            </Panel>
            <Panel title="Who they do SEO for">
              <CountTable rows={m.workFor} />
            </Panel>
          </div>
        );
      }}
    </QueryView>
  );
}
