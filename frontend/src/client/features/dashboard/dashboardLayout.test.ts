import { describe, expect, it } from "vitest";
import {
  DASHBOARD_CARDS,
  moveCard,
  resolveLayout,
} from "@/client/features/dashboard/dashboardLayout";

const defaultIds = DASHBOARD_CARDS.map((card) => card.id);

describe("resolveLayout", () => {
  it("shows every card in default order when nothing is saved", () => {
    expect(resolveLayout(null)).toEqual(
      defaultIds.map((id) => ({ id, hidden: false })),
    );
  });

  it("keeps the saved order and hidden cards, then appends cards added later", () => {
    const out = resolveLayout([
      { id: "audit", hidden: false },
      { id: "gsc", hidden: true },
    ]);
    expect(out.slice(0, 2)).toEqual([
      { id: "audit", hidden: false },
      { id: "gsc", hidden: true },
    ]);
    expect(out.map((card) => card.id)).toEqual([
      "audit",
      "gsc",
      ...defaultIds.filter((id) => id !== "audit" && id !== "gsc"),
    ]);
    expect(out.slice(2).every((card) => !card.hidden)).toBe(true);
  });

  it("drops retired and repeated ids", () => {
    const out = resolveLayout([
      { id: "local_seo", hidden: false },
      { id: "rank_tracking", hidden: true },
      { id: "rank_tracking", hidden: false },
    ]);
    expect(out[0]).toEqual({ id: "rank_tracking", hidden: true });
    expect(out).toHaveLength(defaultIds.length);
  });
});

describe("moveCard", () => {
  const cards = resolveLayout(null);

  it("swaps with the neighbour", () => {
    const moved = moveCard(cards, 1, -1);
    expect(moved.slice(0, 2).map((card) => card.id)).toEqual([
      cards[1].id,
      cards[0].id,
    ]);
    expect(cards[0].id).toBe(defaultIds[0]);
  });

  it("does not move past either end", () => {
    expect(moveCard(cards, 0, -1)).toBe(cards);
    expect(moveCard(cards, cards.length - 1, 1)).toBe(cards);
  });
});
