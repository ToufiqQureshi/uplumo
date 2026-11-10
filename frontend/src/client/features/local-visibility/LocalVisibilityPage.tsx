import { useState } from "react";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { Card, CardContent } from "@/client/components/ui/card";
import { Skeleton } from "@/client/components/ui/skeleton";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/client/components/ui/tabs";
import { useHostedPlanGate } from "@/client/features/billing/HostedPlanGate";
import { AIAnswersTab } from "./AIAnswersTab";
import { BusinessSetup } from "./BusinessSetup";
import { CheckTab } from "./CheckTab";
import { GridTab } from "./GridTab";
import { type Overview, useOverview } from "./localVisibilityApi";
import { ReviewsTab } from "./ReviewsTab";
import { WatchTab } from "./WatchTab";

const DESCRIPTION =
  "How your Google Business Profile compares with the businesses that beat you on Google Maps. Nothing is bought until you click a button that shows its credits.";

export function LocalVisibilityPage({ projectId }: { projectId: string }) {
  const plan = useHostedPlanGate();
  const overview = useOverview(projectId);
  return (
    <div className="h-full overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-5xl space-y-6">
        <PageHeader title="Local Visibility" description={DESCRIPTION} />
        {plan === "free" ? (
          <Card>
            <CardContent className="py-6 text-sm">
              Local Visibility is on the Pro and Business plans.
            </CardContent>
          </Card>
        ) : overview.isPending ? (
          <Skeleton className="h-40 w-full" />
        ) : overview.error ? (
          <p role="alert" className="text-sm text-destructive">
            {overview.error.message}
          </p>
        ) : (
          <LocalBody projectId={projectId} overview={overview.data} />
        )}
      </div>
    </div>
  );
}

function LocalBody({
  projectId,
  overview,
}: {
  projectId: string;
  overview: Overview;
}) {
  const [editing, setEditing] = useState(false);
  const business = overview.business;
  if (!business || editing) {
    return (
      <div className="space-y-3">
        <BusinessSetup
          projectId={projectId}
          current={business}
          findCredits={overview.prices.find}
          onDone={() => setEditing(false)}
        />
        {business ? (
          <Button variant="ghost" onClick={() => setEditing(false)}>
            Cancel
          </Button>
        ) : null}
      </div>
    );
  }
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 rounded-lg border p-4 sm:flex-row sm:items-center">
        <div className="min-w-0 flex-1">
          <h2 className="text-lg font-semibold">{business.title}</h2>
          <p className="text-sm text-muted-foreground">
            {[business.category, business.address].filter(Boolean).join(" · ")}
          </p>
          <p className="text-sm text-muted-foreground">
            Maps search: “{business.searchTerm}”
          </p>
        </div>
        <Button variant="outline" onClick={() => setEditing(true)}>
          Change business or search
        </Button>
      </div>
      <Tabs defaultValue="check">
        <TabsList className="flex-wrap">
          <TabsTrigger value="check">Profile vs top 3</TabsTrigger>
          <TabsTrigger value="grid">Map rank grid</TabsTrigger>
          <TabsTrigger value="reviews">Reviews</TabsTrigger>
          <TabsTrigger value="watch">Competitor watch</TabsTrigger>
          <TabsTrigger value="ai">AI answers</TabsTrigger>
        </TabsList>
        <TabsContent value="check" className="pt-4">
          <CheckTab projectId={projectId} overview={overview} />
        </TabsContent>
        <TabsContent value="grid" className="pt-4">
          <GridTab projectId={projectId} overview={overview} />
        </TabsContent>
        <TabsContent value="reviews" className="pt-4">
          <ReviewsTab projectId={projectId} overview={overview} />
        </TabsContent>
        <TabsContent value="watch" className="pt-4">
          <WatchTab projectId={projectId} overview={overview} />
        </TabsContent>
        <TabsContent value="ai" className="pt-4">
          <AIAnswersTab projectId={projectId} business={business} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
