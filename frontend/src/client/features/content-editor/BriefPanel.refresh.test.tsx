// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { BillingUsage } from "@/client/features/billing/usageApi";
import { FreeSeoEditorProvider } from "./FreeSeoEditorContext";
import { BriefPanel } from "./BriefPanel";
import { fetchSearchBrief, type SearchBrief } from "./contentEditorApi";
import { EditorSessionProvider } from "./editor-session";

vi.mock("@/client/features/billing/usageApi", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useBillingUsage: () => ({ data: usage, isLoading: false, isError: false }),
}));
vi.mock("./contentEditorApi", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  fetchSearchBrief: vi.fn(),
}));

const usage: BillingUsage = {
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

const brief: SearchBrief = {
  keyword: "seo",
  locationCode: 2356,
  languageCode: "en",
  fetchedAt: "2026-10-15T10:00:00Z",
  questions: [],
  competitors: [],
  terms: [],
  subtopics: [],
};

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let host: HTMLDivElement;
let root: Root;
let client: QueryClient;

beforeEach(async () => {
  vi.mocked(fetchSearchBrief).mockReset();
  vi.mocked(fetchSearchBrief).mockResolvedValue(brief);
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  client = new QueryClient();
  await act(async () => {
    root.render(
      <QueryClientProvider client={client}>
        <EditorSessionProvider>
          <FreeSeoEditorProvider onDraftFieldChange={() => {}}>
            <BriefPanel
              projectId="project-1"
              keyword="seo"
              content=""
              brief={brief}
              onBrief={() => {}}
            />
          </FreeSeoEditorProvider>
        </EditorSessionProvider>
      </QueryClientProvider>,
    );
  });
});

afterEach(() => {
  act(() => root.unmount());
  host.remove();
  client.clear();
});

function button(label: string) {
  return [...document.querySelectorAll("button")].find(
    (item) => item.textContent?.trim() === label,
  );
}

async function click(label: string) {
  await act(async () => {
    button(label)!.click();
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
}

it("buys a refreshed brief only after the in-app confirmation", async () => {
  await click("Refresh");
  expect(document.body.textContent).toContain("up to 10 credits");
  expect(fetchSearchBrief).not.toHaveBeenCalled();
  await click("Cancel");
  expect(button("Refresh brief")).toBeUndefined();
  expect(fetchSearchBrief).not.toHaveBeenCalled();

  await click("Refresh");
  await click("Refresh brief");
  expect(fetchSearchBrief).toHaveBeenCalledTimes(1);
  expect(fetchSearchBrief).toHaveBeenCalledWith({
    data: { projectId: "project-1", keyword: "seo", refresh: true },
  });
});
