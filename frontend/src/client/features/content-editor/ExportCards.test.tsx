// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import {
  getBlogTheme,
  MAX_THEME_CSS_BYTES,
  saveBlogTheme,
} from "./contentEditorApi";
import { BlogThemeCard } from "./ExportCards";

vi.mock("./contentEditorApi", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  getBlogTheme: vi.fn(),
  saveBlogTheme: vi.fn(),
}));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let host: HTMLDivElement;
let root: Root;
let client: QueryClient;

beforeEach(async () => {
  vi.mocked(getBlogTheme).mockReset();
  vi.mocked(saveBlogTheme).mockReset();
  vi.mocked(getBlogTheme).mockResolvedValue({
    css: "h1{color:red}",
    updatedAt: "2026-11-09T10:00:00Z",
  });
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  client = new QueryClient();
  await act(async () => {
    root.render(
      <QueryClientProvider client={client}>
        <BlogThemeCard projectId="p1" />
      </QueryClientProvider>,
    );
  });
  // Let the theme query resolve and render the editor.
  await settle();
});

afterEach(() => {
  act(() => root.unmount());
  host.remove();
  client.clear();
});

// Runs a UI action and lets queries, mutations and renders settle.
async function settle(action: () => void = () => {}) {
  await act(async () => {
    action();
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
}

function button(label: string) {
  return [...host.querySelectorAll("button")].find(
    (item) => item.textContent === label,
  )!;
}

async function type(value: string) {
  const area = host.querySelector("textarea")!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(
      HTMLTextAreaElement.prototype,
      "value",
    )!.set!.call(area, value);
    area.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

it("starts from the saved theme and saves a change once, even on a double click", async () => {
  expect(host.querySelector("textarea")!.value).toBe("h1{color:red}");
  expect(host.textContent).toContain("Saved · 0.0 KB");
  expect(button("Save theme").disabled).toBe(true);
  let finish!: (value: { css: string; updatedAt: string }) => void;
  vi.mocked(saveBlogTheme).mockReturnValue(
    new Promise((resolve) => {
      finish = resolve;
    }),
  );
  await type("h1{color:blue}");
  const save = button("Save theme");
  await settle(() => save.click());
  expect(save.textContent).toBe("Saving…");
  expect(save.disabled).toBe(true);
  await settle(() => save.click());
  expect(saveBlogTheme).toHaveBeenCalledTimes(1);
  expect(saveBlogTheme).toHaveBeenCalledWith({
    projectId: "p1",
    css: "h1{color:blue}",
  });
  await settle(() =>
    finish({ css: "h1{color:blue}", updatedAt: "2026-11-09T10:05:00Z" }),
  );
  expect(host.querySelector("textarea")!.value).toBe("h1{color:blue}");
  expect(getBlogTheme).toHaveBeenCalledTimes(1);
});

it("blocks a theme over 100 KB and does not retry a failed save", async () => {
  await type("a".repeat(MAX_THEME_CSS_BYTES + 1));
  expect(host.querySelector('[role="alert"]')!.textContent).toContain(
    "over 100 KB",
  );
  expect(button("Save theme").disabled).toBe(true);
  vi.mocked(saveBlogTheme).mockRejectedValue(new Error("Paste CSS only."));
  await type("a".repeat(MAX_THEME_CSS_BYTES));
  await settle(() => button("Save theme").click());
  expect(saveBlogTheme).toHaveBeenCalledTimes(1);
});

it("removes the theme with an empty save", async () => {
  vi.mocked(saveBlogTheme).mockResolvedValue({ css: "", updatedAt: null });
  await settle(() => button("Remove").click());
  expect(saveBlogTheme).toHaveBeenCalledWith({ projectId: "p1", css: "" });
  expect(host.querySelector("textarea")!.value).toBe("");
  expect(host.textContent).not.toContain("Saved ·");
});
