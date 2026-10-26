import { sortBy } from "remeda";
import type { AnalyticsSummary } from "./analyticsApi";

type EventRow = AnalyticsSummary["events"][number];

// Event names that mean a visitor tried to reach the business.
const ENQUIRY =
  /whats\s*app|wa_|call|phone|tel|form|lead|contact|enquir|inquir|book|quote|submit|sign.?up|purchase|checkout|order/i;

/**
 * Events that differ only by letter case ("whatsapp", "WhatsApp") are one
 * action tagged twice. Merge them, keeping the most used spelling.
 */
export function mergeEvents(events: EventRow[]) {
  const groups = new Map<string, EventRow[]>();
  for (const event of events) {
    const key = event.name.trim().toLowerCase();
    groups.set(key, [...(groups.get(key) ?? []), event]);
  }
  const merged = [...groups.values()].map((rows) => ({
    name: sortBy(rows, [(row) => row.count, "desc"])[0]?.name ?? "",
    count: rows.reduce((total, row) => total + row.count, 0),
    // Visitor-days can overlap across spellings, so take the largest, not a sum.
    visitors: Math.max(...rows.map((row) => row.visitors)),
    spellings: rows.length,
  }));
  return sortBy(merged, [(row) => row.count, "desc"]);
}

/** Enquiry-like actions (WhatsApp, calls, forms, bookings) and their total. */
export function enquiries(events: EventRow[]) {
  const rows = mergeEvents(events).filter((row) => ENQUIRY.test(row.name));
  return { rows, total: rows.reduce((sum, row) => sum + row.count, 0) };
}

/**
 * Why the numbers cannot be trusted yet, if they cannot: pageviews arrive but
 * no visits are stitched together (old or broken snippet).
 */
export function trackingProblem(summary: AnalyticsSummary) {
  if (summary.pageviews > 0 && summary.sessions === 0) return "no_sessions";
  return null;
}
