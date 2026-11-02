// @vitest-environment happy-dom
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { expect, it } from "vitest";
import { OverviewBreakdown } from "./OverviewBreakdown";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function render(props: Parameters<typeof OverviewBreakdown>[0]) {
  const host = document.createElement("div");
  const root = createRoot(host);
  act(() => root.render(createElement(OverviewBreakdown, props)));
  return host;
}

it("shows the paid position bands and movement, with unknown as '-'", () => {
  const host = render({
    positions: { top1: 13, top2To3: 29, top4To10: 62, top11To20: 1200 },
    movement: { new: 195, up: 28, down: 38, lost: null },
  });
  const cells = [...host.querySelectorAll("dd")].map((d) => d.textContent);
  expect(cells).toEqual([
    "13",
    "29",
    "62",
    (1200).toLocaleString(),
    "195",
    "28",
    "38",
    "-",
  ]);
});

it("renders nothing for an older cached overview without the fields", () => {
  expect(render({}).innerHTML).toBe("");
});
