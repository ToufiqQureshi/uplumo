import {
  Bookmark,
  Bot,
  Brain,
  ChartNoAxesCombined,
  CircleHelp,
  ClipboardCheck,
  CreditCard,
  FileText,
  Globe,
  LayoutDashboard,
  Link2,
  MessageSquare,
  MessagesSquare,
  PenLine,
  Search,
  Sparkles,
  TrendingUp,
} from "lucide-react";
import {
  linkOptions,
  type LinkOptions,
  type useMatchRoute,
} from "@tanstack/react-router";
import type { ComponentType } from "react";
import { GoogleGlyphMuted } from "@/client/features/gsc/GoogleGlyph";

// Labels match each page's own title, so the header breadcrumb and the page
// heading agree.
const projectNavItems = [
  {
    to: "/p/$projectId/prompt-tracking" as const,
    label: "AI Visibility Tracker",
    icon: Sparkles,
    badge: "New",
  },
  {
    to: "/p/$projectId" as const,
    label: "Dashboard",
    icon: LayoutDashboard,
    // Without exact matching, the index path is a prefix of every project
    // route and the Dashboard item would render active everywhere.
    activeOptions: { exact: true, includeSearch: false },
  },
  {
    to: "/p/$projectId/sam" as const,
    label: "SAM assistant",
    icon: MessagesSquare,
    badge: "Beta",
  },
  {
    to: "/p/$projectId/analytics" as const,
    label: "Analytics",
    icon: ChartNoAxesCombined,
  },
  {
    to: "/p/$projectId/search-performance" as const,
    label: "Search Performance",
    icon: GoogleGlyphMuted,
  },
  {
    to: "/p/$projectId/brand-lookup" as const,
    label: "Brand Lookup",
    icon: Sparkles,
  },
  {
    to: "/p/$projectId/prompt-explorer" as const,
    label: "Prompt Explorer",
    icon: MessageSquare,
  },
  {
    to: "/p/$projectId/rank-tracking" as const,
    label: "Rank Tracking",
    icon: TrendingUp,
  },
  {
    to: "/p/$projectId/audit" as const,
    label: "Site Audit",
    icon: ClipboardCheck,
  },
  {
    to: "/p/$projectId/content-editor" as const,
    label: "Content Editor",
    icon: PenLine,
  },
  {
    to: "/p/$projectId/saved" as const,
    label: "Saved Keywords",
    icon: Bookmark,
  },
  {
    to: "/p/$projectId/keywords" as const,
    label: "Keyword Research",
    icon: Search,
  },
  {
    to: "/p/$projectId/domain" as const,
    label: "Domain Overview",
    icon: Globe,
  },
  {
    to: "/p/$projectId/backlinks" as const,
    label: "Backlinks",
    icon: Link2,
  },
  {
    to: "/p/$projectId/reports" as const,
    label: "Reports",
    icon: FileText,
  },
  {
    to: "/p/$projectId/context" as const,
    label: "Context",
    icon: Brain,
  },
] as const;

type ProjectNavPath = (typeof projectNavItems)[number]["to"];

export type NavItem = {
  label: string;
  icon: ComponentType<{ className?: string }>;
  /** A short tag after the label, such as "New". */
  badge?: string;
  link: LinkOptions;
  /** Sibling pages shown as tabs above the page; the item opens the first. */
  tabs?: NavItem[];
};

type NavGroup = { label: string; items: NavItem[] };

// Project-independent. Rendered inside the project "Reports & setup" group when a
// project is selected, and on its own (connectNavGroup) when none is.
const aiNavItem: NavItem = {
  label: "Agent setup",
  icon: Bot,
  link: linkOptions({ to: "/ai" }),
};

// Shown only when no project is selected; with a project, Agent setup lives in
// the "Reports & setup" group below.
export const connectNavGroup: NavGroup = {
  label: "AI",
  items: [aiNavItem],
};

// One sidebar row for several related pages. The pages keep their own routes,
// so old links and each page's paid-query settings are untouched; switching a
// tab is the same navigation the sidebar used to do.
function tabbedItem(
  label: string,
  icon: NavItem["icon"],
  tabs: NavItem[],
  badge?: string,
): NavItem {
  return { label, icon, badge, link: tabs[0].link, tabs };
}

/** The page of `item` (itself or one of its tabs) the current URL is on. */
export function findActivePage(
  matchRoute: ReturnType<typeof useMatchRoute>,
  item: NavItem,
): NavItem | undefined {
  return (item.tabs ?? [item]).find((page) =>
    matchRoute({ ...page.link, fuzzy: !page.link.activeOptions?.exact }),
  );
}

// Pinned to the sidebar footer, above the account menu.
export const accountNavGroup: NavGroup = {
  label: "Account",
  items: [
    {
      label: "Billing",
      icon: CreditCard,
      link: linkOptions({ to: "/billing/plan" }),
    },
    { label: "Help", icon: CircleHelp, link: linkOptions({ to: "/support" }) },
  ],
};

/** One project page's link, label and icon, shared by the nav and shortcuts. */
export function getProjectNavItem(
  projectId: string,
  path: ProjectNavPath,
): NavItem {
  const { label, icon, ...item } = projectNavItems.find(
    (navItem) => navItem.to === path,
  )!;
  return {
    label,
    icon,
    badge: "badge" in item ? item.badge : undefined,
    link: linkOptions({
      to: item.to,
      activeOptions: "activeOptions" in item ? item.activeOptions : undefined,
      params: { projectId },
      search: {},
    }),
  };
}

// Grouped by the job the user came to do, in plain words for owners who are
// not SEO experts. Related pages share one row and switch with tabs, so the
// sidebar fits one screen.
export function getProjectNavGroups(projectId: string): NavGroup[] {
  const item = (path: ProjectNavPath) => getProjectNavItem(projectId, path);

  return [
    {
      label: "Home",
      items: [item("/p/$projectId"), item("/p/$projectId/sam")],
    },
    {
      label: "Customers & traffic",
      items: [
        tabbedItem("Traffic", ChartNoAxesCombined, [
          item("/p/$projectId/analytics"),
          item("/p/$projectId/search-performance"),
        ]),
        tabbedItem(
          "AI Visibility",
          Sparkles,
          [
            item("/p/$projectId/prompt-tracking"),
            item("/p/$projectId/brand-lookup"),
            item("/p/$projectId/prompt-explorer"),
          ],
          "New",
        ),
      ],
    },
    {
      label: "Your website",
      items: [
        item("/p/$projectId/audit"),
        item("/p/$projectId/rank-tracking"),
        item("/p/$projectId/content-editor"),
      ],
    },
    {
      label: "Research",
      items: [
        tabbedItem("Keywords", Search, [
          item("/p/$projectId/keywords"),
          item("/p/$projectId/saved"),
        ]),
        tabbedItem("Competitors", Globe, [
          item("/p/$projectId/domain"),
          item("/p/$projectId/backlinks"),
        ]),
      ],
    },
    {
      label: "Reports & setup",
      items: [
        item("/p/$projectId/reports"),
        tabbedItem("Setup", Brain, [item("/p/$projectId/context"), aiNavItem]),
      ],
    },
  ];
}
