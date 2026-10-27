import type { AiSource, AnalyticsSummary, Channel } from "./analyticsApi";

const CHANNEL_LABELS: Record<Channel, { label: string; hint: string }> = {
  ai: {
    label: "AI assistants",
    hint: "ChatGPT, Perplexity, Gemini and others",
  },
  search: { label: "Search engines", hint: "Google, Bing and others" },
  social: { label: "Social media", hint: "Facebook, LinkedIn, X and others" },
  paid: { label: "Paid ads", hint: "Campaign-tagged paid traffic" },
  email: { label: "Email", hint: "Email campaigns and newsletters" },
  referral: { label: "Other websites", hint: "Links on other sites" },
  direct: { label: "Direct", hint: "Typed the address or used a bookmark" },
};

export function formatDuration(milliseconds: number): string {
  if (milliseconds < 1000) return "<1s";
  const totalSeconds = Math.floor(milliseconds / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

const AI_SOURCE_LABELS: Record<AiSource, string> = {
  chatgpt: "ChatGPT",
  perplexity: "Perplexity",
  gemini: "Gemini",
  claude: "Claude",
  copilot: "Copilot",
  other_ai: "Other AI tools",
};

// The assistants every owner asks about. Each gets a row even at zero, so
// "nobody came from Claude" reads as a fact rather than a missing row.
const NAMED_AI_SOURCES: AiSource[] = [
  "chatgpt",
  "perplexity",
  "gemini",
  "claude",
  "copilot",
];

export const DEVICE_LABELS = {
  desktop: "Desktop",
  mobile: "Mobile",
  tablet: "Tablet",
} as const;

export interface ShareRow {
  key: string;
  label: string;
  hint?: string;
  /** Visitors (AI sources) or visits (channels), as the caller labels it. */
  count: number;
  /** This row's part of the total, from 0 to 1. */
  share: number;
}

function withShares(rows: Omit<ShareRow, "share">[]): ShareRow[] {
  const total = rows.reduce((sum, row) => sum + row.count, 0);
  const shared = rows.map((row) => ({
    ...row,
    share: total === 0 ? 0 : row.count / total,
  }));
  // A fresh array, so sorting in place is safe. The sort is stable.
  shared.sort((a, b) => b.count - a.count);
  return shared;
}

/**
 * Visit sources, biggest first, with their share of all visits. Each visit
 * (session) has exactly one source, its first page's, so the shares add up to
 * every visit. Visitor-based channel counts cannot do that: one visitor-day
 * with two visits from two sources counts in both.
 */
export function channelRows(
  acquisition: AnalyticsSummary["acquisitionChannels"],
): ShareRow[] {
  return withShares(
    acquisition
      .filter((row) => row.sessions > 0)
      .map((row) => ({
        key: row.channel,
        ...CHANNEL_LABELS[row.channel],
        count: row.sessions,
      })),
  );
}

/**
 * Every named AI assistant (zero when absent) plus "Other AI tools" when it
 * has visitors, biggest first. Ties keep the named order.
 */
export function aiSourceRows(
  aiSources: AnalyticsSummary["aiSources"],
): ShareRow[] {
  const visitors = new Map(aiSources.map((row) => [row.source, row.visitors]));
  const otherAi = visitors.get("other_ai") ?? 0;
  const sources =
    otherAi > 0 ? [...NAMED_AI_SOURCES, "other_ai" as const] : NAMED_AI_SOURCES;
  return withShares(
    sources.map((source) => ({
      key: source,
      label: AI_SOURCE_LABELS[source],
      count: visitors.get(source) ?? 0,
    })),
  );
}

/** The AI channel's part of all visits, from 0 to 1. */
export function aiShareOfVisits(
  acquisition: AnalyticsSummary["acquisitionChannels"],
): number {
  return channelRows(acquisition).find((row) => row.key === "ai")?.share ?? 0;
}

// Plain names for the actions the tracker records on its own.
const ACTION_LABELS: Record<string, string> = {
  whatsapp_click: "WhatsApp",
  call_click: "Phone call",
  sms_click: "SMS",
  email_click: "Email",
  form_submit: "Form",
  file_download: "File download",
  outbound_click: "Link to another site",
};

/** An action's plain name: "WhatsApp" for whatsapp_click, else as tagged. */
export function actionLabel(name: string): string {
  return ACTION_LABELS[name] ?? name;
}

const percent = new Intl.NumberFormat("en-IN", {
  style: "percent",
  maximumFractionDigits: 0,
});

/** "42%", or "<1%" for a share that rounds to zero but is not zero. */
export function formatShare(share: number): string {
  if (share > 0 && share < 0.005) return "<1%";
  return percent.format(share);
}

/**
 * A page path as people read it. The API sends paths percent-encoded, so
 * "/%E0%A4%B9" shows as "/ह". A malformed escape shows as sent.
 */
export function displayPath(path: string): string {
  try {
    return decodeURI(path);
  } catch {
    return path;
  }
}
