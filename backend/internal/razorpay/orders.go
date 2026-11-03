package razorpay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Order is a one-time payment request, priced by the server in currency subunits.
type Order struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
}

// CreateOrder requests a one-time order of amount subunits in currency ("INR"
// or "USD") and verifies the returned amount and currency.
func (c *Client) CreateOrder(ctx context.Context, receipt string, amount int64, currency string) (Order, error) {
	if receipt == "" || len(receipt) > 40 || amount <= 0 || (currency != "INR" && currency != "USD") {
		return Order{}, errors.New("invalid top-up order")
	}
	raw, err := json.Marshal(map[string]any{"amount": amount, "currency": currency, "receipt": receipt, "partial_payment": false})
	if err != nil {
		return Order{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/orders", bytes.NewReader(raw))
	if err != nil {
		return Order{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.keyID, c.keySecret)
	var order Order
	if err = c.do(req, &order); err != nil {
		return Order{}, fmt.Errorf("create top-up order: %w", err)
	}
	if order.ID == "" || order.Amount != amount || order.Currency != currency || order.Receipt != receipt {
		return Order{}, errors.New("payment provider returned an unexpected order")
	}
	return order, nil
}
