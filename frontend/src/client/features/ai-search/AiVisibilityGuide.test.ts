import { createElement, type ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { AiVisibilityGuide } from "./AiVisibilityGuide";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, to }: { children: ReactNode; to: string }) =>
    createElement("a", { href: to }, children),
}));

describe("AiVisibilityGuide", () => {
  it("explains all three tools, marks this page and links the other two", () => {
    const html = renderToStaticMarkup(
      createElement(AiVisibilityGuide, {
        current: "prompt-explorer",
        projectId: "p1",
      }),
    );
    expect(html).toContain("Prompt Explorer (this page)");
    expect(html).toContain('href="/p/$projectId/brand-lookup"');
    expect(html).toContain('href="/p/$projectId/prompt-tracking"');
    expect(html).not.toContain('href="/p/$projectId/prompt-explorer"');
    expect(html).toContain("re-check them on a schedule");
  });
});
