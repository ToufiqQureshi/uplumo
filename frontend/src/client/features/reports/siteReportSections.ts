import {
  Activity,
  Gauge,
  Link2,
  MapPin,
  MousePointer2,
  Search,
  Sparkles,
  TrendingUp,
  Users,
  type LucideIcon,
} from "lucide-react";
import type { SiteReportSection } from "@/client/features/reports/reportApi";

export const SITE_REPORT_SECTION_DETAILS: Record<
  SiteReportSection,
  { label: string; detail: string; icon: LucideIcon }
> = {
  search: {
    label: "Google Search",
    detail: "Clicks, impressions, top searches and pages, rising and falling",
    icon: Search,
  },
  traffic: {
    label: "Google Analytics",
    detail: "Traffic channels, landing pages, engagement and key events",
    icon: Activity,
  },
  analytics: {
    label: "Uplumo Analytics",
    detail: "Traffic, entry pages, AI referrals and tracked actions",
    icon: MousePointer2,
  },
  rankings: {
    label: "Keyword rankings",
    detail: "Tracked positions, movers and ranking pages",
    icon: TrendingUp,
  },
  competitors: {
    label: "Competitor rankings",
    detail:
      "Positions, ranking pages and monthly searches for tracked keywords",
    icon: Users,
  },
  ai_visibility: {
    label: "AI visibility and citations",
    detail: "Answers and cited pages by AI engine and prompt",
    icon: Sparkles,
  },
  local_visibility: {
    label: "Local visibility",
    detail: "Google Business Profile, Maps rankings, competitors and reviews",
    icon: MapPin,
  },
  audit: {
    label: "Site health",
    detail: "Audit issues with pages, speed and quality scores",
    icon: Gauge,
  },
  backlinks: {
    label: "Backlinks",
    detail: "Linking websites gained and lost",
    icon: Link2,
  },
};
