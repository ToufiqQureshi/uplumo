const CHECKOUT_SCRIPT = "https://checkout.razorpay.com/v1/checkout.js";

/** What Razorpay's `payment.failed` event carries that we show. */
interface RazorpayFailure {
  error?: { description?: string; reason?: string };
}

interface RazorpayOptions {
  key: string;
  subscription_id?: string;
  order_id?: string;
  amount?: number;
  currency?: string;
  name: string;
  description: string;
  theme: { color: string };
  handler: () => void;
  modal: { ondismiss: () => void };
}

interface RazorpayInstance {
  open: () => void;
  on: (event: "payment.failed", cb: (failure: RazorpayFailure) => void) => void;
}

declare global {
  interface Window {
    Razorpay?: new (options: RazorpayOptions) => RazorpayInstance;
  }
}

/**
 * How the checkout window ended. `failed` means at least one attempt failed
 * and the customer then closed the window without a successful retry.
 */
type CheckoutOutcome =
  | { status: "paid" }
  | { status: "closed" }
  | { status: "failed"; reason: string };

function loadCheckoutScript(): Promise<void> {
  if (window.Razorpay) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = CHECKOUT_SCRIPT;
    script.async = true;
    script.addEventListener("load", () => resolve());
    script.addEventListener("error", () => {
      // Drop the failed tag so the next attempt loads the script again.
      script.remove();
      reject(
        new Error(
          "Couldn't load Razorpay. Check your connection or ad blocker and try again.",
        ),
      );
    });
    document.head.appendChild(script);
  });
}

/** Razorpay's own description of the failure, or a plain fallback. */
export function failureReason(failure: RazorpayFailure): string {
  const description = failure.error?.description?.trim();
  return description || "The payment was not completed.";
}

/**
 * What to tell a customer whose payment failed. A timed-out payment can
 * still be confirmed by the bank later (the server webhook then grants
 * `item`), otherwise the bank returns any debited money on its own.
 */
export function paymentFailedMessage(reason: string, item: string): string {
  return `${reason} If money left your account, you don't need to pay again: ${item} turns on by itself once your bank confirms the payment, or your bank returns the money, usually within 5 to 7 working days.`;
}

/**
 * Opens Razorpay Checkout. Razorpay keeps its window open after a failed
 * attempt so the customer can retry there; the last failure is reported
 * only if they then close the window without paying.
 */
async function openCheckout(
  options: Omit<RazorpayOptions, "handler" | "modal" | "name" | "theme">,
): Promise<CheckoutOutcome> {
  await loadCheckoutScript();
  const Razorpay = window.Razorpay;
  if (!Razorpay)
    throw new Error("Razorpay didn't start. Reload and try again.");
  return new Promise((resolve) => {
    let lastFailure: string | null = null;
    const checkout = new Razorpay({
      ...options,
      name: "Uplumo",
      theme: { color: "#0f766e" },
      handler: () => resolve({ status: "paid" }),
      modal: {
        ondismiss: () =>
          resolve(
            lastFailure === null
              ? { status: "closed" }
              : { status: "failed", reason: lastFailure },
          ),
      },
    });
    checkout.on("payment.failed", (failure) => {
      lastFailure = failureReason(failure);
    });
    checkout.open();
  });
}

/** Opens Razorpay Checkout for a subscription. */
export function openRazorpayCheckout(
  session: { keyId: string; subscriptionId: string },
  planName = "Pro",
): Promise<CheckoutOutcome> {
  return openCheckout({
    key: session.keyId,
    subscription_id: session.subscriptionId,
    description: `Uplumo ${planName}, billed monthly`,
  });
}

/** Browser success is only a hint; the signed server webhook confirms the grant. */
export function openTopUpCheckout(
  session: {
    keyId: string;
    orderId: string;
    amount: number;
    currency: string;
  },
  description: string,
): Promise<CheckoutOutcome> {
  return openCheckout({
    key: session.keyId,
    order_id: session.orderId,
    amount: session.amount,
    currency: session.currency,
    description,
  });
}
