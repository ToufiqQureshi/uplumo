# Billing (customer side): flow

1. **Plan:** `plan/UpgradeButton.tsx` posts `/api/v1/billing/checkout` (`billing.CheckoutPlan`), which hands back the workspace's unpaid subscription for the same plan and currency, or creates one. `plan/razorpay.ts` opens Razorpay Checkout on it.
2. **Top-up / extra website:** `credits/TopUpsCard.tsx` posts `/api/v1/billing/topups/checkout`, which reuses a pending order or creates one, then opens Razorpay Checkout with the pack's name.
3. Checkout ends in one of three outcomes (`CheckoutOutcome`):
   - `paid`: Razorpay's success handler ran. The plan button polls `/api/v1/billing/status` (`waitForPaidPlan`); the top-up card polls `/api/v1/billing/topups/{id}`. The grant itself comes only from the signed webhook.
   - `failed`: at least one attempt raised `payment.failed` and the customer closed the window without a successful retry. The button or card shows "Payment didn't go through" with Razorpay's reason and **Try again**. On the plan button, billing status is refreshed once, so a late bank confirmation shows without a reload.
   - `closed`: closed before any attempt. Nothing is shown.
4. **Try again** repeats step 1 or 2. The server hands back the same unpaid subscription or pending order, so it never starts a second payable one.

## Paid calls per click

None to DataForSEO or OpenRouter. One Razorpay create call at most (subscription or order), and none when an unpaid one is reused. Razorpay API calls are free; nothing is retried automatically (`retry: false`).

## Payments and receipts

1. The Billing page shows **Payments and receipts** to owners and admins (`receipts/PaymentsCard.tsx`) when the workspace has at least one payment. It reads `GET /api/v1/billing/payments` (`billing.Service.Payments`) from `go_billing_payments`, newest first.
2. **View receipt** opens `/billing/receipts/$paymentId`, which reads `GET /api/v1/billing/payments/{id}/receipt`. A payment of another workspace answers 404. **Download PDF** opens the browser's print dialog; the print styles show only the receipt.
3. **Billing details** (name, address, country, tax ID) are read from `GET /api/v1/billing/details` and saved with `POST /api/v1/billing/details` into `go_billing_details`. Every receipt, old ones included, shows the current details.

Paid calls: none. Everything is read from Uplumo's own database; Razorpay is not called.
