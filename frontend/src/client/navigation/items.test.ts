import { describe, expect, it } from "vitest";
import { getProjectNavGroups } from "@/client/navigation/items";

describe("getProjectNavGroups", () => {
  const items = getProjectNavGroups("p1").flatMap((group) => group.items);
  const pages = items.flatMap((item) => item.tabs ?? [item]);

  it("keeps every page reachable exactly once from the sidebar or a tab", () => {
    const routes = pages.map((page) => String(page.link.to));
    expect(new Set(routes).size).toBe(routes.length);
    expect(routes).toHaveLength(17);
  });

  it("shows twelve sidebar rows", () => {
    expect(items).toHaveLength(11);
  });

  it("opens a tabbed row on its first tab", () => {
    for (const item of items.filter((row) => row.tabs)) {
      expect(item.link.to).toBe(item.tabs![0].link.to);
    }
  });
});
