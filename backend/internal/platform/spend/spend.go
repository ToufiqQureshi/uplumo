// Package spend is the contract between work that costs Uplumo money at a
// provider and the credit ledger that pays for it. A provider transport asks a
// Guard for a Hold before it sends a billed request, and settles the hold with
// the real cost afterwards. Nothing billed may be sent without a hold.
package spend

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

var (
	// ErrNoCredits means the workspace has not got the credits the work needs.
	ErrNoCredits = errors.New("not enough credits for this request")
	// ErrPaused means Uplumo stopped all paid provider work for the day: the
	// day's total spend reached the safety limit.
	ErrPaused = errors.New("paid data requests are paused for today")
	// ErrWorkspaceDailyLimit means this workspace used its share of credits
	// for today; other workspaces are not affected.
	ErrWorkspaceDailyLimit = errors.New("this workspace reached its daily credit limit")
	// ErrPlanLimits means a downgrade left retained data over the active
	// plan's limits. Reads and cleanup stay available, but paid calls stop.
	ErrPlanLimits = errors.New("workspace data exceeds the active plan limits")
	// ErrIdempotentReplay means a paid request with this idempotency key was already submitted.
	ErrIdempotentReplay = errors.New("request was already submitted")
)

// Feature names what the money is spent on. It labels the spend in the
// ledger, so the Billing screen can show where credits went.
type Feature string

// Features that spend credits.
const (
	RankTracking    Feature = "rank_tracking"
	KeywordResearch Feature = "keyword_research"
	DomainResearch  Feature = "domain_research"
	Backlinks       Feature = "backlinks"
	AIVisibility    Feature = "ai_visibility"
	SiteAudit       Feature = "site_audit"
	AIAssistant     Feature = "ai_assistant"
	AIWriting       Feature = "ai_writing"
	// Reference is free lookup data such as location lists. It is held and
	// recorded like any other request, but never needs credits.
	Reference Feature = "reference"
)

// Features lists every feature that can spend credits, in the order the
// Billing screen shows them.
func Features() []Feature {
	return []Feature{RankTracking, KeywordResearch, DomainResearch, Backlinks, AIVisibility, SiteAudit, AIAssistant, AIWriting}
}

// Request is one billed piece of provider work about to be sent.
type Request struct {
	OrganizationID string
	Feature        Feature
	Endpoint       string
	IdempotencyKey string
	// EstimateUSD is the most the request can cost at the provider. It is
	// held before the request is sent, so it must not be an average.
	EstimateUSD float64
}

type (
	idempotencyKeyContextKey struct{}
	userIDContextKey         struct{}
)

// idempotencyScope is one user action's key and how often each piece of its
// work has been held so far. It is shared by every context derived from the
// one WithIdempotencyKey returned, so parallel legs of an action count together.
type idempotencyScope struct {
	key  string
	mu   sync.Mutex
	uses map[[sha256.Size]byte]int
}

// WithIdempotencyKey attaches the key of one user action (the browser's
// Idempotency-Key) to ctx. Every hold made for that action derives its own key
// from it with ScopedKey; the key itself is never used as a hold key, because
// one action can legitimately buy several different results.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKeyContextKey{}, &idempotencyScope{key: key, uses: map[[sha256.Size]byte]int{}})
}

// IdempotencyKeyFromContext reads the action key previously attached to ctx.
func IdempotencyKeyFromContext(ctx context.Context) string {
	scope, _ := ctx.Value(idempotencyKeyContextKey{}).(*idempotencyScope)
	if scope == nil {
		return ""
	}
	return scope.key
}

// ScopedKey is the idempotency key for one piece of billed work inside the
// action in ctx, identified by parts (for example the method, path and body of
// a provider request). It is "" when ctx carries no action key, so background
// work without a key is never deduplicated.
//
// The same work in a replay of the same action always gets the same key, so a
// retry cannot buy it twice. Different work in one action gets different keys,
// so one action can buy a summary and its history, or nine map points. Identical
// work repeated inside one action is numbered, so the second copy is its own
// purchase on the first run and the same two keys again on a replay.
func ScopedKey(ctx context.Context, parts ...[]byte) string {
	scope, _ := ctx.Value(idempotencyKeyContextKey{}).(*idempotencyScope)
	if scope == nil || scope.key == "" {
		return ""
	}
	digest := sha256.New()
	digest.Write([]byte(scope.key))
	var length [binary.MaxVarintLen64]byte
	for _, part := range parts {
		digest.Write(length[:binary.PutUvarint(length[:], uint64(len(part)))])
		digest.Write(part)
	}
	var sum [sha256.Size]byte
	digest.Sum(sum[:0])
	scope.mu.Lock()
	occurrence := scope.uses[sum]
	scope.uses[sum] = occurrence + 1
	scope.mu.Unlock()
	return hex.EncodeToString(sum[:]) + "." + strconv.Itoa(occurrence)
}

type creditMeterContextKey struct{}

// WithCreditMeter makes ctx count the credits settled for it, so the HTTP layer
// can tell the customer what the action cost them. Contexts derived from the
// returned one (also detached ones) share the count.
func WithCreditMeter(ctx context.Context) context.Context {
	return context.WithValue(ctx, creditMeterContextKey{}, new(atomic.Int64))
}

// RecordCredits adds credits that were charged for ctx's action. It does
// nothing when ctx has no meter.
func RecordCredits(ctx context.Context, credits int64) {
	if meter, ok := ctx.Value(creditMeterContextKey{}).(*atomic.Int64); ok && credits > 0 {
		meter.Add(credits)
	}
}

// CreditsCharged is the credits charged so far for ctx's action.
func CreditsCharged(ctx context.Context) int64 {
	if meter, ok := ctx.Value(creditMeterContextKey{}).(*atomic.Int64); ok {
		return meter.Load()
	}
	return 0
}

// WithUserID attaches the authenticated user for credit ledger attribution.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// UserIDFromContext reads the authenticated user previously attached to ctx.
func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDContextKey{}).(string)
	return userID
}

// Hold is credit set aside for one request.
type Hold interface {
	// Settle charges the real provider cost and releases the rest. Call it
	// exactly once, also when the request failed (with the cost the provider
	// reported, often zero).
	Settle(ctx context.Context, costUSD float64) error
}

// ProviderCostHold records the provider's actual cost separately from the
// customer charge when a provider billed work that failed to return a result.
type ProviderCostHold interface {
	SettleProviderCost(ctx context.Context, providerCostUSD, customerCostUSD float64, endpoint string) error
}

// SettleProviderCost records provider spend separately from the customer charge when supported.
func SettleProviderCost(ctx context.Context, hold Hold, providerCostUSD, customerCostUSD float64, endpoint string) error {
	if providerHold, ok := hold.(ProviderCostHold); ok {
		return providerHold.SettleProviderCost(ctx, providerCostUSD, customerCostUSD, endpoint)
	}
	return hold.Settle(ctx, customerCostUSD)
}

// Guard decides whether billed work may start.
type Guard interface {
	// Hold sets credit aside for req, or refuses with ErrNoCredits or
	// ErrPaused. Any other error must also stop the request: a guard that
	// cannot decide fails closed.
	Hold(ctx context.Context, req Request) (Hold, error)
}

// Refused reports whether err is the guard refusing the work: no credits,
// or paid work paused for the day.
func Refused(err error) bool {
	return errors.Is(err, ErrIdempotentReplay) || errors.Is(err, ErrNoCredits) || errors.Is(err, ErrPaused) || errors.Is(err, ErrWorkspaceDailyLimit) || errors.Is(err, ErrPlanLimits)
}

const (
	noCreditsMessage  = "You have used this month's credits. They refill on your next billing date, or upgrade your plan for more."
	pausedMessage     = "Live data is paused for a short while. Please try again in a few hours."
	dailyMessage      = "You have used today's share of your credits. It resets at 5:30 AM IST, and your monthly credits are safe."
	planLimitsMessage = "Your workspace is over its current plan limits. Your data is safe; archive sites or remove team members or tracked keywords, then try again."
	duplicateMessage  = "This request was already submitted. Refresh the page before trying again."
)

// Message is what to tell the customer about a refusal, or "" when err is
// not one.
func Message(err error) string {
	switch {
	case errors.Is(err, ErrNoCredits):
		return noCreditsMessage
	case errors.Is(err, ErrPaused):
		return pausedMessage
	case errors.Is(err, ErrWorkspaceDailyLimit):
		return dailyMessage
	case errors.Is(err, ErrPlanLimits):
		return planLimitsMessage
	case errors.Is(err, ErrIdempotentReplay):
		return duplicateMessage
	}
	return ""
}

// WriteHTTPError answers a refused request and reports whether err was one:
// 402 insufficient_credits, 429 daily_credit_limit, or 503 usage_paused.
func WriteHTTPError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, ErrIdempotentReplay):
		httpx.WriteError(w, http.StatusConflict, "duplicate_request", duplicateMessage)
	case errors.Is(err, ErrNoCredits):
		httpx.WriteError(w, http.StatusPaymentRequired, "insufficient_credits", noCreditsMessage)
	case errors.Is(err, ErrPaused):
		httpx.WriteError(w, http.StatusServiceUnavailable, "usage_paused", pausedMessage)
	case errors.Is(err, ErrWorkspaceDailyLimit):
		httpx.WriteError(w, http.StatusTooManyRequests, "daily_credit_limit", dailyMessage)
	case errors.Is(err, ErrPlanLimits):
		httpx.WriteError(w, http.StatusPaymentRequired, "plan_limits_exceeded", planLimitsMessage)
	default:
		return false
	}
	return true
}
