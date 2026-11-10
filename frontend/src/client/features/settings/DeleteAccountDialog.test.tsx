// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { DeleteAccountDialog } from "./DeleteAccountDialog";

vi.mock("@/lib/auth-client", () => ({ signOutAndRedirect: vi.fn() }));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

it("sends the optional reason once on confirmation and keeps a failure visible", async () => {
  const client = new QueryClient();
  const host = document.body.appendChild(document.createElement("div"));
  const root = createRoot(host);
  let finish: ((response: Response) => void) | undefined;
  const send = vi.spyOn(globalThis, "fetch").mockImplementation(
    () =>
      new Promise<Response>((resolve) => {
        finish = resolve;
      }),
  );
  try {
    await act(async () => {
      root.render(
        <QueryClientProvider client={client}>
          <DeleteAccountDialog open onOpenChange={vi.fn()} />
        </QueryClientProvider>,
      );
    });
    expect(send).not.toHaveBeenCalled();
    const reason = document.querySelector("select");
    expect(reason?.value).toBe("");
    await act(async () => {
      if (reason) reason.value = "missing_feature";
      reason?.dispatchEvent(new Event("change", { bubbles: true }));
    });
    const button = Array.from(document.querySelectorAll("button")).find(
      (element) => element.textContent === "Delete my account",
    );
    expect(button).toBeDefined();
    await act(async () => {
      button?.click();
      button?.click();
    });
    expect(send).toHaveBeenCalledExactlyOnceWith("/api/auth/delete-account", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: '{"confirm":true,"reason":"missing_feature"}',
    });
    await act(async () => {
      finish?.(
        Response.json(
          {
            code: "SHARED_WORKSPACE",
            message: "Leave shared workspaces first.",
          },
          { status: 409 },
        ),
      );
    });
    await vi.waitFor(() =>
      expect(document.querySelector("[role=alert]")?.textContent).toBe(
        "Leave shared workspaces first.",
      ),
    );
  } finally {
    await act(async () => root.unmount());
    host.remove();
    client.clear();
    send.mockRestore();
  }
});
