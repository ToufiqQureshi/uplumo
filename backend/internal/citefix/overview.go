package citefix

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

const (
	googleAIOPath      = "/v3/serp/google/organic/task_post"
	googleAIOGet       = "/v3/serp/google/organic/task_get/advanced/"
	googleAIOBatchSize = 10
	googleAIOCacheTTL  = 7 * 24 * time.Hour
)

var (
	// ErrGoogleAIOUnavailable means the DataForSEO-backed check is not wired.
	ErrGoogleAIOUnavailable = errors.New("google AI Overview checks are unavailable")
	// ErrGoogleAIOPlanRequired means the organization does not have the required plan.
	ErrGoogleAIOPlanRequired = errors.New("google AI Overview checks require an AI Visibility plan")
	// ErrGoogleAIOBusy means another refresh currently holds the project lock.
	ErrGoogleAIOBusy = errors.New("a Google AI Overview check is already running for this project")
	// ErrGoogleAIONotConfigured means no tracked questions are available to check.
	ErrGoogleAIONotConfigured = errors.New("save tracked questions in AI Visibility first")
)

// GoogleAIOPlans gates the explicit paid refresh with the AI Visibility plan.
type GoogleAIOPlans interface {
	HasPaidPlan(ctx context.Context, organizationID string) (bool, error)
}

// GoogleAIOverviewRunner is the HTTP handler's paid submit/free collect
// boundary. Report reads only its saved state and never calls this interface.
type GoogleAIOverviewRunner interface {
	Refresh(ctx context.Context, organizationID, projectID, idempotencyKey string) (GoogleAIOverviewState, error)
	Collect(ctx context.Context, organizationID, projectID string) (GoogleAIOverviewState, error)
}

// GoogleAIOverviewCitation is copied from the shared AI Visibility SERP
// parser. Every provider reference field is retained for the report.
type GoogleAIOverviewCitation = aisearch.Citation

// GoogleAIOverviewAnswer is one tracked question's current saved SERP result.
type GoogleAIOverviewAnswer struct {
	Prompt        string                     `json:"prompt"`
	Topic         string                     `json:"topic"`
	Status        string                     `json:"status"`
	RequestedAt   *time.Time                 `json:"requestedAt"`
	CachedUntil   *time.Time                 `json:"cachedUntil"`
	Answer        string                     `json:"answer"`
	Citations     []GoogleAIOverviewCitation `json:"citations"`
	CustomerCited bool                       `json:"customerCited"`
}

// GoogleAIOverviewState says what can be checked, the approved price cap, and
// each paid answer/reference saved for the project. A report read is free.
type GoogleAIOverviewState struct {
	Available          bool                     `json:"available"`
	ProviderConfigured bool                     `json:"providerConfigured"`
	Domain             string                   `json:"domain"`
	Questions          []string                 `json:"questions"`
	MaxQuestions       int                      `json:"maxQuestions"`
	MaxCredits         int64                    `json:"maxCredits"`
	Results            []GoogleAIOverviewAnswer `json:"results"`
}

// GoogleAIOverview submits and collects only Google SERP AI Overview tasks.
// It reuses DataForSEO's shared cost-recording client and AI Visibility's
// tracked question configuration.
type GoogleAIOverview struct {
	DB     *pgxpool.Pool
	Client *dataforseo.Client
	Plans  GoogleAIOPlans
	Now    func() time.Time
	NewID  func() string
}

type googleAIOPrompt struct {
	Text, Topic string
}

type googleAIOConfig struct {
	Domain       string                   `json:"domain"`
	Prompts      []aisearch.TrackedPrompt `json:"prompts"`
	LocationCode int                      `json:"locationCode"`
	LanguageCode string                   `json:"languageCode"`
}

type googleAIOPostResponse struct {
	StatusCode int                 `json:"status_code"`
	Tasks      []googleAIOPostTask `json:"tasks"`
}

type googleAIOPostTask struct {
	ID         string `json:"id"`
	StatusCode int    `json:"status_code"`
	Data       struct {
		Keyword string `json:"keyword"`
	} `json:"data"`
}

func (s GoogleAIOverview) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s GoogleAIOverview) newID() string {
	if s.NewID != nil {
		return s.NewID()
	}
	return "cfo_" + rand.Text()
}

func (s Store) googleAIOverview(ctx context.Context, organizationID, projectID string) (*GoogleAIOverviewState, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	state, _, _, err := loadGoogleAIOverview(ctx, s.DB, organizationID, projectID, now())
	if err != nil {
		return nil, err
	}
	state.ProviderConfigured = s.GoogleAIOConfigured
	state.Available = state.Available && state.ProviderConfigured
	return &state, nil
}

func loadGoogleAIOverview(ctx context.Context, db *pgxpool.Pool, organizationID, projectID string, now time.Time) (GoogleAIOverviewState, googleAIOConfig, []googleAIOPrompt, error) {
	state := GoogleAIOverviewState{Questions: []string{}, Results: []GoogleAIOverviewAnswer{}}
	var raw []byte
	err := db.QueryRow(ctx, `SELECT t.configuration FROM go_ai_trackers t JOIN projects p ON p.id=t.project_id
		WHERE t.organization_id=$1 AND t.project_id=$2 AND p.archived_at IS NULL`, organizationID, projectID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, googleAIOConfig{}, nil, nil
	}
	if err != nil {
		return state, googleAIOConfig{}, nil, fmt.Errorf("load tracked questions for Google AI Overview: %w", err)
	}
	var config googleAIOConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return state, config, nil, fmt.Errorf("decode tracked questions for Google AI Overview: %w", err)
	}
	config.Domain = strings.TrimSpace(config.Domain)
	prompts := make([]googleAIOPrompt, 0, googleAIOBatchSize)
	seenPrompts := map[string]bool{}
	for _, prompt := range config.Prompts {
		text := strings.Join(strings.Fields(prompt.Text), " ")
		key := strings.ToLower(text)
		if prompt.Paused || len([]rune(text)) < 3 || len([]rune(text)) > 500 || seenPrompts[key] {
			continue
		}
		seenPrompts[key] = true
		prompts = append(prompts, googleAIOPrompt{Text: text, Topic: prompt.Topic})
		if len(prompts) == googleAIOBatchSize {
			break
		}
	}
	if len(prompts) == 0 {
		return state, config, prompts, nil
	}
	state.Available, state.Domain = true, config.Domain
	state.MaxQuestions = len(prompts)
	state.MaxCredits = billing.CreditsForUSD(dataforseo.GoogleAIOverviewBatchCostUSD(len(prompts), 10))
	state.Questions = make([]string, len(prompts))
	hashes := make([]string, len(prompts))
	for i, prompt := range prompts {
		state.Questions[i] = prompt.Text
		hashes[i] = googleAIOPromptHash(prompt.Text, config)
	}
	rows, err := db.Query(ctx, `SELECT prompt_hash,prompt,status,requested_at,answer,citations_json
		FROM go_citefix_ai_overviews WHERE organization_id=$1 AND project_id=$2 AND prompt_hash=ANY($3)`, organizationID, projectID, hashes)
	if err != nil {
		return state, config, prompts, fmt.Errorf("read saved Google AI Overview results: %w", err)
	}
	byHash := map[string]GoogleAIOverviewAnswer{}
	for rows.Next() {
		var hash, prompt, status string
		var requested time.Time
		var answer string
		var rawCitations []byte
		if err := rows.Scan(&hash, &prompt, &status, &requested, &answer, &rawCitations); err != nil {
			rows.Close()
			return state, config, prompts, fmt.Errorf("scan saved Google AI Overview result: %w", err)
		}
		var citations []GoogleAIOverviewCitation
		if err := json.Unmarshal(rawCitations, &citations); err != nil {
			rows.Close()
			return state, config, prompts, fmt.Errorf("decode saved Google AI Overview references: %w", err)
		}
		if citations == nil {
			citations = []GoogleAIOverviewCitation{}
		}
		cachedUntil := requested.Add(googleAIOCacheTTL)
		byHash[hash] = GoogleAIOverviewAnswer{Prompt: prompt, Status: status, RequestedAt: &requested,
			CachedUntil: &cachedUntil, Answer: answer, Citations: citations, CustomerCited: citationsContainDomain(citations, config.Domain)}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return state, config, prompts, fmt.Errorf("read saved Google AI Overview results: %w", err)
	}
	rows.Close()
	for i, prompt := range prompts {
		result, ok := byHash[hashes[i]]
		if !ok {
			result = GoogleAIOverviewAnswer{Prompt: prompt.Text, Topic: prompt.Topic, Status: "not_checked", Citations: []GoogleAIOverviewCitation{}}
		}
		result.Topic = prompt.Topic
		if result.RequestedAt != nil && !now.Before(result.RequestedAt.Add(googleAIOCacheTTL)) && result.Status != "not_checked" {
			result.Status = "stale"
		}
		state.Results = append(state.Results, result)
	}
	return state, config, prompts, nil
}

func googleAIOPromptHash(prompt string, config googleAIOConfig) string {
	encoded, _ := json.Marshal(struct {
		Prompt, Domain string
		Location       int
		Language       string
	}{
		strings.ToLower(strings.Join(strings.Fields(prompt), " ")), strings.ToLower(config.Domain), config.LocationCode, config.LanguageCode,
	})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func googleAIOKeywords(config googleAIOConfig, prompts []googleAIOPrompt) ([]byte, error) {
	tasks := make([]map[string]any, len(prompts))
	for i, prompt := range prompts {
		keyword := strings.NewReplacer("%", "%25", "+", "%2B").Replace(prompt.Text)
		tasks[i] = map[string]any{"keyword": keyword, "location_code": config.LocationCode,
			"language_code": config.LanguageCode, "depth": 10, "load_async_ai_overview": true}
	}
	return json.Marshal(tasks)
}

// Refresh buys at most ten stale/missing questions. A database advisory lock
// serializes same-project clicks, and the seven-day cache makes a second click
// or concurrent request a zero-provider-call read.
func (s GoogleAIOverview) Refresh(ctx context.Context, organizationID, projectID, idempotencyKey string) (result GoogleAIOverviewState, returnErr error) {
	if s.DB == nil || s.Client == nil {
		return GoogleAIOverviewState{}, ErrGoogleAIOUnavailable
	}
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 128 {
		return GoogleAIOverviewState{}, errors.New("a valid request key is required")
	}
	if s.Plans == nil {
		return GoogleAIOverviewState{}, ErrGoogleAIOPlanRequired
	}
	allowed, err := s.Plans.HasPaidPlan(ctx, organizationID)
	if err != nil {
		return GoogleAIOverviewState{}, fmt.Errorf("check Google AI Overview plan: %w", err)
	}
	if !allowed {
		return GoogleAIOverviewState{}, ErrGoogleAIOPlanRequired
	}
	conn, err := s.DB.Acquire(ctx)
	if err != nil {
		return GoogleAIOverviewState{}, fmt.Errorf("lock Google AI Overview refresh: %w", err)
	}
	defer conn.Release()
	lockKey := "citefix-google-aio:" + organizationID + ":" + projectID
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, lockKey); err != nil {
		return GoogleAIOverviewState{}, err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		if _, unlockErr := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, lockKey); unlockErr != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("unlock Google AI Overview refresh: %w", unlockErr))
		}
	}()
	now := s.now()
	state, config, prompts, err := loadGoogleAIOverview(ctx, s.DB, organizationID, projectID, now)
	if err != nil {
		return state, err
	}
	state.ProviderConfigured = true
	if len(prompts) == 0 {
		return state, ErrGoogleAIONotConfigured
	}
	hashes := make([]string, len(prompts))
	for i, prompt := range prompts {
		hashes[i] = googleAIOPromptHash(prompt.Text, config)
	}
	rows, err := s.DB.Query(ctx, `SELECT prompt_hash,requested_at FROM go_citefix_ai_overviews
		WHERE organization_id=$1 AND project_id=$2 AND prompt_hash=ANY($3)`, organizationID, projectID, hashes)
	if err != nil {
		return state, fmt.Errorf("check Google AI Overview cache: %w", err)
	}
	fresh := map[string]bool{}
	for rows.Next() {
		var hash string
		var requested time.Time
		if err := rows.Scan(&hash, &requested); err != nil {
			rows.Close()
			return state, err
		}
		fresh[hash] = now.Before(requested.Add(googleAIOCacheTTL))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return state, err
	}
	rows.Close()
	stale := []googleAIOPrompt{}
	for i, prompt := range prompts {
		if !fresh[hashes[i]] {
			stale = append(stale, prompt)
		}
	}
	if len(stale) == 0 {
		return state, nil
	}
	if len(stale) > googleAIOBatchSize {
		stale = stale[:googleAIOBatchSize]
	}
	for _, prompt := range stale {
		hash := googleAIOPromptHash(prompt.Text, config)
		_, err := s.DB.Exec(ctx, `INSERT INTO go_citefix_ai_overviews(id,organization_id,project_id,prompt_hash,prompt,status,requested_at,citations_json)
			SELECT $1,$2,p.id,$4,$5,'unknown',$6,'[]'::jsonb FROM projects p WHERE p.id=$3 AND p.organization_id=$2 AND p.archived_at IS NULL
			ON CONFLICT(project_id,prompt_hash) DO UPDATE SET prompt=EXCLUDED.prompt,status='unknown',task_id='',requested_at=EXCLUDED.requested_at,
				completed_at=NULL,answer='',citations_json='[]'::jsonb`, s.newID(), organizationID, projectID, hash, prompt.Text, now)
		if err != nil {
			return state, fmt.Errorf("reserve Google AI Overview evidence row: %w", err)
		}
	}
	body, err := googleAIOKeywords(config, stale)
	if err != nil {
		return state, err
	}
	ctx = spend.WithIdempotencyKey(ctx, idempotencyKey)
	ceiling := dataforseo.GoogleAIOverviewBatchCostUSD(len(stale), 10)
	err = s.Client.DoBudgeted(ctx, organizationID, spend.CiteFix, ceiling, func(request dataforseo.DoBudgetedRequest) error {
		raw, callErr := request(http.MethodPost, googleAIOPath, body, false)
		if callErr != nil {
			return callErr
		}
		var response googleAIOPostResponse
		if err := json.Unmarshal(raw, &response); err != nil {
			return errors.Join(dataforseo.ErrNothingDelivered, err)
		}
		if response.StatusCode != 20000 {
			return errors.Join(dataforseo.ErrNothingDelivered, errors.New("DataForSEO rejected the Google AI Overview request"))
		}
		byKeyword := map[string]googleAIOPostTask{}
		for _, task := range response.Tasks {
			if strings.TrimSpace(task.Data.Keyword) != "" {
				byKeyword[strings.ToLower(strings.Join(strings.Fields(task.Data.Keyword), " "))] = task
			}
		}
		for _, prompt := range stale {
			hash := googleAIOPromptHash(prompt.Text, config)
			task, ok := byKeyword[strings.ToLower(strings.Join(strings.Fields(strings.NewReplacer("%", "%25", "+", "%2B").Replace(prompt.Text)), " "))]
			status, taskID := "failed", ""
			if ok && task.ID != "" && (task.StatusCode == 20100 || task.StatusCode == 20000) {
				status, taskID = "pending", task.ID
			}
			if _, err := s.DB.Exec(ctx, `UPDATE go_citefix_ai_overviews SET task_id=$4,status=$5,completed_at=CASE WHEN $5='failed' THEN now() ELSE NULL END
				WHERE organization_id=$1 AND project_id=$2 AND prompt_hash=$3`, organizationID, projectID, hash, taskID, status); err != nil {
				return fmt.Errorf("save Google AI Overview task: %w", err)
			}
		}
		return nil
	})
	if spend.Refused(err) {
		var cleanupErr error
		for _, prompt := range stale {
			if _, err := s.DB.Exec(context.WithoutCancel(ctx), `DELETE FROM go_citefix_ai_overviews WHERE organization_id=$1 AND project_id=$2 AND prompt_hash=$3 AND status='unknown'`, organizationID, projectID, googleAIOPromptHash(prompt.Text, config)); err != nil {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove unsubmitted Google AI Overview reservation: %w", err))
			}
		}
		return GoogleAIOverviewState{}, errors.Join(err, cleanupErr)
	}
	if err != nil {
		if errors.Is(err, dataforseo.ErrNothingDelivered) {
			var cleanupErr error
			for _, prompt := range stale {
				if _, updateErr := s.DB.Exec(context.WithoutCancel(ctx), `UPDATE go_citefix_ai_overviews SET status='failed',completed_at=now() WHERE organization_id=$1 AND project_id=$2 AND prompt_hash=$3 AND status='unknown'`, organizationID, projectID, googleAIOPromptHash(prompt.Text, config)); updateErr != nil {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("save failed Google AI Overview status: %w", updateErr))
				}
			}
			err = errors.Join(err, cleanupErr)
		}
		// Unknown network outcomes remain cached with no task id; they are never
		// automatically submitted again during the seven-day evidence window.
		return GoogleAIOverviewState{}, err
	}
	return s.Collect(ctx, organizationID, projectID)
}

// Collect saves completed results for a queued Google AI Overview check.
func (s GoogleAIOverview) Collect(ctx context.Context, organizationID, projectID string) (GoogleAIOverviewState, error) {
	if s.DB == nil {
		return GoogleAIOverviewState{}, ErrGoogleAIOUnavailable
	}
	if s.Client != nil {
		rows, err := s.DB.Query(ctx, `SELECT prompt_hash,task_id FROM go_citefix_ai_overviews WHERE organization_id=$1 AND project_id=$2
			AND status='pending' AND task_id<>'' AND requested_at>$3 ORDER BY requested_at LIMIT $4`, organizationID, projectID, s.now().Add(-googleAIOCacheTTL), googleAIOBatchSize)
		if err != nil {
			return GoogleAIOverviewState{}, fmt.Errorf("list Google AI Overview tasks: %w", err)
		}
		tasks := []struct{ hash, id string }{}
		for rows.Next() {
			var task struct{ hash, id string }
			if err := rows.Scan(&task.hash, &task.id); err != nil {
				rows.Close()
				return GoogleAIOverviewState{}, err
			}
			tasks = append(tasks, task)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return GoogleAIOverviewState{}, err
		}
		rows.Close()
		for _, task := range tasks {
			path := googleAIOGet + url.PathEscape(task.id)
			raw, err := s.Client.DoUnmetered(ctx, http.MethodGet, path, nil, false)
			if err != nil {
				continue
			}
			var response struct {
				StatusCode int `json:"status_code"`
				Tasks      []struct {
					StatusCode int               `json:"status_code"`
					Result     []json.RawMessage `json:"result"`
				} `json:"tasks"`
			}
			if err := json.Unmarshal(raw, &response); err != nil || response.StatusCode != 20000 || len(response.Tasks) != 1 {
				continue
			}
			taskStatus := response.Tasks[0].StatusCode
			if taskStatus == 20100 || taskStatus == 40601 || taskStatus == 40602 {
				continue
			}
			status, answer, citations := "failed", "", []GoogleAIOverviewCitation{}
			if taskStatus == 20000 || taskStatus == 40102 {
				status = "completed"
				if len(response.Tasks[0].Result) > 0 {
					answer, citations, err = aisearch.ParseGoogleAIOverview(response.Tasks[0].Result[0])
					if err != nil {
						status = "failed"
						answer = ""
						citations = []GoogleAIOverviewCitation{}
					}
				}
			}
			encoded, err := json.Marshal(citations)
			if err != nil {
				return GoogleAIOverviewState{}, fmt.Errorf("encode Google AI Overview citations: %w", err)
			}
			if _, err := s.DB.Exec(ctx, `UPDATE go_citefix_ai_overviews SET status=$3,completed_at=now(),answer=$4,citations_json=$5
				WHERE organization_id=$1 AND project_id=$2 AND prompt_hash=$6 AND status='pending'`, organizationID, projectID, status, answer, encoded, task.hash); err != nil {
				return GoogleAIOverviewState{}, fmt.Errorf("save Google AI Overview result: %w", err)
			}
		}
	}
	state, _, _, err := loadGoogleAIOverview(ctx, s.DB, organizationID, projectID, s.now())
	state.ProviderConfigured = s.Client != nil
	state.Available = state.Available && state.ProviderConfigured
	return state, err
}

func citationsContainDomain(citations []GoogleAIOverviewCitation, rawDomain string) bool {
	target := normalizeCitationDomain(rawDomain)
	if target == "" {
		return false
	}
	for _, citation := range citations {
		if citation.Domain == nil {
			continue
		}
		domain := normalizeCitationDomain(*citation.Domain)
		if domain == target || strings.HasSuffix(domain, "."+target) {
			return true
		}
	}
	return false
}

func normalizeCitationDomain(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return ""
		}
		value = parsed.Hostname()
	}
	value = strings.TrimPrefix(value, "www.")
	return strings.TrimSuffix(value, ".")
}

// keep these compile-time assertions close to the implementation.
var _ GoogleAIOverviewRunner = GoogleAIOverview{}
