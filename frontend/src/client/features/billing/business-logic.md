# Billing (customer side): decisions

- **The browser never grants anything.** Razorpay's success handler only starts polling; the plan or pack turns on from the signed webhook. _Why:_ a closed tab or a forged callback must not change what a workspace has.
- **A failed payment is explained, not hidden.** Razorpay keeps its window open after a failure so the customer can retry there; when they close it instead, Uplumo shows Razorpay's reason, says not to pay again if money was taken, and offers **Try again**. _Why:_ a UPI timeout (owner's own attempt, 8 Nov 2026) left the customer with no word from Uplumo.
- **The copy promises only what happens.** A timed-out payment the bank confirms later is granted by the webhook (subscriptions activate, late top-up authorizations are captured by `billing.captureTopUpPayment`); otherwise the bank returns the money. No refund timing beyond "usually 5 to 7 working days".
- **Retry reuses the unpaid checkout.** _Why:_ a second payable subscription or order could charge the customer twice.
- **Razorpay's checkout description names what is bought** (the plan, or the pack name), never a fixed product. _Why:_ an extra-website purchase was described as "Additional usage credits".
