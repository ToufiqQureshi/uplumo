import { PageHeader } from "@/client/components/PageHeader";
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/client/components/ui/card";

// Placeholder until the feature ships. It makes no request and spends
// nothing; the list says what is planned, not what works today.
const PLANNED = [
  {
    title: "Profile vs top 3",
    text: "Compare your business profile with the three businesses above you on the map, and get the fixes they have and you don't.",
  },
  {
    title: "Map rank",
    text: "See where you rank on the map across your area for a search, point by point.",
  },
  {
    title: "Reviews",
    text: "What customers praise and complain about, and reply drafts for reviews without an answer.",
  },
  {
    title: "Competitors",
    text: "Weekly changes in competitors' ratings, reviews and posts.",
  },
  {
    title: "AI answers",
    text: "Whether AI assistants name your business for local searches.",
  },
];

export function LocalVisibilityPage() {
  return (
    <div className="h-full overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-5xl space-y-6">
        <PageHeader
          title="Local Visibility"
          description="Coming soon on the Pro and Business plans. How your business shows up on maps, in reviews and in AI answers."
        />
        <div className="grid gap-3 sm:grid-cols-2">
          {PLANNED.map((item) => (
            <Card key={item.title}>
              <CardHeader>
                <CardTitle>
                  <h2>{item.title}</h2>
                </CardTitle>
                <CardDescription>{item.text}</CardDescription>
              </CardHeader>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}
