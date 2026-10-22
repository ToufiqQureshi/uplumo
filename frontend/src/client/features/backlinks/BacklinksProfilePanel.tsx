import { sortBy } from "remeda";
import { HelpLabel } from "@/client/components/HelpLabel";
import type { BacklinksOverviewData } from "./backlinksPageTypes";
import { formatCompactDate, formatNumber } from "./backlinksPageUtils";

type Profile = NonNullable<BacklinksOverviewData["summary"]["profile"]>;

const PLATFORM_LABELS: Record<string, string> = {
  unknown: "Other sites",
  blogs: "Blogs",
  cms: "CMS sites",
  organization: "Company sites",
  news: "News",
  "message-boards": "Forums",
  ecommerce: "Online shops",
};
const PLACEMENT_LABELS: Record<string, string> = {
  details: "Inside page details",
  article: "Inside an article",
  footer: "Footer",
  header: "Header",
  nav: "Menu",
  aside: "Sidebar",
  figure: "Image caption",
  section: "Page section",
  main: "Main content",
};
const LINK_TYPE_LABELS: Record<string, string> = {
  anchor: "Text link",
  image: "Image link",
  redirect: "Redirect",
  canonical: "Canonical",
  form: "Form",
  frame: "Frame",
  rss: "RSS",
  unknown: "Other",
};
const LINK_ATTRIBUTE_LABELS: Record<string, string> = {
  nofollow: "Nofollow",
  ugc: "User content",
  sponsored: "Sponsored",
  alternate: "Alternate",
  unknown: "Other",
};

/**
 * The link-profile breakdown the summary task already returns with its totals.
 * Shown so the customer sees everything the report paid for.
 */
export function BacklinksProfilePanel({ profile }: { profile: Profile }) {
  const facts = [
    {
      label: "First link seen",
      value: profile.firstSeen ? formatCompactDate(profile.firstSeen) : "-",
      help: "When the crawler first found a link to this site.",
    },
    {
      label: "Linking IPs",
      value: formatNumber(profile.referringIps ?? null),
      help: "Different server addresses the links come from. Far fewer IPs than domains can mean a link network.",
    },
    {
      label: "Linking subnets",
      value: formatNumber(profile.referringSubnets ?? null),
      help: "Different server networks the links come from. A small number points to links built on the same hosting.",
    },
    {
      label: "Nofollow domains",
      value: formatNumber(profile.referringDomainsNofollow ?? null),
      help: "Domains that link with at least one nofollow link. Nofollow links pass little ranking value.",
    },
    {
      label: "Pages crawled",
      value: formatNumber(profile.crawledPages ?? null),
      help: "Pages on your site the crawler checked for links.",
    },
  ];
  const lists = [
    {
      title: "Domain endings",
      help: "Where linking sites are registered (.com, .in, .store...). Many cheap endings such as .store, .site or .online often mean spam links.",
      counts: profile.tlds,
      label: (key: string) => `.${key}`,
    },
    {
      title: "Type of linking site",
      help: "What kind of site links to you.",
      counts: profile.platformTypes,
      label: (key: string) => PLATFORM_LABELS[key] ?? key,
    },
    {
      title: "Where the link sits",
      help: "The part of the page the link is in. Links inside an article usually count more than footer links.",
      counts: profile.placements,
      label: (key: string) => PLACEMENT_LABELS[key] ?? key,
    },
    {
      title: "Linking site country",
      help: "Country of the linking site, where the crawler could tell.",
      counts: profile.countries,
      label: (key: string) => key,
    },
    {
      title: "Link type",
      help: "How the link is built: a normal text link, an image, a redirect or a canonical tag.",
      counts: profile.linkTypes,
      label: (key: string) => LINK_TYPE_LABELS[key] ?? key,
    },
    {
      title: "Link attribute",
      help: "The rel attribute on the link: nofollow, sponsored or user-generated content (UGC).",
      counts: profile.linkAttributes,
      label: (key: string) => LINK_ATTRIBUTE_LABELS[key] ?? key,
    },
  ].filter((list) => Object.keys(list.counts).length > 0);

  return (
    <div className="space-y-3 rounded-lg border border-border p-3">
      <div>
        <h3 className="text-sm font-medium">Link profile</h3>
        <p className="text-xs text-muted-foreground">
          Where the links come from, from the same report
        </p>
      </div>
      <div className="grid grid-cols-2 gap-x-6 gap-y-4 md:grid-cols-4">
        {facts.map((fact) => (
          <div key={fact.label}>
            <div className="text-xs tracking-wide text-muted-foreground uppercase">
              <HelpLabel label={fact.label} helpText={fact.help} />
            </div>
            <p className="text-lg font-semibold">{fact.value}</p>
          </div>
        ))}
      </div>
      {lists.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          {lists.map((list) => (
            <CountList key={list.title} {...list} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function CountList({
  title,
  help,
  counts,
  label,
}: {
  title: string;
  help: string;
  counts: Record<string, number>;
  label: (key: string) => string;
}) {
  const entries = sortBy(Object.entries(counts), [
    ([, count]) => count,
    "desc",
  ]).slice(0, 6);
  const total = Object.values(counts).reduce((sum, n) => sum + n, 0);
  return (
    <div className="space-y-1.5">
      <div className="text-xs font-medium">
        <HelpLabel label={title} helpText={help} />
      </div>
      <ul className="space-y-1">
        {entries.map(([key, count]) => (
          <li key={key} className="text-xs">
            <div className="flex justify-between gap-2">
              <span className="truncate">{label(key)}</span>
              <span className="tabular-nums text-muted-foreground">
                {formatNumber(count)}
              </span>
            </div>
            <div className="h-1 rounded bg-muted">
              <div
                className="h-1 rounded bg-primary"
                style={{
                  width: `${total ? Math.max(2, (count / total) * 100) : 0}%`,
                }}
              />
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
