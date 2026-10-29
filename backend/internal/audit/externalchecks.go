package audit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/jobs"
)

// ExternalChecksQueue isolates slow third-party checks from the audit worker.
const ExternalChecksQueue = "site_audit_external"

const (
	maxExternalLinks    = 200
	maxExternalImages   = 30
	maxProbeSources     = 3
	maxProbeURLLength   = 2048
	maxProbeRobotsBytes = 64 * 1024
	largeImageBytes     = 500_000
	probeTimeout        = 5 * time.Second
	probeBudget         = 25 * time.Second
	probeHostInterval   = time.Second
	probeWorkers        = 4
	probeAttempts       = 10
)

type externalProbeSource struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type externalProbe struct {
	Target  string                `json:"target"`
	Kind    string                `json:"kind"`
	Sources []externalProbeSource `json:"sources"`
}
type externalJob struct {
	AuditID        string          `json:"auditId"`
	ProjectID      string          `json:"projectId"`
	OrganizationID string          `json:"organizationId"`
	StartURL       string          `json:"startUrl"`
	Probes         []externalProbe `json:"probes"`
	Limited        bool            `json:"limited"`
	Started        bool            `json:"started"`
}

func safeProbeURL(raw string) bool {
	if len(raw) == 0 || len(raw) > maxProbeURLLength || !isCrawlableURL(raw) {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.User == nil && u.Hostname() != ""
}

// collectExternalProbes remembers only bounded public URL candidates; it makes
// no network requests and never retains credentials embedded in a URL.
func (s *aiSiteSignals) collectExternalProbes(page CrawledPageResult) {
	add := func(kind, raw string) {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.User != nil {
			s.probeLimited = true
			return
		}
		target, ok := normalizeURL(raw, page.URL)
		if !ok || !safeProbeURL(target) || !safeProbeURL(page.URL) {
			s.probeLimited = true
			return
		}
		if s.probeIndex == nil {
			s.probeIndex = map[string]int{}
		}
		key := kind + "\x00" + target
		if index, exists := s.probeIndex[key]; exists {
			item := &s.probes[index]
			for _, source := range item.Sources {
				if source.ID == page.ID {
					return
				}
			}
			if len(item.Sources) < maxProbeSources {
				item.Sources = append(item.Sources, externalProbeSource{ID: page.ID, URL: page.URL})
			} else {
				s.probeLimited = true
			}
			return
		}
		if kind == "link" && s.probeLinks >= maxExternalLinks || kind == "image" && s.probeImages >= maxExternalImages {
			s.probeLimited = true
			return
		}
		if kind == "link" {
			s.probeLinks++
		} else {
			s.probeImages++
		}
		s.probeIndex[key] = len(s.probes)
		s.probes = append(s.probes, externalProbe{Target: target, Kind: kind, Sources: []externalProbeSource{{ID: page.ID, URL: page.URL}}})
	}
	for _, link := range page.Links {
		if !link.IsInternal {
			add("link", link.TargetURL)
		}
	}
	for _, image := range page.Images {
		add("image", image.Src)
	}
}

// ExternalChecks uses its own queue and per-workspace advisory lock: slow
// third-party hosts never occupy the ordinary site-audit worker.
type ExternalChecks struct {
	queue   *jobs.Queue
	db      *pgxpool.Pool
	repo    *Repository
	guard   *Guard
	limiter hostLimiter
	logger  *slog.Logger
}

// NewExternalChecks wires the isolated queue, guarded transport and host limiter.
func NewExternalChecks(queue *jobs.Queue, db *pgxpool.Pool, repo *Repository, guard *Guard, rdb *redis.Client, logger *slog.Logger) *ExternalChecks {
	if guard == nil {
		guard = NewGuard()
	}
	if logger == nil {
		logger = slog.Default()
	}
	var limiter hostLimiter = unavailableHostLimiter{}
	if rdb != nil {
		limiter = redisHostLimiter{client: rdb}
	}
	return &ExternalChecks{queue: queue, db: db, repo: repo, guard: guard, limiter: limiter, logger: logger}
}

func (e *ExternalChecks) schedule(ctx context.Context, job JobPayload, signals *aiSiteSignals) error {
	if e == nil || len(signals.probes) == 0 && !signals.probeLimited {
		return nil
	}
	if len(signals.probes) == 0 {
		return e.repo.InsertIssues(ctx, job.AuditID, []DetectedIssue{{IssueType: IssueExternalChecksIncomplete, PageURL: job.StartURL}})
	}
	payload := externalJob{AuditID: job.AuditID, ProjectID: job.ProjectID, OrganizationID: job.OrganizationID, StartURL: job.StartURL, Probes: signals.probes, Limited: signals.probeLimited}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode external checks: %w", err)
	}
	pending := DetectedIssue{IssueType: IssueExternalChecksPending, PageURL: job.StartURL}
	if err := e.repo.InsertIssues(ctx, job.AuditID, []DetectedIssue{pending}); err != nil {
		return err
	}
	_, err = e.queue.Enqueue(ctx, jobs.EnqueueInput{Queue: ExternalChecksQueue, IdempotencyKey: job.AuditID, Payload: raw, MaxAttempts: probeAttempts, Timeout: 40 * time.Second})
	if err != nil {
		finishErr := e.finish(ctx, payload, nil, true)
		return errors.Join(fmt.Errorf("queue external checks: %w", err), finishErr)
	}
	return nil
}

// Handler is idempotent across worker restarts. A persistent started marker is
// written before network dispatch; an ambiguous restart marks coverage partial
// instead of issuing the same HEAD requests again.
func (e *ExternalChecks) Handler() jobs.Handler {
	return func(ctx context.Context, job jobs.Job) error {
		var input externalJob
		if err := json.Unmarshal(job.Payload, &input); err != nil {
			return fmt.Errorf("decode external check job: %w", err)
		}
		if input.AuditID == "" || input.ProjectID == "" || input.OrganizationID == "" || len(input.Probes) > maxExternalLinks+maxExternalImages {
			return errors.New("invalid external check job")
		}
		if input.Started {
			if err := e.finish(ctx, input, nil, true); err != nil {
				return err
			}
			return e.clearPayload(ctx, job)
		}
		valid, err := e.validateInput(ctx, input)
		if err != nil {
			return err
		}
		if !valid {
			e.logger.WarnContext(ctx, "invalid external check job", "audit_id", input.AuditID)
			if err := e.finish(ctx, input, nil, true); err != nil {
				return err
			}
			return e.clearPayload(ctx, job)
		}
		started, err := e.claim(ctx, input, job)
		if err != nil {
			return err
		}
		if !started {
			if job.Attempts >= job.MaxAttempts {
				if err := e.finish(ctx, input, nil, true); err != nil {
					return err
				}
				return e.clearPayload(ctx, job)
			}
			return errors.New("workspace external checks are busy")
		}
		probeCtx, cancel := context.WithTimeout(ctx, probeBudget)
		issues, incomplete := runExternalProbes(probeCtx, e.guard, input.Probes, e.limiter)
		cancel()
		finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer finishCancel()
		if err := e.finish(finishCtx, input, issues, incomplete || input.Limited); err != nil {
			return err
		}
		// Candidate URLs can contain sensitive query parameters. Remove them
		// from the durable queue payload after the issue rows are saved.
		return e.clearPayload(finishCtx, job)
	}
}

// claim serializes only the short start transaction, then releases the database
// connection before any network wait. A started, leased job reserves its own
// workspace until completion or lease expiry.
func (e *ExternalChecks) claim(ctx context.Context, input externalJob, job jobs.Job) (bool, error) {
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin external check claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var locked, busy bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`, "audit-external:"+input.OrganizationID).Scan(&locked); err != nil {
		return false, fmt.Errorf("lock external check claim: %w", err)
	}
	if !locked {
		return false, nil
	}
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM go_jobs
		WHERE queue=$1 AND state='running' AND lease_until > now() AND id <> $2
		AND payload->>'organizationId'=$3 AND payload->>'started'='true')`, ExternalChecksQueue, job.ID, input.OrganizationID).Scan(&busy); err != nil {
		return false, fmt.Errorf("check workspace external claims: %w", err)
	}
	if busy {
		return false, nil
	}
	tag, err := tx.Exec(ctx, `UPDATE go_jobs SET payload=jsonb_set(payload, '{started}', 'true'::jsonb)
		WHERE id=$1 AND state='running' AND claim_version=$2 AND lease_until > now()`, job.ID, job.ClaimVersion)
	if err != nil {
		return false, fmt.Errorf("claim external checks: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return false, jobs.ErrClaimLost
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit external check claim: %w", err)
	}
	return true, nil
}

func (e *ExternalChecks) clearPayload(ctx context.Context, job jobs.Job) error {
	tag, err := e.db.Exec(ctx, `UPDATE go_jobs SET payload=payload - 'probes' WHERE id=$1 AND state='running' AND claim_version=$2 AND lease_until > now()`, job.ID, job.ClaimVersion)
	if err != nil {
		return fmt.Errorf("clear external check payload: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return jobs.ErrClaimLost
	}
	return nil
}

// Validate ownership and saved page references before dispatching network work.
func (e *ExternalChecks) validateInput(ctx context.Context, input externalJob) (bool, error) {
	var startURL string
	err := e.db.QueryRow(ctx, `SELECT a.start_url FROM go_audits a JOIN projects p ON p.id=a.project_id
		WHERE a.id=$1 AND a.project_id=$2 AND p.organization_id=$3`, input.AuditID, input.ProjectID, input.OrganizationID).Scan(&startURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("authorize external check job: %w", err)
	}
	if startURL != input.StartURL {
		return false, nil
	}
	ids := make([]string, 0, len(input.Probes))
	links, images := 0, 0
	seen := make(map[string]bool, len(input.Probes))
	for _, candidate := range input.Probes {
		if !safeProbeURL(candidate.Target) || len(candidate.Sources) == 0 || len(candidate.Sources) > maxProbeSources || seen[candidate.Kind+"\x00"+candidate.Target] {
			return false, nil
		}
		seen[candidate.Kind+"\x00"+candidate.Target] = true
		switch candidate.Kind {
		case "link":
			links++
		case "image":
			images++
		default:
			return false, nil
		}
		for _, source := range candidate.Sources {
			ids = append(ids, source.ID)
		}
	}
	if links > maxExternalLinks || images > maxExternalImages {
		return false, nil
	}
	rows, err := e.db.Query(ctx, `SELECT id, url FROM go_audit_pages WHERE audit_id=$1 AND id=ANY($2)`, input.AuditID, ids)
	if err != nil {
		return false, fmt.Errorf("load external check pages: %w", err)
	}
	defer rows.Close()
	pages := make(map[string]string, len(ids))
	for rows.Next() {
		var id, pageURL string
		if err := rows.Scan(&id, &pageURL); err != nil {
			return false, fmt.Errorf("read external check page: %w", err)
		}
		pages[id] = pageURL
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("read external check pages: %w", err)
	}
	for _, candidate := range input.Probes {
		for _, source := range candidate.Sources {
			if source.ID == "" || pages[source.ID] != source.URL {
				return false, nil
			}
		}
	}
	return true, nil
}

func (e *ExternalChecks) finish(ctx context.Context, job externalJob, issues []DetectedIssue, incomplete bool) error {
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin external check results: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var found bool
	err = tx.QueryRow(ctx, `SELECT true FROM go_audits a JOIN projects p ON p.id=a.project_id
		WHERE a.id=$1 AND a.project_id=$2 AND p.organization_id=$3 FOR UPDATE OF a`, job.AuditID, job.ProjectID, job.OrganizationID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("authorize external check results: %w", err)
	}
	// No pending row means another attempt already finished. Never replace a
	// successful result with an ambiguous-restart warning.
	var pending bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM go_audit_issues WHERE audit_id=$1 AND issue_type=$2)`, job.AuditID, string(IssueExternalChecksPending)).Scan(&pending); err != nil {
		return fmt.Errorf("find external check state: %w", err)
	}
	if !pending {
		return nil
	}
	if incomplete {
		issues = append(issues, DetectedIssue{IssueType: IssueExternalChecksIncomplete, PageURL: job.StartURL})
	}
	if err := insertIssuesTx(ctx, tx, job.AuditID, issues); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM go_audit_issues WHERE audit_id=$1 AND issue_type=$2`, job.AuditID, string(IssueExternalChecksPending)); err != nil {
		return fmt.Errorf("finish external check state: %w", err)
	}
	return tx.Commit(ctx)
}

type externalHost struct {
	mu          sync.Mutex
	next        time.Time
	robots      *RobotsResult
	unavailable bool
}
type externalProber struct {
	client  *http.Client
	limiter hostLimiter
	mu      sync.Mutex
	hosts   map[string]*externalHost
}

type hostLimiter interface {
	wait(context.Context, string) error
}

type unavailableHostLimiter struct{}

func (unavailableHostLimiter) wait(context.Context, string) error {
	return errors.New("external host rate limiter unavailable")
}

// Redis slots protect the destination host across workers and app instances;
// keys contain only a hash and expire after one second. Failure is unknown
// coverage, not an unthrottled fallback.
type redisHostLimiter struct{ client *redis.Client }

func (l redisHostLimiter) wait(ctx context.Context, origin string) error {
	key := fmt.Sprintf("audit:external-host:%x", sha256.Sum256([]byte(origin)))
	for {
		ok, err := l.client.SetNX(ctx, key, "1", probeHostInterval).Result()
		if err != nil {
			return fmt.Errorf("reserve external host slot: %w", err)
		}
		if ok {
			return nil
		}
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (p *externalProber) wait(ctx context.Context, host *externalHost, origin string) error {
	if err := host.wait(ctx); err != nil {
		return err
	}
	if p.limiter != nil {
		return p.limiter.wait(ctx, origin)
	}
	return nil
}

func (p *externalProber) host(origin string) *externalHost {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hosts[origin] == nil {
		p.hosts[origin] = &externalHost{}
	}
	return p.hosts[origin]
}

func (h *externalHost) wait(ctx context.Context) error {
	if delay := time.Until(h.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	h.next = time.Now().Add(probeHostInterval)
	return nil
}

func (p *externalProber) request(ctx context.Context, method, target string) (*http.Response, error) {
	requestCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	request, err := http.NewRequestWithContext(requestCtx, method, target, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	request.Header.Set("User-Agent", auditUserAgent)
	response, err := p.client.Do(request)
	if err != nil {
		cancel()
		return nil, err
	}
	response.Body = &detachedBody{ReadCloser: response.Body, cancel: cancel}
	return response, nil
}

func (p *externalProber) probe(ctx context.Context, candidate externalProbe) (int, int64, bool) {
	if !safeProbeURL(candidate.Target) || (candidate.Kind != "link" && candidate.Kind != "image") {
		return 0, 0, false
	}
	u, err := url.Parse(candidate.Target)
	if err != nil {
		return 0, 0, false
	}
	origin := u.Scheme + "://" + u.Host
	host := p.host(origin)
	host.mu.Lock()
	defer host.mu.Unlock()
	if host.robots == nil && !host.unavailable {
		if err := p.wait(ctx, host, strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))); err != nil {
			return 0, 0, false
		}
		response, err := p.request(ctx, http.MethodGet, origin+"/robots.txt")
		if err != nil {
			host.unavailable = true
			return 0, 0, false
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxProbeRobotsBytes+1))
		_ = response.Body.Close()
		if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
			empty := ""
			parsed := parseRobotsTxt(origin, &empty)
			host.robots = &parsed
		} else if response.StatusCode >= 200 && response.StatusCode < 300 && readErr == nil && len(body) <= maxProbeRobotsBytes {
			text := string(body)
			parsed := parseRobotsTxt(origin, &text)
			host.robots = &parsed
		} else {
			host.unavailable = true
			return 0, 0, false
		}
	}
	if host.unavailable || !host.robots.IsAllowed(candidate.Target) {
		return 0, 0, false
	}
	if err := p.wait(ctx, host, strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))); err != nil {
		return 0, 0, false
	}
	response, err := p.request(ctx, http.MethodHead, candidate.Target)
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= 300 && response.StatusCode < 400 || response.StatusCode == 401 || response.StatusCode == 403 || response.StatusCode == 405 || response.StatusCode == 429 {
		return 0, 0, false
	}
	if candidate.Kind == "image" && (!strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "image/") || response.ContentLength < 0) {
		return 0, 0, false
	}
	return response.StatusCode, response.ContentLength, true
}

func runExternalProbes(ctx context.Context, guard *Guard, candidates []externalProbe, limiter hostLimiter) ([]DetectedIssue, bool) {
	if guard == nil {
		guard = NewGuard()
	}
	client := *guard.NewClient(probeTimeout)
	client.Timeout = probeTimeout
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	prober := externalProber{client: &client, hosts: map[string]*externalHost{}, limiter: limiter}
	var mu sync.Mutex
	issues := []DetectedIssue{}
	incomplete := false
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, probeWorkers)
	for _, candidate := range candidates {
		select {
		case <-ctx.Done():
			mu.Lock()
			incomplete = true
			mu.Unlock()
			wg.Wait()
			return issues, incomplete
		case semaphore <- struct{}{}:
		}
		wg.Go(func() {
			defer func() { <-semaphore }()
			status, size, verified := prober.probe(ctx, candidate)
			mu.Lock()
			defer mu.Unlock()
			if !verified {
				incomplete = true
				return
			}
			issue := IssueType("")
			if candidate.Kind == "link" && status >= 400 {
				issue = IssueBrokenExternalLink
			}
			if candidate.Kind == "image" && status >= 200 && status < 300 && size > largeImageBytes {
				issue = IssueLargeImage
			}
			if issue == "" {
				return
			}
			for _, source := range candidate.Sources {
				id := source.ID
				issues = append(issues, DetectedIssue{IssueType: issue, PageID: &id, PageURL: source.URL, DedupeKey: candidate.Target, Details: map[string]any{"target": candidate.Target, "statusCode": status, "bytes": size}})
			}
		})
	}
	wg.Wait()
	return issues, incomplete
}
