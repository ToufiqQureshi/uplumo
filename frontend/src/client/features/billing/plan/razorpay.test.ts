import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  failureReason,
  openRazorpayCheckout,
  openTopUpCheckout,
  paymentFailedMessage,
} from "./razorpay";

type Script = (checkout: FakeCheckout) => void;

/** A Razorpay Checkout stand-in that plays `script` once opened. */
class FakeCheckout {
  static script: Script = () => {};
  static last: FakeCheckout | null = null;
  failed: Array<(failure: unknown) => void> = [];
  constructor(
    public options: {
      description: string;
      handler: () => void;
      modal: { ondismiss: () => void };
    },
  ) {
    FakeCheckout.last = this;
  }
  on(event: string, cb: (failure: unknown) => void) {
    if (event === "payment.failed") this.failed.push(cb);
  }
  open() {
    FakeCheckout.script(this);
  }
  fail(failure: unknown) {
    for (const cb of this.failed) cb(failure);
  }
  pay() {
    this.options.handler();
  }
  close() {
    this.options.modal.ondismiss();
  }
}

const session = { keyId: "rzp_test_fake", subscriptionId: "sub_fake" };
const timedOut = {
  error: {
    description:
      "Payment was unsuccessful as you could not complete it in time.",
    reason: "payment_timed_out",
  },
};

beforeEach(() => {
  FakeCheckout.script = () => {};
  FakeCheckout.last = null;
  vi.stubGlobal("window", { Razorpay: FakeCheckout });
});
afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Razorpay checkout outcome", () => {
  it.each<[string, Script, unknown]>([
    ["paid at once", (c) => c.pay(), { status: "paid" }],
    ["closed without trying", (c) => c.close(), { status: "closed" }],
    [
      "failed, then closed",
      (c) => {
        c.fail(timedOut);
        c.close();
      },
      { status: "failed", reason: timedOut.error.description },
    ],
    [
      "failed, then paid on Razorpay's own retry",
      (c) => {
        c.fail(timedOut);
        c.pay();
      },
      { status: "paid" },
    ],
    [
      "failed twice, then closed: reports the last failure",
      (c) => {
        c.fail({ error: { description: "Card declined." } });
        c.fail({ error: { description: "Bank server down." } });
        c.close();
      },
      { status: "failed", reason: "Bank server down." },
    ],
  ])("%s", async (_name, script, want) => {
    FakeCheckout.script = script;
    await expect(openRazorpayCheckout(session)).resolves.toEqual(want);
  });

  it("names the top-up being bought, not a fixed product", async () => {
    FakeCheckout.script = (c) => c.close();
    await openTopUpCheckout(
      {
        keyId: "rzp_test_fake",
        orderId: "order_fake",
        amount: 99900,
        currency: "INR",
      },
      "1 more website",
    );
    expect(FakeCheckout.last?.options.description).toBe("1 more website");
  });

  it("fails clearly when the Razorpay script did not start", async () => {
    vi.stubGlobal("window", { Razorpay: undefined });
    vi.stubGlobal("document", {
      createElement: () => ({
        addEventListener: (event: string, cb: () => void) => {
          if (event === "load") cb();
        },
      }),
      head: { appendChild: () => {} },
    });
    await expect(openRazorpayCheckout(session)).rejects.toThrow(
      "Razorpay didn't start",
    );
  });
});

describe("failure copy", () => {
  it.each([
    [{}, "The payment was not completed."],
    [{ error: {} }, "The payment was not completed."],
    [{ error: { description: "   " } }, "The payment was not completed."],
    [{ error: { description: " UPI app declined. " } }, "UPI app declined."],
    [{ error: { description: "Paiement refusé 💳" } }, "Paiement refusé 💳"],
  ])("reason of %o", (failure, want) => {
    expect(failureReason(failure)).toBe(want);
  });

  it("tells the customer not to pay again and what happens to the money", () => {
    const message = paymentFailedMessage("Timed out.", "Uplumo Business");
    expect(message).toContain("Timed out.");
    expect(message).toContain("don't need to pay again");
    expect(message).toContain("Uplumo Business turns on by itself");
    expect(message).toContain("bank returns the money");
  });
});
