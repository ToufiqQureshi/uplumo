import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { RenderingWarnings } from "./RenderingWarnings";
import type { AuditResultsData } from "../auditApi";

function results(renderJavaScript: boolean): AuditResultsData {
  return {
    audit: {
      id: "audit",
      startUrl: "https://example.test",
      status: "completed",
      pagesCrawled: 0,
      pagesTotal: 0,
      startedAt: "2026-10-20T00:00:00Z",
      completedAt: "2026-10-20T00:01:00Z",
      config: { maxPages: 10, lighthouseStrategy: "none", renderJavaScript },
    },
    pages: [],
    lighthouse: [],
    issues: [
      {
        id: "shell",
        pageId: null,
        pageUrl: "https://example.test",
        issueType: "javascript-rendering-suspected",
        severity: "warning",
        detailsJson: null,
      },
    ],
  };
}

describe("audit rendering guidance", () => {
  it.each([false, true])(
    "offers usable guidance for render request %s",
    (rendered) => {
      const html = renderToStaticMarkup(
        createElement(RenderingWarnings, { data: results(rendered) }),
      );
      expect(html).toContain("checks initial HTML");
      expect(html).toContain("browser");
      expect(html).not.toContain("Turn on Render JavaScript");
      expect(html).not.toContain("rendering was enabled");
      if (rendered)
        expect(html).toContain("does not prove JavaScript was executed");
    },
  );
});
