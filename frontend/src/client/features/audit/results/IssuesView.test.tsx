// @vitest-environment happy-dom
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { IssuesView } from "./IssuesView";
import type { AuditResultsData } from "./types";
import { importPageDraft } from "@/client/features/content-editor/contentEditorApi";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));
vi.mock("@tanstack/react-router", async (original) => ({
  ...(await original<object>()),
  useParams: () => ({ projectId: "p1" }),
  useNavigate: () => navigate,
}));
vi.mock(
  "@/client/features/content-editor/contentEditorApi",
  async (original) => ({
    ...(await original<object>()),
    importPageDraft: vi.fn(),
  }),
);

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function auditPage(url: string, statusCode: number | null) {
  return {
    id: url,
    url,
    statusCode,
    fetchClass: "ok" as const,
    redirectUrl: null,
    title: null,
    metaDescription: null,
    canonicalUrl: null,
    robotsMeta: null,
    xRobotsTag: null,
    headerCanonicalUrl: null,
    ogTitle: null,
    ogDescription: null,
    ogImage: null,
    h1Count: 0,
    h2Count: 0,
    h3Count: 0,
    h4Count: 0,
    h5Count: 0,
    h6Count: 0,
    headingOrder: [],
    wordCount: 0,
    contentHash: null,
    imagesTotal: 0,
    imagesMissingAlt: 0,
    images: [],
    internalLinkCount: 0,
    externalLinkCount: 0,
    hasStructuredData: false,
    hreflangTags: [],
    isIndexable: true,
    crawlDepth: 0,
    inSitemap: true,
    responseTimeMs: 10,
  };
}

const live = "https://example.com/guide";
const gone = "https://example.com/old";
const pages: AuditResultsData["pages"] = [
  auditPage(live, 200),
  auditPage(gone, 404),
];
const issues = [live, gone].map((pageUrl) => ({
  id: pageUrl,
  pageId: pageUrl,
  pageUrl,
  issueType: "missing-title",
  severity: "critical" as const,
  detailsJson: null,
}));

let root: Root;
let host: HTMLDivElement;
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(importPageDraft).mockResolvedValue({
    id: "draft-guide",
    projectId: "p1",
    title: "Guide",
    targetKeyword: "",
    slug: "guide",
    metaDescription: "",
    content: "Text",
    createdAt: "2026-11-13T00:00:00Z",
    updatedAt: "2026-11-13T00:00:00Z",
  });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  act(() =>
    root.render(
      createElement(
        QueryClientProvider,
        { client: new QueryClient() },
        createElement(IssuesView, { issues, pages, tabs: null }),
      ),
    ),
  );
  const group = [...host.querySelectorAll("button")].find((b) =>
    b.textContent?.includes("Missing title tag"),
  );
  act(() => group?.click());
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
});

const editButtons = () =>
  [...host.querySelectorAll("button")].filter((b) =>
    b.getAttribute("aria-label")?.startsWith("Edit:"),
  );

it("offers the editor only for pages that loaded", () => {
  expect(editButtons().map((b) => b.getAttribute("aria-label"))).toEqual([
    `Edit: ${live}`,
  ]);
});

it("opens the page's draft in the editor in one click", async () => {
  await act(async () => editButtons()[0]?.click());
  expect(importPageDraft).toHaveBeenCalledTimes(1);
  expect(importPageDraft).toHaveBeenCalledWith("p1", live);
  await vi.waitFor(() =>
    expect(navigate).toHaveBeenCalledWith({
      to: "/p/$projectId/content-editor",
      params: { projectId: "p1" },
      search: { draft: "draft-guide" },
    }),
  );
});
