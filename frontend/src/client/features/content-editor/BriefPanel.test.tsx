import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BillingUsage } from "@/client/features/billing/usageApi";
import { BriefPanel } from "./BriefPanel";
import { EditorSessionProvider } from "./editor-session";

const usageState: {
  data: BillingUsage | undefined;
  isLoading: boolean;
  isError: boolean;
} = { data: undefined, isLoading: true, isError: false };

vi.mock("@/client/features/billing/usageApi", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useBillingUsage: () => usageState,
}));

const paidUsage: BillingUsage = {
  plan: "pro",
  unlimited: false,
  chargesCredits: true,
  balanceCredits: 7420,
  monthlyCredits: 10000,
  periodStart: "2026-10-01T00:00:00Z",
  periodEnd: null,
  spent: [],
  costs: [
    {
      feature: "keyword_research",
      action: "content_brief",
      unit: "search brief",
      credits: 9,
      maxCredits: 10,
    },
  ],
};

function renderBrief() {
  return renderToStaticMarkup(
    createElement(
      QueryClientProvider,
      { client: new QueryClient() },
      createElement(
        EditorSessionProvider,
        null,
        createElement(BriefPanel, {
          projectId: "project-1",
          keyword: "seo",
          content: "",
          brief: null,
          onBrief: () => {},
        }),
      ),
    ),
  );
}

describe("BriefPanel credit quote", () => {
  beforeEach(() => {
    usageState.data = paidUsage;
    usageState.isLoading = false;
    usageState.isError = false;
  });

  it("shows the server quote before enabling brief generation", () => {
    const html = renderBrief();
    const button = html.match(/<button[^>]*>/)?.[0] ?? "";

    expect(html).toContain("Up to 10 credits for a new brief");
    expect(html).toContain("7,420 left");
    expect(button).not.toMatch(/\sdisabled(?:\s|=|>)/);
  });

  it("keeps the paid action disabled until its quote has loaded", () => {
    usageState.data = undefined;
    usageState.isLoading = true;
    const html = renderBrief();

    expect(html).toContain("Loading credit quote");
    expect(html.match(/<button[^>]*>/)?.[0]).toMatch(/\sdisabled(?:\s|=|>)/);
  });

  it("keeps the paid action disabled when the server has no quote", () => {
    usageState.data = { ...paidUsage, costs: [] };
    const html = renderBrief();

    expect(html).toContain("Credit quote unavailable");
    expect(html.match(/<button[^>]*>/)?.[0]).toMatch(/\sdisabled(?:\s|=|>)/);
  });
});
