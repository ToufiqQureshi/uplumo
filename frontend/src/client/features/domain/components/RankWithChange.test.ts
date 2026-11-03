// @vitest-environment happy-dom
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { expect, it } from "vitest";
import { RankWithChange } from "./RankWithChange";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function text(props: Parameters<typeof RankWithChange>[0]) {
  const host = document.createElement("div");
  const root = createRoot(host);
  act(() => root.render(createElement(RankWithChange, props)));
  return host.textContent;
}

it.each([
  [{ position: 3, previousPosition: 7 }, "3↑4"],
  [{ position: 9, previousPosition: 4 }, "9↓5"],
  [{ position: 5, isNew: true }, "5New"],
  [{ position: 5, previousPosition: 5 }, "5"],
  [{ position: 5 }, "5"],
  [{ position: null, previousPosition: 2 }, "-"],
])("%o renders %s", (props, want) => {
  expect(text(props)).toBe(want);
});
