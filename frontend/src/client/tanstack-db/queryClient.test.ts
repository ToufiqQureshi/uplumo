import { describe, expect, it, vi } from "vitest";
import { toast } from "sonner";
import { queryClient } from "@/client/tanstack-db/queryClient";
import { QueryObserver } from "@tanstack/query-core";

vi.mock("sonner", () => ({ toast: { error: vi.fn() } }));

function runFailingMutation(
  options: { onError?: () => void; meta?: { errorToast?: false } } = {},
) {
  return queryClient
    .getMutationCache()
    .build(queryClient, {
      mutationFn: () => Promise.reject(new Error("RATE_LIMITED")),
      ...options,
    })
    .execute(undefined)
    .catch(() => undefined);
}

describe("default mutation error toast", () => {
  it("shows the standard message for a mutation without its own handling", async () => {
    await runFailingMutation();
    expect(toast.error).toHaveBeenCalledWith(
      "Too many requests. Please wait and try again.",
    );
  });

  it("stays quiet when the mutation has its own onError or opts out", async () => {
    await runFailingMutation({ onError: () => undefined });
    await runFailingMutation({ meta: { errorToast: false } });
    expect(toast.error).not.toHaveBeenCalled();
  });
});

describe("queryClient", () => {
  it("keeps a failed purchase idle on remount and permits an explicit retry", async () => {
    const queryKey = ["paid-remount-regression"];
    const queryFn = vi.fn(() =>
      Promise.reject(new Error("provider unavailable")),
    );
    const first = new QueryObserver(queryClient, { queryKey, queryFn });
    const unsubscribeFirst = first.subscribe(() => undefined);
    await vi.waitFor(() => expect(first.getCurrentResult().isError).toBe(true));
    unsubscribeFirst();
    const second = new QueryObserver(queryClient, { queryKey, queryFn });
    const unsubscribeSecond = second.subscribe(() => undefined);
    try {
      await new Promise((resolve) => setTimeout(resolve, 0));
      expect(queryFn).toHaveBeenCalledTimes(1);
      await second.refetch();
      expect(queryFn).toHaveBeenCalledTimes(2);
    } finally {
      unsubscribeSecond();
      queryClient.removeQueries({ queryKey });
    }
  });

  it("never retries queries by default so a paid call cannot double-charge", () => {
    const retry = queryClient.getDefaultOptions().queries?.retry;
    expect(retry).toBe(false);
  });
});
