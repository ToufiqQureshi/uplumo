// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ReportProblemForm } from "./ReportProblemForm";
import { readScreenshot, reportPath, submitReport } from "./reportApi";

vi.mock("./reportApi", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  submitReport: vi.fn(),
}));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let host: HTMLDivElement;
let root: Root;
let client: QueryClient;

beforeEach(async () => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  client = new QueryClient();
  vi.mocked(submitReport).mockReset();
  await act(async () => {
    root.render(
      <QueryClientProvider client={client}>
        <ReportProblemForm />
      </QueryClientProvider>,
    );
  });
});

afterEach(() => {
  act(() => root.unmount());
  host.remove();
  client.clear();
});

async function type(selector: string, value: string) {
  const element = host.querySelector<HTMLInputElement | HTMLTextAreaElement>(
    selector,
  )!;
  const prototype =
    element.tagName === "TEXTAREA"
      ? HTMLTextAreaElement.prototype
      : HTMLInputElement.prototype;
  await act(async () => {
    Object.getOwnPropertyDescriptor(prototype, "value")!.set!.call(
      element,
      value,
    );
    element.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

async function submit() {
  await act(async () => {
    host
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
}

it("keeps failed reports and their submission ID for manual retry, then confirms the ticket", async () => {
  vi.mocked(submitReport).mockRejectedValueOnce(
    new Error("Temporarily unavailable"),
  );
  await type("textarea", "The balance does not update after payment.");
  await type("#report-page", "/billing?token=private#secret");
  await submit();
  await vi.waitFor(() =>
    expect(host.textContent).toContain("Temporarily unavailable"),
  );
  expect(submitReport).toHaveBeenCalledTimes(1);
  expect(host.querySelector("textarea")!.value).toContain("balance");
  const first = vi.mocked(submitReport).mock.calls[0]?.[0];
  expect(first?.pagePath).toBe("/billing");
  expect(first?.description).toContain("balance");
  expect(first?.appVersion).toBeTruthy();
  vi.mocked(submitReport).mockResolvedValue({
    ticketId: "12345678-1234-4234-8234-123456789012",
  });
  await submit();
  await vi.waitFor(() =>
    expect(host.textContent).toContain("12345678-1234-4234-8234-123456789012"),
  );
  expect(vi.mocked(submitReport).mock.calls[1]?.[0].submissionId).toBe(
    first?.submissionId,
  );
  expect(host.querySelector("form")).toBeNull();
});

it("explains why a short description cannot be submitted", async () => {
  await type("textarea", "  404  ");
  const description = host.querySelector("textarea")!;
  const send = host.querySelector<HTMLButtonElement>("button[type=submit]")!;
  expect(send.disabled).toBe(false);
  await act(async () => send.click());
  expect(host.querySelector("[role=alert]")?.textContent).toContain(
    "at least 10 characters",
  );
  expect(document.activeElement).toBe(description);
  expect(submitReport).not.toHaveBeenCalled();
});

it("dispatches once when the same form is submitted twice before rendering", async () => {
  let resolve: ((result: { ticketId: string }) => void) | undefined;
  vi.mocked(submitReport).mockReturnValue(
    new Promise((done) => {
      resolve = done;
    }),
  );
  await type("textarea", "The keyword table does not load.");
  await act(async () => {
    const form = host.querySelector("form")!;
    form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    );
    form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    );
  });
  expect(submitReport).toHaveBeenCalledTimes(1);
  await act(async () =>
    resolve?.({ ticketId: "12345678-1234-4234-8234-123456789012" }),
  );
});

it("rejects unsafe paths and unsupported or oversized screenshots", async () => {
  expect(reportPath("https://other.example/path?token=secret")).toBe(
    "/support",
  );
  expect(reportPath("//other.example/path")).toBe("/support");
  expect(reportPath("/billing?token=secret#private")).toBe("/billing");
  await expect(
    readScreenshot(new File(["svg"], "unsafe.svg", { type: "image/svg+xml" })),
  ).rejects.toThrow("PNG or JPEG");
  await expect(
    readScreenshot(
      new File([new Uint8Array(1024 * 1024 + 1)], "large.png", {
        type: "image/png",
      }),
    ),
  ).rejects.toThrow("smaller than 1 MB");
});
