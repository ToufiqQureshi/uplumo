package razorpay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

// maxInvoicePages bounds one subscription's history at 1,000 invoices.
const maxInvoicePages = 10

// ListSubscriptionInvoices reads every invoice of one subscription. It only
// reads; Razorpay does not charge for API calls.
func (c *Client) ListSubscriptionInvoices(ctx context.Context, subscriptionID string) ([]Invoice, error) {
	var all []Invoice
	for page := range maxInvoicePages {
		q := url.Values{"subscription_id": {subscriptionID}, "count": {"100"}, "skip": {strconv.Itoa(page * 100)}}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/invoices?"+q.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("build invoices request: %w", err)
		}
		req.SetBasicAuth(c.keyID, c.keySecret)
		var res struct {
			Items []Invoice `json:"items"`
		}
		if err := c.do(req, &res); err != nil {
			return nil, fmt.Errorf("list invoices of subscription %s: %w", subscriptionID, err)
		}
		all = append(all, res.Items...)
		if len(res.Items) < 100 {
			return all, nil
		}
	}
	return all, nil
}
