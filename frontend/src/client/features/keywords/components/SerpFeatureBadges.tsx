import { Badge } from "@/client/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/client/components/ui/tooltip";

type Feature = { label: string; title: string; action: string };

// What each Google result type means for the customer, and what to do about
// it. Ordered by how much it changes the plan: a map or AI answer above the
// results matters more than a video carousel.
const FEATURES: Array<[types: string[], feature: Feature]> = [
  [
    ["local_pack", "map", "hotels_pack", "google_hotels"],
    {
      label: "Maps",
      title: "Map or hotel listings",
      action:
        "Google shows businesses on a map above the websites. Your Google Business Profile (photos, reviews, hours) matters as much as your website here.",
    },
  ],
  [
    ["ai_overview"],
    {
      label: "AI",
      title: "AI Overview",
      action:
        "Google answers with AI first. Give a short, direct answer near the top of your page so the AI answer can quote you.",
    },
  ],
  [
    ["featured_snippet"],
    {
      label: "Snippet",
      title: "Featured snippet",
      action:
        "One page is shown as a highlighted answer. A clear 40-60 word answer under a matching heading can win it.",
    },
  ],
  [
    ["people_also_ask"],
    {
      label: "Q&A",
      title: "People also ask",
      action:
        "Google lists related questions. Answer them in an FAQ section on your page.",
    },
  ],
  [
    ["video", "short_videos"],
    {
      label: "Video",
      title: "Videos",
      action:
        "Google shows videos. A short video on YouTube for this search can rank alongside your page.",
    },
  ],
];

const MAX_BADGES = 3;

/** The plain-language result types Google shows for a keyword, most important first. */
export function serpFeatureLabels(types: readonly string[] | undefined) {
  if (!types?.length) return [];
  return FEATURES.filter(([match]) => match.some((t) => types.includes(t))).map(
    ([, feature]) => feature,
  );
}

export function SerpFeatureBadges({
  types,
}: {
  types: readonly string[] | undefined;
}) {
  const features = serpFeatureLabels(types).slice(0, MAX_BADGES);
  if (features.length === 0) return <span>-</span>;
  return (
    <span className="inline-flex gap-1">
      {features.map((feature) => (
        <Tooltip key={feature.label}>
          <TooltipTrigger
            delay={0}
            render={
              <Badge
                variant="secondary"
                tabIndex={0}
                className="h-6 cursor-help"
              />
            }
            aria-label={feature.title}
          >
            {feature.label}
          </TooltipTrigger>
          <TooltipContent className="max-w-72 flex-col items-start gap-1">
            <span className="font-semibold">{feature.title}</span>
            <span>{feature.action}</span>
          </TooltipContent>
        </Tooltip>
      ))}
    </span>
  );
}
