package audit

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
)

// maxCrawlDepth bounds link-following depth.
const maxCrawlDepth = 6

// Runner executes one audit job end to end: discovery, crawl, cross-page
// checks, Lighthouse, then finalize. It is idempotent: a retry re-derives
// deterministic row ids and upserts, so a partially written audit completes
// instead of duplicating.
type Runner struct {
	repo           Store
	progress       *Progress
	guard          *Guard
	crawler        *Crawler
	lighthouse     LighthouseProvider
	fieldData      FieldDataProvider
	renderer       JavaScriptRenderer
	rendering      RenderingMeter
	externalChecks *ExternalChecks
	logger         *slog.Logger
	now            func() time.Time
	resolver       CrawlerAccessResolver
	// minCheckTime is minLighthouseCheckTime; tests shorten it.
	minCheckTime time.Duration
}

// RunnerConfig configures a Runner.
type RunnerConfig struct {
	Repository Store
	Progress   *Progress
	Guard      *Guard
	Crawler    *Crawler
	Lighthouse LighthouseProvider
	// FieldData looks up real-visitor data for each check; nil uses only
	// what PageSpeed Insights embeds.
	FieldData      FieldDataProvider
	Renderer       JavaScriptRenderer
	Rendering      RenderingMeter
	ExternalChecks *ExternalChecks
	Logger         *slog.Logger
	Now            func() time.Time
	Resolver       CrawlerAccessResolver
}

// NewRunner builds a Runner.
func NewRunner(config RunnerConfig) *Runner {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	guard := config.Guard
	if guard == nil {
		guard = NewGuard()
	}
	crawler := config.Crawler
	if crawler == nil {
		crawler = NewCrawler(CrawlerOptions{Guard: guard})
	}
	return &Runner{
		repo: config.Repository, progress: config.Progress, guard: guard, crawler: crawler,
		lighthouse: config.Lighthouse, fieldData: config.FieldData, renderer: config.Renderer,
		rendering: config.Rendering, externalChecks: config.ExternalChecks, logger: logger, now: now,
		minCheckTime: minLighthouseCheckTime,
		resolver:     config.Resolver,
	}
}

const (
	// auditDeadline bounds one audit's crawl and checks.
	auditDeadline = 30 * time.Minute
	// lighthouseBudget bounds the Lighthouse checks that follow, on their own
	// clock: a crawl that used its whole deadline must not leave them none.
	// 20 checks take about 10 minutes; crawl + checks stays inside the job
	// queue's 45-minute timeout.
	lighthouseBudget = 12 * time.Minute
	// minLighthouseCheckTime is the least time worth starting a check with;
	// below it the remaining checks are recorded as skipped.
	minLighthouseCheckTime = 30 * time.Second
	// persistTimeout bounds the writes that save finished work after the
	// phase's own deadline has passed.
	persistTimeout = 30 * time.Second
)

// Run executes the audit. A returned error means the audit failed and was
// marked failed; a nil error means it completed (possibly with partial data).
func (r *Runner) Run(ctx context.Context, job JobPayload) error {
	runCtx, cancel := context.WithTimeout(ctx, auditDeadline)
	defer cancel()

	if err := r.setPhase(runCtx, job, "discovery"); err != nil {
		return err
	}
	if err := r.run(runCtx, ctx, job); err != nil {
		r.failAudit(ctx, job, err)
		return err
	}
	return nil
}

// run performs the phases and completes the audit. ctx bounds the crawl;
// jobCtx is the job's own context, which the Lighthouse phase and completion
// run under.
func (r *Runner) run(ctx, jobCtx context.Context, job JobPayload) error {
	maxPages := job.Config.MaxPages
	if maxPages <= 0 {
		maxPages = DefaultAuditPages
	}
	origin := getOrigin(job.StartURL)
	if origin == "" {
		return ErrStartURLInvalid
	}

	access := r.resolveCrawlerAccess(ctx, job, origin)
	discoverer := discoverer{guard: r.guard, client: r.guard.NewClient(0), access: access, now: r.now}
	seedURLs, robotsText := discoverer.DiscoverURLs(ctx, origin, maxPages, access)
	robots := parseRobotsTxt(origin, robotsText)
	aiIssues := aiAccessIssues(origin, robotsText, discoverer.hasLLMsTxt(ctx, origin))
	if err := r.repo.InsertIssues(ctx, job.AuditID, aiIssues); err != nil {
		return err
	}

	ai := &aiSiteSignals{}
	pagesCrawled, err := r.crawl(ctx, job, origin, seedURLs, robots, maxPages, ai)
	if err != nil {
		return err
	}

	// The crawl may have used its whole deadline. Optional paid rendering and
	// final checks run on the job's clock, so a full crawl is still finalized.
	ctx = jobCtx
	if job.Config.RenderJavaScript && r.renderer != nil {
		if err := r.renderJavaScriptPages(ctx, job, ai); err != nil {
			r.logger.WarnContext(ctx, "render app-shell pages for Site Audit", "audit_id", job.AuditID, "err", err)
		}
	}
	if err := r.finalizeChecks(ctx, job); err != nil {
		return err
	}
	if err := r.repo.InsertIssues(ctx, job.AuditID, ai.issues(job.StartURL)); err != nil {
		return err
	}
	lighthouseCtx, cancelLighthouse := context.WithTimeout(ctx, lighthouseBudget)
	defer cancelLighthouse()
	if err := r.lighthousePhase(lighthouseCtx, job); err != nil {
		return err
	}
	if err := r.repo.CompleteAudit(ctx, job.AuditID, job.AuditID, pagesCrawled, maxPages); err != nil {
		return err
	}
	if r.externalChecks != nil {
		if err := r.externalChecks.schedule(ctx, job, ai); err != nil {
			r.logger.WarnContext(ctx, "schedule external audit checks", "audit_id", job.AuditID, "err", err)
		}
	}
	if r.progress != nil {
		_ = r.progress.Clear(ctx, job.AuditID)
	}
	if r.rendering != nil && len(job.RenderingLocks) > 0 {
		_ = r.rendering.Settle(ctx, job.OrganizationID, job.AuditID, job.RenderingLocks, RenderUsage{})
	}
	return nil
}

// renderJavaScriptPages sends only pages already flagged as app shells. The
// repository claim is permanent, so a worker restart cannot buy the same URL
// again after an ambiguous provider outcome.
func (r *Runner) renderJavaScriptPages(ctx context.Context, job JobPayload, ai *aiSiteSignals) error {
	shellIDs, err := r.repo.ShellPageIDs(ctx, job.AuditID)
	if err != nil {
		return err
	}
	if len(shellIDs) == 0 {
		return nil
	}
	pages, err := r.repo.ListSlimPages(ctx, job.AuditID)
	if err != nil {
		return err
	}
	byURL := make(map[string]SlimPage, len(pages))
	urls := make([]string, 0, min(len(shellIDs), MaxRenderedPagesPerAudit))
	for _, page := range pages {
		if _, shell := shellIDs[page.ID]; !shell || !isOKHTMLPage(page) {
			continue
		}
		byURL[page.URL] = page
		urls = append(urls, page.URL)
	}
	slices.Sort(urls)
	urls = urls[:min(len(urls), MaxRenderedPagesPerAudit)]
	claimed, err := r.repo.ClaimJavaScriptRenderPages(ctx, job.AuditID, urls)
	if err != nil {
		return err
	}
	if len(claimed) == 0 {
		return nil
	}
	if err := r.setPhase(ctx, job, "javascript_rendering"); err != nil {
		return err
	}
	rendered, renderErr := r.renderer.Render(ctx, job.OrganizationID, job.AuditID, claimed)
	for _, pageURL := range claimed {
		entry, ok := rendered[pageURL]
		if !ok || strings.TrimSpace(entry.HTML) == "" || entry.StatusCode < 200 || entry.StatusCode >= 300 {
			if markErr := r.repo.MarkJavaScriptRenderFailed(ctx, job.AuditID, pageURL); markErr != nil {
				r.logger.WarnContext(ctx, "record failed JavaScript render", "audit_id", job.AuditID, "url", pageURL, "err", markErr)
			}
			continue
		}
		slim := byURL[pageURL]
		analysis := AnalyzeHTML(entry.HTML, pageURL, entry.StatusCode, slim.ResponseTimeMs, "")
		page := CrawledPageResult{
			ID: slim.ID, URL: pageURL, StatusCode: entry.StatusCode, FetchClass: FetchOK,
			XRobotsTag: slim.XRobotsTag, HeaderCanonicalURL: slim.HeaderCanonicalURL,
			ResponseTimeMs: slim.ResponseTimeMs, CrawlDepth: slim.CrawlDepth, InSitemap: slim.InSitemap,
			IsHTML: true, HTMLBytes: len(entry.HTML),
		}
		applyAnalysis(&page, analysis, http.Header{})
		page.XRobotsTag = slim.XRobotsTag
		page.AI.SnippetBlocked = page.AI.SnippetBlocked || blocksSnippet(slim.XRobotsTag)
		page.AI.HSTS = slim.HSTS
		page.HeaderCanonicalURL = slim.HeaderCanonicalURL
		page.IsIndexable = computeIsIndexable(page.RobotsMeta, page.XRobotsTag)
		if err := r.repo.SaveJavaScriptRenderedPage(ctx, job.AuditID, page, entry.CostUSD); err != nil {
			return err
		}
		ai.add(page, page.URL == job.StartURL || isHomePage(page.URL))
	}
	return renderErr
}

// crawlQueueItem is one URL waiting to be fetched.
type crawlQueueItem struct {
	url       string
	depth     int
	inSitemap bool
}

// crawl runs the BFS crawl, persisting each sub-batch as it lands.
func (r *Runner) crawl(ctx context.Context, job JobPayload, _ string, seedURLs []string, robots RobotsResult, maxPages int, ai *aiSiteSignals) (int, error) {
	startNormalized, ok := normalizeURL(job.StartURL, "")
	if !ok {
		return 0, ErrStartURLInvalid
	}
	visited := map[string]struct{}{}
	seeds := map[string]bool{}
	var blockedSitemap []DetectedIssue
	for _, seed := range seedURLs {
		if normalized, ok := normalizeURL(seed, ""); ok {
			seeds[normalized] = true
			if !robots.IsAllowed(normalized) {
				blockedSitemap = append(blockedSitemap, DetectedIssue{IssueType: IssueSitemapBlocked, PageURL: normalized})
			}
		}
	}
	if err := r.repo.InsertIssues(ctx, job.AuditID, blockedSitemap); err != nil {
		return 0, err
	}
	queue := []crawlQueueItem{}
	if robots.IsAllowed(startNormalized) {
		queue = append(queue, crawlQueueItem{url: startNormalized, depth: 0, inSitemap: seeds[startNormalized]})
	}
	for _, seed := range seedURLs {
		if normalized, ok := normalizeURL(seed, ""); ok && robots.IsAllowed(normalized) {
			queue = append(queue, crawlQueueItem{url: normalized, inSitemap: true})
		}
	}

	throttle := NewCrawlThrottle(r.now().Add(auditDeadline), nil, nil)
	window := CrawlWindow.Initial
	pagesCrawled := 0
	stoppedEarly := false

	for len(queue) > 0 && pagesCrawled < maxPages {
		if ctx.Err() != nil {
			return pagesCrawled, ctx.Err()
		}
		batchSize := min(window, len(queue))
		batchSize = min(batchSize, maxPages-pagesCrawled)
		batch := queue[:batchSize]
		queue = queue[batchSize:]

		ready, err := throttle.Ready(ctx)
		if err != nil {
			return pagesCrawled, err
		}
		if !ready {
			stoppedEarly = true
			break
		}

		results := make([]CrawlResult, len(batch))
		for i, item := range batch {
			if _, seen := visited[item.url]; seen {
				continue
			}
			visited[item.url] = struct{}{}
			results[i] = r.fetchWithRetry(ctx, throttle, job, item.url)
		}

		pages := make([]CrawledPageResult, 0, len(batch))
		issues := make([]DetectedIssue, 0, len(batch))
		progressEntries := make([]ProgressEntry, 0, len(batch))
		for i, item := range batch {
			result := results[i]
			if result.URL == "" {
				continue // already visited
			}
			page := result.CrawledPageResult
			page.InSitemap = item.inSitemap
			page.ID = DeterministicAuditRowID(job.AuditID, page.URL)
			depth := item.depth
			if item.depth != 0 || i != 0 {
				page.CrawlDepth = &depth
			}
			page.HTMLBytes = result.HTMLBytesRead
			pages = append(pages, page)
			ai.add(page, page.URL == startNormalized || isHomePage(page.URL))
			issues = append(issues, runPageReporters(page)...)
			progressEntries = append(progressEntries, ProgressEntry{
				URL: page.URL, StatusCode: page.StatusCode, Title: page.Title, CrawledAt: r.now().UnixMilli(),
			})
		}

		if err := r.repo.InsertCrawledBatch(ctx, job.AuditID, pages, issues); err != nil {
			return pagesCrawled, err
		}
		pagesCrawled += len(pages)
		if r.progress != nil && len(progressEntries) > 0 {
			if err := r.progress.PushCrawledURLs(ctx, job.AuditID, progressEntries); err != nil {
				r.logger.WarnContext(ctx, "push crawl progress", "audit_id", job.AuditID, "err", err)
			}
		}
		crawled := pagesCrawled
		if err := r.repo.UpdateAuditProgress(ctx, job.AuditID, job.AuditID, ProgressUpdate{
			PagesCrawled: &crawled, CurrentPhase: phasePtr("crawling"),
		}); err != nil {
			return pagesCrawled, err
		}

		window = AdjustCrawlWindow(window, pages, CrawlWindow)

		// Within the same crawl budget, visit internal canonical and language
		// targets as well as ordinary links so their status can be checked.
		for _, page := range pages {
			if page.CrawlDepth != nil && *page.CrawlDepth >= maxCrawlDepth {
				continue
			}
			links := append([]PageLink(nil), page.Links...)
			targets := []string{page.CanonicalURL, page.HeaderCanonicalURL}
			for _, alternate := range page.Technical.Hreflangs {
				targets = append(targets, alternate.URL)
			}
			for _, target := range targets {
				if target == "" {
					continue
				}
				normalized, ok := normalizeURL(target, page.URL)
				if ok && isSameOrigin(normalized, job.StartURL) {
					links = append(links, PageLink{TargetURL: normalized, IsInternal: true})
				}
			}
			for _, link := range links {
				if !link.IsInternal {
					continue
				}
				next, ok := normalizeURL(link.TargetURL, "")
				if !ok {
					continue
				}
				if _, seen := visited[next]; seen {
					continue
				}
				if !isCrawlableURL(next) || !robots.IsAllowed(next) {
					continue
				}
				depth := 0
				if page.CrawlDepth != nil {
					depth = *page.CrawlDepth + 1
				} else {
					depth = 1
				}
				queue = append(queue, crawlQueueItem{url: next, depth: depth})
			}
		}
	}

	if stoppedEarly || throttle.Stopped() {
		issue := DetectedIssue{IssueType: IssueCrawlRateLimited, PageURL: job.StartURL}
		if err := r.repo.InsertIssues(ctx, job.AuditID, []DetectedIssue{issue}); err != nil {
			return pagesCrawled, err
		}
	}
	return pagesCrawled, nil
}

// fetchWithRetry fetches a URL, retrying a 429 while the throttle allows.
func (r *Runner) fetchWithRetry(ctx context.Context, throttle *CrawlThrottle, job JobPayload, rawURL string) CrawlResult {
	attempt := 0
	for {
		crawler := *r.crawler
		crawler.access = r.resolveCrawlerAccess(ctx, job, rawURL)
		result := crawler.FetchPage(ctx, rawURL)
		if result.FetchClass != FetchRateLimited {
			if result.FetchClass != FetchError {
				_ = throttle.Recovered(ctx)
			}
			result.RateLimited = attempt > 0
			return result
		}
		retryAfter := ""
		allowed, err := throttle.Backoff(ctx, attempt, retryAfter)
		if err != nil || !allowed {
			result.RateLimited = true
			return result
		}
		ready, err := throttle.Ready(ctx)
		if err != nil || !ready {
			result.RateLimited = true
			return result
		}
		attempt++
		result.RateLimited = true
	}
}

func (r *Runner) resolveCrawlerAccess(ctx context.Context, job JobPayload, rawURL string) *CrawlerAccess {
	if r.resolver == nil {
		return nil
	}
	host := hostOf(rawURL)
	if host == "" {
		return nil
	}
	access, _, err := r.resolver.Resolve(ctx, job.OrganizationID, job.ProjectID, host)
	if err != nil {
		r.logger.WarnContext(ctx, "resolve crawler access", "host", host, "err", err)
		return nil
	}
	return access
}

// finalizeChecks runs the cross-page checks and persists their issues.
func (r *Runner) finalizeChecks(ctx context.Context, job JobPayload) error {
	issues, _, err := r.repo.RunMultipageChecks(ctx, job.AuditID)
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		if err := r.repo.InsertIssues(ctx, job.AuditID, issues); err != nil {
			return err
		}
	}
	return nil
}

// lighthousePhase runs the sampled Lighthouse checks.
func (r *Runner) lighthousePhase(ctx context.Context, job JobPayload) error {
	if job.Config.LighthouseStrategy == LighthouseNone || r.lighthouse == nil {
		return nil
	}
	if err := r.setPhase(ctx, job, "lighthouse"); err != nil {
		return err
	}
	pages, err := r.repo.ListSlimPages(ctx, job.AuditID)
	if err != nil {
		return err
	}
	samples := make([]LighthouseSamplePage, 0, len(pages))
	for _, page := range pages {
		status := 0
		if page.StatusCode != nil {
			status = *page.StatusCode
		}
		samples = append(samples, LighthouseSamplePage{URL: page.URL, StatusCode: status, FetchClass: page.FetchClass})
	}
	selected := SelectLighthouseSample(samples, job.StartURL, job.Config.LighthouseStrategy)
	results := make([]LighthouseRecord, 0, len(selected)*2)
	completed, failed := 0, 0
	for _, pageURL := range selected {
		pageID := DeterministicAuditRowID(job.AuditID, pageURL)
		for _, strategy := range []string{"mobile", "desktop"} {
			var fetched LighthouseFetchResult
			if deadline, ok := ctx.Deadline(); ctx.Err() != nil || (ok && time.Until(deadline) < r.minCheckTime) {
				fetched = FailedLighthouseFetch(pageURL, pageID, strategy, "Skipped: the audit ran out of time for page speed checks.")
			} else {
				fetched = FetchLighthouseResult(ctx, r.lighthouse, r.fieldData, r.logger, job.OrganizationID, pageURL, pageID, strategy)
			}
			record := LighthouseRecord{
				PageID: pageID, URL: pageURL, Strategy: strategy,
				PerformanceScore:   intFromFloat(fetched.Result.PerformanceScore),
				AccessibilityScore: intFromFloat(fetched.Result.AccessibilityScore),
				BestPracticesScore: intFromFloat(fetched.Result.BestPracticesScore),
				SeoScore:           intFromFloat(fetched.Result.SeoScore),
				LcpMs:              fetched.Result.LcpMs, CLS: fetched.Result.CLS, InpMs: fetched.Result.InpMs, TtfbMs: fetched.Result.TtfbMs,
				ErrorMessage: fetched.Result.ErrorMessage,
			}
			if fetched.PayloadJSON != "" {
				payload := fetched.PayloadJSON
				record.PayloadJSON = &payload
			}
			results = append(results, record)
			if fetched.Result.ErrorMessage == nil {
				completed++
			} else {
				failed++
			}
		}
	}
	// The phase's deadline may have passed by now; the finished checks are
	// saved regardless, on a short clock of their own.
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistTimeout)
	defer cancel()
	if err := r.repo.InsertLighthouseResults(persistCtx, job.AuditID, results); err != nil {
		return err
	}
	return r.repo.UpdateAuditProgress(persistCtx, job.AuditID, job.AuditID, ProgressUpdate{
		LighthouseCompleted: &completed, LighthouseFailed: &failed, CurrentPhase: phasePtr("finalizing"),
	})
}

// setPhase records the audit's current phase.
func (r *Runner) setPhase(ctx context.Context, job JobPayload, phase string) error {
	return r.repo.UpdateAuditProgress(ctx, job.AuditID, job.AuditID, ProgressUpdate{CurrentPhase: &phase})
}

// failAudit marks the audit failed, classifying the error.
func (r *Runner) failAudit(ctx context.Context, job JobPayload, cause error) {
	failCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	info := ClassifyAuditError(cause)
	if err := r.repo.FailAudit(failCtx, job.AuditID, job.AuditID, info, ""); err != nil {
		r.logger.ErrorContext(failCtx, "mark audit failed", "audit_id", job.AuditID, "err", err)
	}
	if r.rendering != nil && len(job.RenderingLocks) > 0 {
		if err := r.rendering.Release(failCtx, job.RenderingLocks); err != nil {
			r.logger.WarnContext(failCtx, "release rendering locks", "audit_id", job.AuditID, "err", err)
		}
	}
	if r.progress != nil {
		_ = r.progress.Clear(failCtx, job.AuditID)
	}
}

// phasePtr returns a pointer to a phase string.
func phasePtr(phase string) *string { return &phase }

// intFromFloat rounds an optional float score to an optional int.
func intFromFloat(value *float64) *int {
	if value == nil {
		return nil
	}
	result := int(*value + 0.5)
	return &result
}
