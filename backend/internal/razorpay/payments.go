package razorpay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

// CapturePayment captures an authorized payment for exactly amount paise in
// INR. Razorpay auto-captures only payments authorized within its window
// (12 minutes on this account); a later authorization stays uncaptured and
// is refunded days later unless it is captured. A payment that is already
// captured is not an error: the capture already happened.
func (c *Client) CapturePayment(ctx context.Context, id string, amount int64) (Payment, error) {
	if id == "" || amount <= 0 {
		return Payment{}, errors.New("invalid capture request")
	}
	var payment Payment
	err := c.post(ctx, "/payments/"+url.PathEscape(id)+"/capture", map[string]any{"amount": amount, "currency": "INR"}, &payment)
	if apiErr, ok := errors.AsType[*APIError](err); ok && apiErr.StatusCode == http.StatusBadRequest && apiErr.Code == "BAD_REQUEST_ERROR" && alreadyCaptured(apiErr.Description) {
		return Payment{ID: id, Amount: amount, Currency: "INR", Status: "captured", Captured: true}, nil
	}
	if err != nil {
		return Payment{}, fmt.Errorf("capture payment %s: %w", id, err)
	}
	if payment.ID != id || payment.Amount != amount || payment.Currency != "INR" || !payment.Captured {
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
