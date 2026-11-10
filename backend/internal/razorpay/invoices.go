package razorpay

import (
	"context"
	"fmt"
	"net/url"
)

// Invoice is a paid or unpaid charge of a subscription. Amounts are in the
// currency's subunits; PaidAt is Unix seconds, nil until paid.
type Invoice struct {
	ID         string `json:"id"`
	PaymentID  string `json:"payment_id"`
	Status     string `json:"status"`
	AmountPaid int64  `json:"amount_paid"`
	Currency   string `json:"currency"`
	PaidAt     *int64 `json:"paid_at"`
}

// ListSubscriptionInvoices reads the invoices of one subscription, up to
// 1,000. It only reads; Razorpay does not charge for API calls.
func (c *Client) ListSubscriptionInvoices(ctx context.Context, subscriptionID string) ([]Invoice, error) {
	invoices, _, err := listAll[Invoice](ctx, c, "/invoices", url.Values{"subscription_id": {subscriptionID}})
	if err != nil {
		return nil, fmt.Errorf("list invoices of subscription %s: %w", subscriptionID, err)
	}
	return invoices, nil
}
