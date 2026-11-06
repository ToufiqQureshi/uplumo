import { Tabs, TabsList, TabsTrigger } from "@/client/components/ui/tabs";
import {
  SEARCH_GROWTH_VIEWS,
  type SearchGrowthView,
} from "@/types/schemas/search-performance";

const VIEW_LABEL: Record<SearchGrowthView, string> = {
  overview: "Overview",
  fix: "Fix",
  grow: "Grow",
  explore: "Explore",
  indexing: "Indexing",
  conversions: "Conversions",
};

/** The Search Growth screens, one job each. */
export function ViewTabs({
  value,
  onChange,
}: {
  value: SearchGrowthView;
  onChange: (view: SearchGrowthView) => void;
}) {
  return (
    <Tabs
      value={value}
      onValueChange={(next) => {
        const view = SEARCH_GROWTH_VIEWS.find((item) => item === next);
        if (view) onChange(view);
      }}
    >
      <TabsList
        variant="line"
        className="h-auto! flex-wrap justify-start *:flex-none"
      >
        {SEARCH_GROWTH_VIEWS.map((item) => (
          <TabsTrigger key={item} value={item}>
            {VIEW_LABEL[item]}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
