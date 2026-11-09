package razorpay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Payment is the part of a Razorpay payment billing reads.
type Payment struct {
	ID       string `json:"id"`
	OrderID  string `json:"order_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
	Captured bool   `json:"captured"`
}

// PaymentAttempt is one payment attempt as Razorpay lists it, failed ones
// included, for the staff console.
type PaymentAttempt struct {
	ID               string `json:"id"`
	OrderID          string `json:"order_id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	Method           string `json:"method"`
	Email            string `json:"email"`
	ErrorReason      string `json:"error_reason"`
	ErrorDescription string `json:"error_description"`
	CreatedAt        int64  `json:"created_at"`
}

// maxListPages bounds one listing at 1,000 items.
const maxListPages = 10

// listAll reads every item of a Razorpay collection at path, 100 per page,
// up to maxListPages. truncated is true when there were more.
func listAll[T any](ctx context.Context, c *Client, path string, q url.Values) (items []T, truncated bool, err error) {
	for page := range maxListPages {
		q.Set("count", "100")
		q.Set("skip", strconv.Itoa(page*100))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+q.Encode(), nil)
		if err != nil {
			return nil, false, fmt.Errorf("build %s request: %w", path, err)
		}
		req.SetBasicAuth(c.keyID, c.keySecret)
		var res struct {
			Items []T `json:"items"`
		}
		if err := c.do(req, &res); err != nil {
			return nil, false, fmt.Errorf("list %s: %w", path, err)
		}
		items = append(items, res.Items...)
		if len(res.Items) < 100 {
			return items, false, nil
		}
	}
	return items, true, nil
}

// window is Razorpay's from/to filter, in Unix seconds.
func window(from, to time.Time) url.Values {
	return url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(to.Unix(), 10)}}
}

// ListPayments reads every payment attempt created in [from, to), newest
// first, up to 1,000. It only reads; Razorpay does not charge for API calls.
func (c *Client) ListPayments(ctx context.Context, from, to time.Time) ([]PaymentAttempt, bool, error) {
	return listAll[PaymentAttempt](ctx, c, "/payments", window(from, to))
}

// Refund is one refund of a payment. Amount is in the currency's subunits.
type Refund struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"`
}

// ListRefunds reads every refund created in [from, to), up to 1,000. It only
// reads.
func (c *Client) ListRefunds(ctx context.Context, from, to time.Time) ([]Refund, bool, error) {
	return listAll[Refund](ctx, c, "/refunds", window(from, to))
}

// CapturePayment captures an authorized payment for exactly amount subunits
// (paise, cents) in currency, the currency of the order. Razorpay auto-captures only payments authorized within its window
// (12 minutes on this account); a later authorization stays uncaptured and
// is refunded days later unless it is captured. A payment that is already
// captured is not an error: the capture already happened.
func (c *Client) CapturePayment(ctx context.Context, id string, amount int64, currency string) (Payment, error) {
	if id == "" || amount <= 0 || (currency != "INR" && currency != "USD") {
		return Payment{}, errors.New("invalid capture request")
	}
	var payment Payment
	err := c.post(ctx, "/payments/"+url.PathEscape(id)+"/capture", map[string]any{"amount": amount, "currency": currency}, &payment)
	if apiErr, ok := errors.AsType[*APIError](err); ok && apiErr.StatusCode == http.StatusBadRequest && apiErr.Code == "BAD_REQUEST_ERROR" && alreadyCaptured(apiErr.Description) {
		return Payment{ID: id, Amount: amount, Currency: currency, Status: "captured", Captured: true}, nil
	}
	if err != nil {
		return Payment{}, fmt.Errorf("capture payment %s: %w", id, err)
	}
	if payment.ID != id || payment.Amount != amount || payment.Currency != currency || !payment.Captured {
		return Payment{}, fmt.Errorf("capture payment %s: unexpected answer %+v", id, payment)
	}
	return payment, nil
}

// RefundPayment refunds payment id in full. Razorpay refuses a refund above
// what is left to refund, so repeating it cannot pay the customer twice.
func (c *Client) RefundPayment(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("invalid refund request")
	}
	var refund struct {
		ID string `json:"id"`
	}
	if err := c.post(ctx, "/payments/"+url.PathEscape(id)+"/refund", map[string]any{}, &refund); err != nil {
		return fmt.Errorf("refund payment %s: %w", id, err)
	}
	if refund.ID == "" {
		return fmt.Errorf("refund payment %s: response has no refund id", id)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, body, dst any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.keyID, c.keySecret)
	return c.do(req, dst)
}

// alreadyCaptured matches Razorpay's answer to capturing a captured payment,
// "This payment has already been captured".
func alreadyCaptured(description string) bool {
	return strings.Contains(strings.ToLower(description), "already been captured")
}
