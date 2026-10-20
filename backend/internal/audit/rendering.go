package audit

import "context"

// IsAuditRenderingAllowed is shared by capabilities and start validation.
// Environment hints alone cannot enable rendering: the Go crawler currently
// fetches HTTP HTML and has no JavaScript renderer wired into execution.
func IsAuditRenderingAllowed(_ func(string) string) bool { return false }

// RenderUsage counts the provider units one audit's rendering used.
type RenderUsage struct {
	CloudflareAttempts int `json:"cloudflareAttempts"`
	ContextCredits     int `json:"contextCredits"`
}

// RenderingLock is one held credit reservation.
type RenderingLock struct {
	LockID           string `json:"lockId"`
	FeatureID        string `json:"featureId"`
	EstimatedCredits int    `json:"estimatedCredits"`
}

// RenderingMeter holds and settles rendering credits around a hosted audit.
// Self-hosted deployments pass nil, where rendering is unmetered.
type RenderingMeter interface {
	// Lock reserves the worst-case rendering cost. It returns ErrPaymentRequired
	// (wrapped) when the balances cannot cover it.
	Lock(ctx context.Context, organizationID, auditID string, maxPages int) ([]RenderingLock, error)
	// Release returns every held credit for an audit that never started.
	Release(ctx context.Context, locks []RenderingLock) error
	// Settle charges the credits used and releases the rest. It never fails a
	// caller: an unsettled hold expires on its own.
	Settle(ctx context.Context, organizationID, auditID string, locks []RenderingLock, usage RenderUsage) error
}
