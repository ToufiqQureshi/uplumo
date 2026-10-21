package aisearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/jobs"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/market"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

func trackingEndpoint(engine string) string {
	if engine == "google_ai_overview" {
		return "/v3/serp/google/organic"
	}
	return "/v3/ai_optimization/" + engine + "/llm_scraper"
}

// Execute resumes free task collection after a crash. An uncertain paid POST is
// never replayed: submitting/pending observations on a retry become failed.
func (s *Tracking) Execute(ctx context.Context, job jobs.Job) error {
	var payload struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return err
	}
	var org, project, status string
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT organization_id,project_id,status,configuration FROM go_ai_tracking_runs WHERE id=$1`, payload.RunID).Scan(&org, &project, &status, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if status != "queued" && status != "running" {
		return nil
	}
	var c TrackingConfig
	if err = json.Unmarshal(raw, &c); err != nil {
		return err
	}
	if err = s.allowed(ctx, org); err != nil {
		if errors.Is(err, errTrackingPlan) {
			return s.failRun(ctx, payload.RunID, "plan_required")
		}
		return err
	}
	var active bool
	if err = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1 AND organization_id=$2 AND archived_at IS NULL)`, project, org).Scan(&active); err != nil {
		return err
	}
	if !active {
		return s.failRun(ctx, payload.RunID, "project_archived")
	}
	if s.Search == nil {
		return s.failRun(ctx, payload.RunID, "provider_unavailable")
	}
	tag, err := s.DB.Exec(ctx, `UPDATE go_ai_tracking_runs SET status='running' WHERE id=$1 AND status IN ('queued','running')`, payload.RunID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if job.Attempts > 1 {
		if _, err = s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='failed',error_code='interrupted' WHERE run_id=$1 AND status IN ('pending','submitting')`, payload.RunID); err != nil {
			return err
		}
	} else {
		answers, e := s.Answers(ctx, org, project, payload.RunID)
		if e != nil {
			return e
		}
		credits := c.estimate().Credits
		budgetCtx := spend.WithIdempotencyKey(ctx, "ai-tracking:"+payload.RunID)
		err = s.Search.provider.client.DoBudgeted(budgetCtx, org, spend.AIVisibility, float64(credits)/1280, func(request dataforseo.DoBudgetedRequest) error {
			return s.submit(ctxWithBudget(budgetCtx, request), org, payload.RunID, c, answers, request)
		})
		if err != nil {
			s.Search.logger.ErrorContext(ctx, "AI tracking collection failed", "run", payload.RunID, "err", err)
			code := "provider_error"
			if spend.Refused(err) {
				code = "credit_limit"
			}
			if e := s.failUnsubmitted(ctx, payload.RunID, code); e != nil {
				return e
			}
		}
	}
	for {
		answers, e := s.Answers(ctx, org, project, payload.RunID)
		if e != nil {
			return e
		}
		pending := false
		for _, a := range answers {
			if a.Status != "submitted" {
				continue
			}
			done, answer, citations, recommended, e := s.collect(ctx, a)
			if e != nil {
				pending = true
				s.Search.logger.WarnContext(ctx, "AI tracking task read failed", "run", payload.RunID, "err", e)
				continue
			}
			if !done {
				pending = true
				continue
			}
			if e = s.finishAnswer(ctx, a.ID, c, answer, citations, recommended, ""); e != nil {
				return e
			}
		}
		if !pending {
			break
		}
		timer := time.NewTimer(15 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				end, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				return s.failRun(end, payload.RunID, "collection_timeout")
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	_, err = s.DB.Exec(ctx, `UPDATE go_ai_tracking_runs r SET status=CASE
 WHEN NOT EXISTS(SELECT 1 FROM go_ai_tracking_answers WHERE run_id=r.id AND status='completed') THEN 'failed'
 WHEN EXISTS(SELECT 1 FROM go_ai_tracking_answers WHERE run_id=r.id AND status='failed') THEN 'partial' ELSE 'completed' END,finished_at=now()
 WHERE id=$1`, payload.RunID)
	return err
}
func (s *Tracking) failUnsubmitted(ctx context.Context, id, code string) error {
	_, err := s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='failed',error_code=$2 WHERE run_id=$1 AND status IN ('pending','submitting')`, id, code)
	return err
}
func (s *Tracking) failRun(ctx context.Context, id, code string) error {
	_, err := s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='failed',error_code=$2 WHERE run_id=$1 AND status NOT IN ('completed','failed')`, id, code)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, `UPDATE go_ai_tracking_runs SET status=CASE WHEN EXISTS(SELECT 1 FROM go_ai_tracking_answers WHERE run_id=$1 AND status='completed') THEN 'partial' ELSE 'failed' END,error_code=$2,finished_at=now() WHERE id=$1`, id, code)
	return err
}
func (s *Tracking) submit(ctx context.Context, org, run string, c TrackingConfig, answers []TrackingAnswer, request dataforseo.DoBudgetedRequest) error {
	for _, engine := range c.Engines {
		group := []TrackingAnswer{}
		for _, a := range answers {
			if a.Engine == engine && a.Status == "pending" {
				group = append(group, a)
			}
		}
		if len(group) == 0 {
			continue
		}
		ids := make([]string, len(group))
		for i, a := range group {
			ids[i] = a.ID
		}
		tag, err := s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='submitting' WHERE run_id=$1 AND id=ANY($2) AND status='pending'`, run, ids)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != int64(len(group)) {
			return errors.New("AI answer was already claimed")
		}
		if engine == "claude" || engine == "perplexity" {
			model := s.Search.provider.models.latest(ctx, org, engine)
			for _, a := range group {
				raw, e := s.Search.provider.llmResponse(ctx, responseRequest{organizationID: org, model: engine, modelName: model, prompt: a.Prompt, webSearch: true, countryCode: strings.ToUpper(market.ISOCode(c.LocationCode)), maxTokens: promptMaxOutputTokens})
				if e != nil {
					return e
				}
				if e = s.finishAnswer(ctx, a.ID, c, extractText(raw), extractCitations(raw), nil, ""); e != nil {
					return e
				}
			}
			continue
		}
		tasks := make([]map[string]any, len(group))
		lang := c.LanguageCode
		if engine != "google_ai_overview" {
			switch lang {
			case "nb":
				lang = "no"
			case "tl":
				lang = "fil"
			case "pt":
				if engine == "gemini" {
					lang = "pt-BR"
				}
			}
		}
		for i, a := range group {
			tasks[i] = map[string]any{"keyword": strings.NewReplacer("%", "%25", "+", "%2B").Replace(a.Prompt), "location_code": c.LocationCode, "language_code": lang, "tag": a.ID}
			if engine == "google_ai_overview" {
				tasks[i]["depth"] = 10
				tasks[i]["load_async_ai_overview"] = true
			}
		}
		body, err := json.Marshal(tasks)
		if err != nil {
			return err
		}
		response, err := request(http.MethodPost, trackingEndpoint(engine)+"/task_post", body, false)
		if err != nil {
			return err
		}
		var envelope struct {
			StatusCode int `json:"status_code"`
			Tasks      []struct {
				ID         string `json:"id"`
				StatusCode int    `json:"status_code"`
				Data       struct {
					Tag string `json:"tag"`
				} `json:"data"`
			} `json:"tasks"`
		}
		if err = json.Unmarshal(response, &envelope); err != nil {
			return err
		}
		if envelope.StatusCode != 20000 {
			return errors.New("AI queue submission failed")
		}
		for _, task := range envelope.Tasks {
			if task.StatusCode != 20100 || task.ID == "" || !slices.Contains(ids, task.Data.Tag) {
				continue
			}
			if _, err = s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET task_id=$2,status='submitted' WHERE id=$1 AND run_id=$3 AND status='submitting'`, task.Data.Tag, task.ID, run); err != nil {
				return err
			}
		}
		if _, err = s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='failed',error_code='provider_rejected' WHERE id=ANY($1) AND status='submitting'`, ids); err != nil {
			return err
		}
	}
	return nil
}
func (s *Tracking) collect(ctx context.Context, a TrackingAnswer) (bool, string, []Citation, []RecommendedBrand, error) {
	raw, err := s.Search.provider.client.DoUnmetered(ctx, http.MethodGet, trackingEndpoint(a.Engine)+"/task_get/advanced/"+url.PathEscape(a.TaskID), nil, true)
	if err != nil {
		return false, "", nil, nil, err
	}
	var env struct {
		StatusCode int `json:"status_code"`
		Tasks      []struct {
			StatusCode int               `json:"status_code"`
			Result     []json.RawMessage `json:"result"`
		} `json:"tasks"`
	}
	if err = json.Unmarshal(raw, &env); err != nil {
		return false, "", nil, nil, err
	}
	if env.StatusCode != 20000 || len(env.Tasks) != 1 {
		return false, "", nil, nil, errors.New("invalid task response")
	}
	code := env.Tasks[0].StatusCode
	if slices.Contains([]int{20100, 40601, 40602}, code) {
		return false, "", nil, nil, nil
	}
	if code != 20000 && code != 40102 {
		_, err = s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='failed',error_code='provider_failed' WHERE id=$1 AND status='submitted'`, a.ID)
		return false, "", nil, nil, err
	}
	if len(env.Tasks[0].Result) == 0 {
		return true, "", []Citation{}, nil, nil
	}
	answer, citations, recommended, err := parseTrackingAnswer(env.Tasks[0].Result[0], a.Engine)
	return err == nil, answer, citations, recommended, err
}

type trackingSource struct {
	URL             string  `json:"url"`
	Title           *string `json:"title"`
	SourceName      *string `json:"source_name"`
	PublicationDate *string `json:"publication_date"`
}

type trackingBrandEntity struct {
	Title    string `json:"title"`
	Category string `json:"category"`
}

const maxRecommendedBrands = 20

func parseTrackingAnswer(raw []byte, engine string) (string, []Citation, []RecommendedBrand, error) {
	var r struct {
		Markdown string           `json:"markdown"`
		Sources  []trackingSource `json:"sources"`
		Items    []struct {
			Type          string                `json:"type"`
			Markdown      string                `json:"markdown"`
			References    []trackingSource      `json:"references"`
			BrandEntities []trackingBrandEntity `json:"brand_entities"`
			Items         []struct {
				References []trackingSource `json:"references"`
			} `json:"items"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", nil, nil, err
	}
	text, sources := r.Markdown, r.Sources
	var entities []trackingBrandEntity
	for _, i := range r.Items {
		// An AI Overview result also lists organic results; only the
		// overview's own brands were named by the AI.
		if engine != "google_ai_overview" || i.Type == "ai_overview" {
			entities = append(entities, i.BrandEntities...)
		}
	}
	if engine == "google_ai_overview" {
		text = ""
		sources = nil
		for _, i := range r.Items {
			if i.Type == "ai_overview" {
				text = i.Markdown
				sources = append(sources, i.References...)
				for _, child := range i.Items {
					sources = append(sources, child.References...)
				}
				break
			}
		}
	}
	citations := []Citation{}
	seen := map[string]bool{}
	for _, source := range sources {
		safe, ok := safeHTTPURL(source.URL)
		if !ok || seen[safe] {
			continue
		}
		seen[safe] = true
		citations = append(citations, Citation{URL: safe, Domain: safeHostname(safe), Title: source.Title,
			Source: truncatedPtr(source.SourceName, maxTitleLength), PublishedAt: truncatedPtr(source.PublicationDate, 40)})
		if len(citations) >= maxCitations {
			break
		}
	}
	return truncate(text, 64000), citations, recommendedBrands(entities), nil
}

// recommendedBrands keeps the distinct brands the AI named, first mention first.
func recommendedBrands(entities []trackingBrandEntity) []RecommendedBrand {
	brands := []RecommendedBrand{}
	seen := map[string]bool{}
	for _, e := range entities {
		name := strings.TrimSpace(e.Title)
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		brands = append(brands, RecommendedBrand{Name: truncate(name, maxBrandEntityLength), Category: truncate(e.Category, 60), Position: len(brands) + 1})
		if len(brands) == maxRecommendedBrands {
			break
		}
	}
	return brands
}

var trackingURLs = regexp.MustCompile(`https?://[^\s)\]>]+|www\.[^\s)\]>]+`)

func (s *Tracking) finishAnswer(ctx context.Context, id string, c TrackingConfig, text string, citations []Citation, recommended []RecommendedBrand, code string) error {
	brands := []TrackedBrand{{c.Brand, c.Domain}}
	brands = append(brands, c.Competitors...)
	observed := make([]storedBrand, 0, len(brands)+len(recommended))
	prose := trackingProse(text)
	for _, b := range brands {
		cited := false
		for _, citation := range citations {
			if citation.Domain != nil && (*citation.Domain == b.Domain || strings.HasSuffix(*citation.Domain, "."+b.Domain)) {
				cited = true
			}
		}
		observed = append(observed, storedBrand{BrandObservation: BrandObservation{b.Name, b.Domain, mentionsBrand(prose, b.Name), cited}})
	}
	// Tracked brands stay first: rate queries read brands->0 as the target.
	for _, r := range recommended {
		observed = append(observed, storedBrand{BrandObservation: BrandObservation{Name: r.Name}, Recommended: true, Category: r.Category, Position: r.Position})
	}
	if citations == nil {
		citations = []Citation{}
	}
	raw, e := json.Marshal(citations)
	if e != nil {
		return e
	}
	br, e := json.Marshal(observed)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec(ctx, `UPDATE go_ai_tracking_answers SET status='completed',answer=$2,citations=$3,brands=$4,collected_at=now(),error_code=$5 WHERE id=$1 AND status IN ('submitted','submitting')`, id, truncate(text, 64000), raw, br, code)
	return e
}

// RunWorkers uses the existing durable queue. Scheduling itself makes no paid call.
func (s *Tracking) RunWorkers(ctx context.Context, logger interface{ Error(string, ...any) }) error {
	queue, err := jobs.New(s.DB)
	if err != nil {
		return err
	}
	worker := jobs.Worker{Queue: queue, QueueName: trackingQueue, Handle: s.Execute, Lease: time.Minute, PollInterval: 2 * time.Second}
	for range 2 {
		go func() {
			for ctx.Err() == nil {
				err := worker.Run(ctx)
				if ctx.Err() != nil {
					return
				}
				logger.Error("AI tracking worker restarting", "err", err)
				timer := time.NewTimer(5 * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}()
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tick, cancel := context.WithTimeout(ctx, time.Minute)
				err := s.Tick(tick)
				cancel()
				if err != nil && ctx.Err() == nil {
					logger.Error("AI tracking schedule failed", "err", fmt.Errorf("tick: %w", err))
				}
			}
		}
	}()
	return nil
}

var trackingLinks = regexp.MustCompile(`\[([^]\n]*)\]\((https?://[^)\s]+)\)`)

func trackingProse(markdown string) string {
	clean := trackingLinks.ReplaceAllStringFunc(markdown, func(link string) string {
		parts := trackingLinks.FindStringSubmatch(link)
		u, e := url.Parse(parts[2])
		if e != nil {
			return ""
		}
		label := strings.TrimSpace(parts[1])
		if strings.EqualFold(strings.TrimPrefix(label, "www."), strings.TrimPrefix(u.Hostname(), "www.")) {
			return ""
		}
		return label
	})
	return trackingURLs.ReplaceAllString(clean, "")
}
