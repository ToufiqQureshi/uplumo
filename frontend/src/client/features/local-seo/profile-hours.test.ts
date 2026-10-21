// @vitest-environment happy-dom
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { getLocalBusinessProfile } from "./localSeoApi";
import { LocalSEOPage } from "./LocalSEOPage";

vi.mock("./localSeoApi", () => ({
  getLocalBusinessProfile: vi.fn(),
  findLocalBusinesses: vi.fn(),
  runLocalGrid: vi.fn(),
}));
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

let root: Root;
let host: HTMLDivElement;
beforeEach(() => {
  vi.resetAllMocks();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
});

async function lookup(profile: Record<string, unknown>) {
  vi.mocked(getLocalBusinessProfile).mockResolvedValue({ profile });
  act(() => root.render(createElement(LocalSEOPage, { projectId: "project" })));
  for (const [label, value] of [
    ["Your business name", "Lake clinic"],
    ["Center latitude", "18.75"],
    ["Center longitude", "73.4"],
  ]) {
    const labelElement = [...host.querySelectorAll("label")].find((el) =>
      el.textContent?.includes(label),
    );
    const input = labelElement!.querySelector("input")!;
    act(() => {
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value",
      )!.set!.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  }
  const button = [...host.querySelectorAll("button")].find((el) =>
    el.textContent?.includes("Audit business profile"),
  )!;
  expect(button.disabled).toBe(false);
  await act(async () => button.click());
  expect(getLocalBusinessProfile).toHaveBeenCalledTimes(1);
}

it("renders hours from the documented business-info work_time object", async () => {
  await lookup({
    title: "Lake clinic",
    work_time: {
      work_hours: {
        timetable: { monday: [{ open: { hour: 9, minute: 0 } }] },
        current_status: "open",
      },
    },
  });
  expect(host.textContent).toContain("work hours");
  expect(host.textContent).toContain('"current_status":"open"');
  expect(host.textContent).toContain('"monday"');
});

it("retains legacy root hours when nested hours are absent", async () => {
  await lookup({ work_hours: { current_status: "closed" } });
  expect(host.textContent).toContain('"current_status":"closed"');
});

it("prefers current nested hours over a legacy root value", async () => {
  await lookup({
    work_time: { work_hours: { current_status: "open" } },
    work_hours: { current_status: "closed" },
  });
  expect(host.textContent).toContain('"current_status":"open"');
  expect(host.textContent).not.toContain('"current_status":"closed"');
});
