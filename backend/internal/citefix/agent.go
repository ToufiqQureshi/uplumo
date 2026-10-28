package citefix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// The agent's modes and prices. The credit caps are owner decisions (27 Oct
// 2026): the most one page can ever cost the customer, shown before the click.
const (
	ModeQuick = "quick"
	ModeDeep  = "deep"

	QuickMaxCredits int64 = 25
	DeepMaxCredits  int64 = 60
	// MaxAgentPages is how many pages one batch may hold and how many runs a
	// project may start in agentRateWindow; DefaultAgentPages are preselected.
	MaxAgentPages     = 20
	DefaultAgentPages = 5

	// AnalystModel reads the page and plans the fixes (both modes);
	// WriterModel rewrites the page (Deep only). Pinned: the ceilings below
	// are computed from their prices.
	AnalystModel = "deepseek/deepseek-v4.1-flash"
	WriterModel  = "deepseek/deepseek-v4-pro"
)

// Price caps sent to OpenRouter as provider max_price (USD per million
// tokens), so no provider can bill above what the hold assumed. They sit at
// the top of the mainstream providers' prices on 27 Oct 2026; one Flash
// provider asked $30.91 per million output tokens.
var (
	analystPrice = openrouter.MaxPrice{Prompt: 0.30, Completion: 1.20}
	writerPrice  = openrouter.MaxPrice{Prompt: 1.30, Completion: 2.60}
)

const (
	analystMaxTokens = 3_000
	writerMaxTokens  = 6_000
	// maxPageTextBytes bounds the page text sent to the models.
	maxPageTextBytes = 16_000
	maxPageHTMLBytes = 2 << 20
	fetchTimeout     = 15 * time.Second
	contextTimeout   = 10 * time.Second
	callTimeout      = 120 * time.Second
	settleWait       = 10 * time.Second
	// agentRateWindow with MaxAgentPages stops a loop or a script from
	// buying the same project's pages again and again.
	agentRateWindow = 10 * time.Minute
	// maxUnpaidFailures runs per workspace may fail without a charge in
	// unpaidFailureWindow. An unusable answer costs the customer nothing but
	// Uplumo pays the model, so a page built to break the agent must not buy
	// free model calls without end. Restart-stale runs do not count.
	maxUnpaidFailures   = 5
	unpaidFailureWindow = 24 * time.Hour
	// staleRunAfter frees a page whose run was cut off (a restart) so it
	// can be run again.
	staleRunAfter = 10 * time.Minute
	maxGSCQueries = 15
	// promptVersion is part of the cache key: changing the prompts or models
	// must not serve an answer written for the old ones.
	promptVersion = "citefix-agent-v1"
)

// Agent errors the handler maps to HTTP.
var (
	ErrAgentUnavailable  = errors.New("cite fix agent is not configured")
	ErrAgentPlanRequired = errors.New("the cite fix agent needs a paid plan")
	ErrAgentPageNotFound = errors.New("page is not in the latest scan")
	ErrAgentBusy         = errors.New("this page is already being worked on")
	ErrAgentRateLimited  = errors.New("too many agent runs for this project")
	ErrAgentPageFetch    = errors.New("the page could not be read")
	ErrAgentFailed       = errors.New("the agent could not finish this page")
)

// AgentValidationError is a bad request.
type AgentValidationError struct{ Message string }

func (e *AgentValidationError) Error() string { return e.Message }

// AgentOffer tells the page whether the agent can run and what it costs.
type AgentOffer struct {
	Available       bool  `json:"available"`
	PlanAllowed     bool  `json:"planAllowed"`
	QuickMaxCredits int64 `json:"quickMaxCredits"`
	DeepMaxCredits  int64 `json:"deepMaxCredits"`
	MaxPages        int   `json:"maxPages"`
	DefaultPages    int   `json:"defaultPages"`
}

// AgentInput is one page to work on.
type AgentInput struct {
	URL            string `json:"url"`
	Mode           string `json:"mode"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// AgentResult is what the agent produced for one page.
type AgentResult struct {
	URL       string    `json:"url"`
	Mode      string    `json:"mode"`
	Cached    bool      `json:"cached"`
	Credits   int64     `json:"credits"`
	CreatedAt time.Time `json:"createdAt"`
	// Partial is true when Deep's rewrite failed and the result is Quick's.
	Partial bool `json:"partial"`
	AgentOutput
}

// AgentOutput is the model's work, validated and trimmed.
type AgentOutput struct {
	Summary         string           `json:"summary"`
	TargetQuestion  string           `json:"targetQuestion"`
	Recommendations []Recommendation `json:"recommendations"`
	AnswerFirst     string           `json:"answerFirst"`
	FAQ             []QA             `json:"faq"`
	TitleTag        string           `json:"titleTag"`
	MetaDescription string           `json:"metaDescription"`
	SchemaJSONLD    string           `json:"schemaJsonLd"`
	Verify          []string         `json:"verify"`
	// Draft is Markdown for the Content Editor: Deep's full rewrite, or
	// Quick's answer and FAQ.
	Draft string `json:"draft"`
}

// Recommendation is one concrete change for this page.
type Recommendation struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Check  string `json:"check"`
}

// QA is one FAQ item.
type QA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// LLM streams one completion.
type LLM interface {
	Stream(context.Context, openrouter.Request, openrouter.Handler) (openrouter.Result, error)
}

// AgentMeter holds credits before the model calls.
type AgentMeter interface {
	HoldWithIdempotencyKey(ctx context.Context, organizationID string, ceilingUSD float64, key string) (spend.Hold, error)
}

// AgentPlans reports whether the workspace's plan includes the agent.
type AgentPlans interface {
	HasPaidPlan(ctx context.Context, organizationID string) (bool, error)
}

// PageQueries reads a page's Search Console queries. *gsc.Service is one.
type PageQueries interface {
	GetPageQueryRows(ctx context.Context, org, project, start, end, page string) ([]gsc.SearchRow, error)
}

// Agent runs the paid Cite Fix agent on one page at a time.
type Agent struct {
	Store   Store
	LLM     LLM
	Meter   AgentMeter
	Plans   AgentPlans
	Queries PageQueries
	// HTTP fetches the customer's page; build it with audit.Guard so a page
	// that resolves to a private address is refused.
	HTTP   *http.Client
	Logger *slog.Logger
	Now    func() time.Time
}

// Offer describes the agent for a workspace. A nil agent is unavailable.
func (a *Agent) Offer(ctx context.Context, organizationID string) (AgentOffer, error) {
	offer := AgentOffer{
		QuickMaxCredits: QuickMaxCredits, DeepMaxCredits: DeepMaxCredits,
		MaxPages: MaxAgentPages, DefaultPages: DefaultAgentPages,
	}
	if a == nil || a.LLM == nil {
		return offer, nil
	}
	offer.Available = true
	allowed, err := a.Plans.HasPaidPlan(ctx, organizationID)
	if err != nil {
		return offer, fmt.Errorf("check cite fix agent plan: %w", err)
	}
	offer.PlanAllowed = allowed
	return offer, nil
}

// Run works on one page. Same page content and mode as an earlier completed
// run: that answer again, 0 credits. Otherwise one hold covers both model
// calls and the customer pays the provider's cost, never above the hold,
// which is never above the mode's cap.
func (a *Agent) Run(ctx context.Context, organizationID, projectID string, input AgentInput) (AgentResult, error) {
	if a == nil || a.LLM == nil {
		return AgentResult{}, ErrAgentUnavailable
	}
	if err := validateAgentInput(input); err != nil {
		return AgentResult{}, err
	}
	allowed, err := a.Plans.HasPaidPlan(ctx, organizationID)
	if err != nil {
		return AgentResult{}, fmt.Errorf("check cite fix agent plan: %w", err)
	}
	if !allowed {
		return AgentResult{}, ErrAgentPlanRequired
	}
	if replay, ok, err := a.Store.runByKey(ctx, organizationID, input.IdempotencyKey); err != nil {
		return AgentResult{}, err
	} else if ok {
		return replay, nil
	}
	target, err := a.Store.agentPage(ctx, organizationID, projectID, input.URL)
	if err != nil {
		return AgentResult{}, err
	}
	page, err := a.fetchPage(ctx, input.URL)
	if err != nil {
		return AgentResult{}, err
	}
	hash := contentHash(page)
	if cached, ok, err := a.Store.cachedRun(ctx, projectID, input.URL, input.Mode, hash); err != nil {
		return AgentResult{}, err
	} else if ok {
		cached.Cached, cached.Credits = true, 0
		if err := a.Store.recordCachedRun(ctx, organizationID, projectID, input, hash, cached); err != nil {
			return AgentResult{}, err
		}
		return cached, nil
	}
	runID, err := a.Store.claimRun(ctx, organizationID, projectID, input, hash, a.now())
	if err != nil {
		return AgentResult{}, err
	}
	result, credits, errCode, runErr := a.work(ctx, organizationID, projectID, input, target, page)
	if runErr != nil {
		if err := a.Store.failRun(context.WithoutCancel(ctx), runID, errCode, credits); err != nil {
			runErr = errors.Join(runErr, err)
		}
		return AgentResult{}, runErr
	}
	result.URL, result.Mode, result.Credits, result.CreatedAt = input.URL, input.Mode, credits, a.now()
	if err := a.Store.completeRun(context.WithoutCancel(ctx), runID, credits, result); err != nil {
		return AgentResult{}, err
	}
	return result, nil
}

// Results lists the project's saved agent results; reading them is free.
func (a *Agent) Results(ctx context.Context, organizationID, projectID string) ([]AgentResult, error) {
	return a.Store.Results(ctx, organizationID, projectID)
}

// fetchedPage is the live page the models read.
type fetchedPage struct {
	Title, MetaDescription string
	H1s                    []string
	Text                   string
}

// fetchPage reads the page as it is now, through the SSRF-guarded client.
// A redirect or an error page means the scan is out of date.
func (a *Agent) fetchPage(ctx context.Context, pageURL string) (fetchedPage, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return fetchedPage{}, fmt.Errorf("%w: %w", ErrAgentPageFetch, err)
	}
	req.Header.Set("User-Agent", audit.UserAgent)
	req.Header.Set("Accept", "text/html")
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return fetchedPage{}, fmt.Errorf("%w: %w", ErrAgentPageFetch, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "html") {
		return fetchedPage{}, fmt.Errorf("%w: status %d, %s", ErrAgentPageFetch, resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	body, err := audit.ReadTextUpTo(resp, maxPageHTMLBytes)
	if err != nil {
		return fetchedPage{}, fmt.Errorf("%w: %w", ErrAgentPageFetch, err)
	}
	analysis := audit.AnalyzeHTML(body, pageURL, resp.StatusCode, 0, "")
	if strings.TrimSpace(analysis.BodyText) == "" {
		return fetchedPage{}, fmt.Errorf("%w: no text on the page", ErrAgentPageFetch)
	}
	return fetchedPage{
		Title: analysis.Title, MetaDescription: analysis.MetaDescription, H1s: analysis.H1s,
		Text: truncateBytes(analysis.BodyText, maxPageTextBytes),
	}, nil
}

// work holds credits, calls the models and settles. credits is what was
// charged (also on failure: zero unless a partial result was delivered).
func (a *Agent) work(ctx context.Context, organizationID, projectID string, input AgentInput, target agentTarget, page fetchedPage) (AgentResult, int64, string, error) {
	evidence := a.evidence(ctx, organizationID, projectID, input.URL)
	analystReq := analystRequest(input.Mode, target, page, evidence)
	ceiling, err := holdCeiling(input.Mode, analystReq, target, page)
	if err != nil {
		return AgentResult{}, 0, "ceiling", err
	}
	hold, err := a.Meter.HoldWithIdempotencyKey(ctx, organizationID, ceiling, "citefix:"+input.IdempotencyKey)
	if err != nil {
		return AgentResult{}, 0, "hold", fmt.Errorf("hold cite fix agent credits: %w", err)
	}

	var providerUSD float64
	analystCtx, cancel := context.WithTimeout(ctx, callTimeout)
	analystResult, err := a.LLM.Stream(analystCtx, analystReq, openrouter.Handler{})
	cancel()
	providerUSD += callCost(analystResult, analystReq, analystPrice)
	output, parseErr := parseAnalystOutput(analystResult.Text)
	if err != nil || parseErr != nil {
		// Nothing usable reached the customer: the provider cost is ours.
		settleErr := a.settle(ctx, hold, providerUSD, 0)
		return AgentResult{}, 0, "analyst", errors.Join(fmt.Errorf("%w: analyst: %w", ErrAgentFailed, errors.Join(err, parseErr)), settleErr)
	}
	customerUSD := providerUSD
	result := AgentResult{AgentOutput: output}
	result.Draft = quickDraft(page, output)

	if input.Mode == ModeDeep {
		writerReq := writerRequest(target, page, analystResult.Text)
		writerCtx, cancel := context.WithTimeout(ctx, callTimeout)
		writerResult, err := a.LLM.Stream(writerCtx, writerReq, openrouter.Handler{})
		cancel()
		writerUSD := callCost(writerResult, writerReq, writerPrice)
		providerUSD += writerUSD
		draft, parseErr := parseWriterOutput(writerResult.Text)
		if err != nil || parseErr != nil {
			// Deliver the analyst's work, charge only for it.
			result.Partial = true
			a.logger().WarnContext(ctx, "cite fix rewrite failed; delivering the quick result", "err", errors.Join(err, parseErr))
		} else {
			customerUSD += writerUSD
			result.Draft = draft
		}
	}
	if customerUSD > ceiling {
		a.logger().ErrorContext(ctx, "cite fix agent cost exceeded its hold", "provider_usd", providerUSD, "ceiling_usd", ceiling)
		customerUSD = ceiling
	}
	// Fail closed: a charge that cannot be recorded returns no result.
	if err := a.settle(ctx, hold, providerUSD, customerUSD); err != nil {
		return AgentResult{}, 0, "settle", fmt.Errorf("settle cite fix agent credits: %w", err)
	}
	return result, billing.CreditsForUSD(customerUSD), "", nil
}

// holdCeiling is the most a run can cost: the analyst call, and for Deep the
// rewrite, whose input is the analyst's prompt plus at most its whole answer.
// Above the mode's cap the run is refused before any credit is held: the
// customer must never pay more than the button showed.
func holdCeiling(mode string, analystReq openrouter.Request, target agentTarget, page fetchedPage) (float64, error) {
	ceiling := requestCeiling(analystReq, analystPrice)
	capUSD := creditsToUSD(QuickMaxCredits)
	if mode == ModeDeep {
		ceiling += requestCeiling(writerRequest(target, page, strings.Repeat("x", analystMaxTokens*3)), writerPrice)
		capUSD = creditsToUSD(DeepMaxCredits)
	}
	if ceiling > capUSD {
		return 0, fmt.Errorf("%w: ceiling $%.4f above the mode cap $%.4f", ErrAgentFailed, ceiling, capUSD)
	}
	return ceiling, nil
}

func (a *Agent) settle(ctx context.Context, hold spend.Hold, providerUSD, customerUSD float64) error {
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleWait)
	defer cancel()
	return spend.SettleProviderCost(settleCtx, hold, providerUSD, customerUSD, "openrouter/citefix")
}

// agentEvidence is the free context: Search Console queries and the AI
// answers that cite other sites. Either may be missing.
type agentEvidence struct {
	Queries []gsc.SearchRow
	Proof   *Proof
	Rivals  []RivalComparison
}

func (a *Agent) evidence(ctx context.Context, organizationID, projectID, pageURL string) agentEvidence {
	evidenceCtx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()
	var evidence agentEvidence
	if a.Queries != nil {
		end := a.now().AddDate(0, 0, -3)
		rows, err := a.Queries.GetPageQueryRows(evidenceCtx, organizationID, projectID, end.AddDate(0, 0, -90).Format(time.DateOnly), end.Format(time.DateOnly), pageURL)
		if err != nil {
			// No Search Console connection is normal; the page is still worked on.
			a.logger().InfoContext(ctx, "cite fix agent runs without Search Console queries", "err", err)
		}
		slices.SortFunc(rows, func(x, y gsc.SearchRow) int { return int(y.Impressions - x.Impressions) })
		evidence.Queries = rows[:min(len(rows), maxGSCQueries)]
	}
	proof, err := a.Store.proof(evidenceCtx, organizationID, projectID)
	if err != nil {
		a.logger().WarnContext(ctx, "cite fix agent runs without AI visibility evidence", "err", err)
	}
	evidence.Proof = proof
	if proof != nil && len(proof.Misses) > 0 {
		mentions, mentionErr := a.Store.mentions(evidenceCtx, organizationID, projectID)
		if mentionErr != nil {
			a.logger().WarnContext(ctx, "cite fix agent runs without saved Brand Lookup pages", "err", mentionErr)
		}
		var auditID string
		var pages []Page
		var legacy bool
		var scanErr error
		scanErr = a.Store.DB.QueryRow(evidenceCtx, `SELECT a.id FROM go_audits a JOIN projects p ON p.id=a.project_id AND p.organization_id=$1
			WHERE a.project_id=$2 AND a.status='completed' AND a.completed_at IS NOT NULL ORDER BY a.started_at DESC LIMIT 1`,
			organizationID, projectID).Scan(&auditID)
		if scanErr == nil {
			pages, legacy, scanErr = a.Store.pages(evidenceCtx, auditID)
		}
		if scanErr == nil && !legacy {
			rivalCtx, rivalCancel := context.WithTimeout(ctx, 40*time.Second)
			evidence.Rivals, scanErr = a.Store.rivalComparisons(rivalCtx, organizationID, projectID, proof, mentions, pages)
			rivalCancel()
		}
		if scanErr != nil {
			a.logger().WarnContext(ctx, "cite fix agent runs without rival page evidence", "err", scanErr)
		}
	}
	return evidence
}

func (a *Agent) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Agent) logger() *slog.Logger {
	if a.Logger != nil {
		return a.Logger
	}
	return slog.Default()
}

func validateAgentInput(input AgentInput) error {
	if input.Mode != ModeQuick && input.Mode != ModeDeep {
		return &AgentValidationError{Message: "Choose Quick or Deep."}
	}
	key := strings.TrimSpace(input.IdempotencyKey)
	if key == "" || len(key) > 128 {
		return &AgentValidationError{Message: "Refresh the page and try again."}
	}
	parsed, err := url.Parse(input.URL)
	if err != nil || len(input.URL) > 2048 || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return &AgentValidationError{Message: "Choose a page from your latest scan."}
	}
	return nil
}

// contentHash keys the cache on what the models would read; the mode is a
// separate column of the cache lookup.
func contentHash(page fetchedPage) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{promptVersion, AnalystModel, WriterModel, page.Title, page.MetaDescription, strings.Join(page.H1s, "\n"), page.Text}, "\x00")))
	return hex.EncodeToString(sum[:])
}

// promptTokens counts every three bytes of the prompt as a token: real text
// runs about four characters a token in English, and Hindi's three-byte
// characters are not fewer than a token each.
func promptTokens(req openrouter.Request) int {
	bytes := 0
	for _, message := range req.Messages {
		bytes += len(message.Content)
	}
	return (bytes + 2) / 3
}

// requestCeiling is the most req can cost at price: its whole prompt and its
// full output allowance.
func requestCeiling(req openrouter.Request, price openrouter.MaxPrice) float64 {
	return (float64(promptTokens(req))*price.Prompt + float64(req.MaxTokens)*price.Completion) / 1_000_000
}

// callCost is what a call cost: OpenRouter's reported cost or, when it sent
// none (a broken stream), the capped price of the tokens it reported or of
// the prompt and the text that arrived, so a missing field never makes a
// call free.
func callCost(result openrouter.Result, req openrouter.Request, price openrouter.MaxPrice) float64 {
	if result.Usage.CostUSD > 0 {
		return result.Usage.CostUSD
	}
	outputTokens := max(result.Usage.CompletionTokens, (len(result.Text)+2)/3)
	inputTokens := result.Usage.PromptTokens
	if inputTokens <= 0 {
		inputTokens = promptTokens(req)
	}
	return (float64(inputTokens)*price.Prompt + float64(outputTokens)*price.Completion) / 1_000_000
}

func creditsToUSD(credits int64) float64 {
	return float64(credits) / (billing.CreditMarkupMultiplier * billing.CreditsPerUSD)
}

// truncateBytes keeps at most limit bytes, ending on a whole character.
func truncateBytes(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	cut := value[:limit]
	for !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

// providerPreferences caps the price and keeps customer pages out of
// providers that train on requests.
func providerPreferences(price openrouter.MaxPrice) *openrouter.ProviderPreferences {
	return &openrouter.ProviderPreferences{MaxPrice: &price, DataCollection: "deny", RequireParameters: true}
}

// Recommendation check IDs must be ones the page knows.
func knownCheck(id string) bool {
	return slices.ContainsFunc(checks, func(c check) bool { return c.id == id })
}

// marshalResult and unmarshalResult store a result in its run row.
func marshalResult(result AgentResult) (string, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode cite fix agent result: %w", err)
	}
	return string(encoded), nil
}
