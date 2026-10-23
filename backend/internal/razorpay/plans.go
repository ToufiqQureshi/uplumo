package razorpay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Plan is the provider's recurring amount and frequency, never client input.
type Plan struct {
	ID       string `json:"id"`
	Period   string `json:"period"`
	Interval int    `json:"interval"`
	Item     struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"item"`
}

// FetchPlan reads the commercial terms before billing creates a subscription.
func (c *Client) FetchPlan(ctx context.Context, id string) (Plan, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/plans/"+url.PathEscape(id), nil)
	if err != nil {
		return Plan{}, fmt.Errorf("build plan request: %w", err)
	}
	req.SetBasicAuth(c.keyID, c.keySecret)
	var plan Plan
	if err := c.do(req, &plan); err != nil {
		return Plan{}, fmt.Errorf("fetch plan: %w", err)
	}
	if plan.ID != id {
		return Plan{}, fmt.Errorf("plan response does not match requested ID")
	}
	return plan, nil
}
