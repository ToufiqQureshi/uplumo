// @vitest-environment happy-dom
import { act, createElement, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BillingUsage } from "@/client/features/billing/usageApi";
import { useConfirmedCost } from "./useConfirmedCost";

const usageState: { data: BillingUsage | undefined } = { data: undefined };

vi.mock("@/client/features/billing/usageApi", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useBillingUsage: () => usageState,
}));

vi.mock("@/client/components/ConfirmDialog", () => ({
  ConfirmDialog: ({ title }: { title: string; children: ReactNode }) =>
    createElement("div", { "data-testid": "confirm" }, title),
}));

const usage: BillingUsage = {
  plan: "pro",
  unlimited: false,
  chargesCredits: true,
  balanceCredits: 9590,
  monthlyCredits: 10000,
  periodStart: "2026-10-01T00:00:00Z",
  periodEnd: null,
  spent: [],
  costs: [
    {
      feature: "ai_visibility",
      action: "brand_lookup",
      unit: "brand lookup",
      credits: 909,
    },
    {
      feature: "ai_visibility",
      action: "prompt_explorer",
      unit: "prompt",
      credits: 128,
    },
  ],
};

function mount(action: string) {
  const run = vi.fn();
  let request: ((run: () => void) => void) | undefined;
  function Probe() {
    const confirm = useConfirmedCost("ai_visibility", action, "lookup");
    request = confirm.request;
    return createElement("div", null, confirm.dialog);
  }
  const container = document.createElement("div");
  act(() => createRoot(container).render(createElement(Probe)));
  act(() => request?.(run));
  return { run, container };
}

describe("useConfirmedCost", () => {
  beforeEach(() => {
    usageState.data = usage;
  });

  it("asks before an expensive action and does not run it yet", () => {
    const { run, container } = mount("brand_lookup");
    expect(run).not.toHaveBeenCalled();
    expect(container.textContent).toContain("Use about 909 credits?");
  });

  it("shows the most the action can charge when the server sends it", () => {
    usageState.data = {
      ...usage,
      costs: [{ ...usage.costs[0], maxCredits: 2240 }],
    };
    expect(mount("brand_lookup").container.textContent).toContain(
      "Use about 909 credits (at most 2,240)?",
    );
  });

  it("runs a cheap action at once", () => {
    const { run, container } = mount("prompt_explorer");
    expect(run).toHaveBeenCalledOnce();
    expect(container.textContent).toBe("");
  });

  it("runs at once where credits are not charged", () => {
    usageState.data = { ...usage, unlimited: true };
    expect(mount("brand_lookup").run).toHaveBeenCalledOnce();
  });

  it("runs at once while usage is loading, so the action is never blocked", () => {
    usageState.data = undefined;
    expect(mount("brand_lookup").run).toHaveBeenCalledOnce();
  });
});
