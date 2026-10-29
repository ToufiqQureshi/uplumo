package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/jobs"
)

func TestCollectExternalProbesBoundedAndCredentialFree(t *testing.T) {
	page := samplePage("page", "https://example.com/")
	page.Links = nil
	page.Images = nil
	for i := range 201 {
		page.Links = append(page.Links, PageLink{TargetURL: fmt.Sprintf("https://other.example/%d", i)})
	}
	page.Links = append(page.Links, PageLink{TargetURL: "https://user@other.example/secret"}, PageLink{TargetURL: "http://127.0.0.1/private"})
	for i := range 31 {
		page.Images = append(page.Images, ImageRef{Src: fmt.Sprintf("/image%d.png", i)})
	}
	var signals aiSiteSignals
	signals.collectExternalProbes(page)
	if signals.probeLinks != 200 || signals.probeImages != 30 || !signals.probeLimited {
		t.Fatalf("probes = links %d images %d limited %v", signals.probeLinks, signals.probeImages, signals.probeLimited)
	}
	if len(signals.probes) != 230 {
		t.Fatalf("got %d candidates", len(signals.probes))
	}
	for _, probe := range signals.probes {
		if strings.Contains(probe.Target, "secret") || strings.Contains(probe.Target, "127.0.0.1") {
			t.Fatalf("unsafe candidate: %q", probe.Target)
		}
	}
	// A duplicate source does not spend another request or duplicate an issue.
	signals.collectExternalProbes(page)
	if len(signals.probes[0].Sources) != 1 {
		t.Fatalf("duplicate page recorded twice: %+v", signals.probes[0].Sources)
	}
}

func TestExternalProbeCandidateValidationAndSourceCap(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"https://outside.example/page", true},
		{"http://user@outside.example/page", false},
		{"http://127.0.0.1/", false},
		{"http://169.254.169.254/", false},
		{"ftp://outside.example/", false},
		{"https://outside.example/" + strings.Repeat("a", 2048), false},
	} {
		if got := safeProbeURL(tc.url); got != tc.want {
			t.Errorf("safe candidate %q = %v, want %v", tc.url, got, tc.want)
		}
	}
	var signals aiSiteSignals
	for i := range 4 {
		page := samplePage(fmt.Sprintf("page-%d", i), fmt.Sprintf("https://example.com/%d", i))
		page.Links = []PageLink{{TargetURL: "https://outside.example/shared"}}
		page.Images = nil
		signals.collectExternalProbes(page)
	}
	if len(signals.probes) != 1 || len(signals.probes[0].Sources) != 3 || !signals.probeLimited {
		t.Fatalf("shared target coverage: %+v limited %v", signals.probes, signals.probeLimited)
	}
}

func TestExternalProbeHonorsRobotsAndClassifiesOnlyVerifiedResults(t *testing.T) {
	for _, tc := range []struct {
		name        string
		robots      string
		status      int
		kind        string
		contentType string
		length      string
		want        IssueType
		incomplete  bool
		heads       int
	}{
		{"broken link", "User-agent: *\nAllow: /", 404, "link", "", "", IssueBrokenExternalLink, false, 1},
		{"healthy link", "", 200, "link", "", "", "", false, 1},
		{"robots blocked", "User-agent: *\nDisallow: /blocked", 404, "link", "", "", "", true, 0},
		{"oversized robots", strings.Repeat("x", 65537), 200, "link", "", "", "", true, 0},
		{"redirect unknown", "", 302, "link", "", "", "", true, 1},
		{"head unsupported", "", 405, "link", "", "", "", true, 1},
		{"rate limited", "", 429, "link", "", "", "", true, 1},
		{"large image", "", 200, "image", "image/png", "500001", IssueLargeImage, false, 1},
		{"small image", "", 200, "image", "image/png", "500000", "", false, 1},
		{"unknown image type", "", 200, "image", "text/html", "600000", "", true, 1},
		{"unknown image size", "", 200, "image", "image/png", "", "", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			methods := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				methods = append(methods, r.Method+" "+r.URL.Path)
				mu.Unlock()
				if r.URL.Path == "/robots.txt" {
					_, _ = w.Write([]byte(tc.robots))
					return
				}
				w.Header().Set("Content-Type", tc.contentType)
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			candidate := externalProbe{Target: "http://destination.example/blocked", Kind: tc.kind, Sources: []externalProbeSource{{ID: "page", URL: "https://example.com/"}}}
			issues, incomplete := runExternalProbes(ctx, &Guard{Client: dialAllTo(server)}, []externalProbe{candidate}, nil)
			if incomplete != tc.incomplete || hasIssue(issues, tc.want) != (tc.want != "") {
				t.Fatalf("issues = %+v, incomplete = %v", issues, incomplete)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(methods) != tc.heads+1 || methods[0] != "GET /robots.txt" || tc.heads == 1 && methods[1] != "HEAD /blocked" {
				t.Fatalf("requests = %v", methods)
			}
		})
	}
}

func TestExternalProbeRateLimitAndCancellation(t *testing.T) {
	var mu sync.Mutex
	var times []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		times = append(times, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	probes := []externalProbe{{Target: "http://rate.example/one", Kind: "link"}, {Target: "http://rate.example/two", Kind: "link"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	issues, incomplete := runExternalProbes(ctx, &Guard{Client: dialAllTo(server)}, probes, nil)
	cancel()
	if incomplete || len(issues) != 0 {
		t.Fatalf("issues = %+v, incomplete = %v", issues, incomplete)
	}
	mu.Lock()
	if len(times) != 3 || times[1].Sub(times[0]) < 900*time.Millisecond || times[2].Sub(times[1]) < 900*time.Millisecond {
		t.Errorf("same-host requests too close: %v", times)
	}
	mu.Unlock()
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, incomplete = runExternalProbes(ctx, &Guard{Client: dialAllTo(server)}, append(probes, probes...), nil)
	if !incomplete {
		t.Fatal("cancelled batch reported complete")
	}
}

func TestExternalHostLimiterIsSharedAcrossWorkersAndFailsClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rdb := progressRedis(ctx, t)
	first := redisHostLimiter{client: rdb}
	second := redisHostLimiter{client: rdb}
	host := "rate-" + suffix(t) + ".example"
	started := time.Now()
	if err := first.wait(ctx, host); err != nil {
		t.Fatal(err)
	}
	shortCtx, shortCancel := context.WithTimeout(ctx, 50*time.Millisecond)
	if err := second.wait(shortCtx, host); err == nil {
		t.Fatal("another worker bypassed the host limit")
	}
	shortCancel()
	if err := second.wait(ctx, "other-"+host); err != nil {
		t.Fatalf("another host was blocked: %v", err)
	}
	if err := second.wait(ctx, host); err != nil || time.Since(started) < 900*time.Millisecond {
		t.Fatalf("second slot arrived too early: %v", err)
	}
	if err := rdb.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.wait(ctx, "offline-"+host); err == nil {
		t.Fatal("Redis failure must not permit unthrottled network requests")
	}
	if err := (unavailableHostLimiter{}).wait(ctx, host); err == nil {
		t.Fatal("missing Redis must not permit unthrottled network requests")
	}
}

func TestExternalProbesLimitConcurrentHosts(t *testing.T) {
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	var releaseOnce sync.Once
	allow := func() { releaseOnce.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			started <- struct{}{}
			<-release
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	defer allow()
	probes := make([]externalProbe, 8)
	for i := range probes {
		probes[i] = externalProbe{Target: fmt.Sprintf("http://host%d.example/page", i), Kind: "link"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	done := make(chan bool, 1)
	go func() {
		_, incomplete := runExternalProbes(ctx, &Guard{Client: dialAllTo(server)}, probes, nil)
		done <- incomplete
	}()
	for range 4 {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("four workers did not start")
		}
	}
	select {
	case <-started:
		t.Fatal("more than four requests ran at once")
	case <-time.After(100 * time.Millisecond):
	}
	allow()
	select {
	case incomplete := <-done:
		if incomplete {
			t.Fatal("bounded requests unexpectedly incomplete")
		}
	case <-ctx.Done():
		t.Fatal("bounded requests did not finish")
	}
}

func TestExternalRequestHasFiveSecondTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	issues, incomplete := runExternalProbes(ctx, &Guard{Client: dialAllTo(server)}, []externalProbe{{Target: "http://slow.example/page", Kind: "link"}}, nil)
	if len(issues) != 0 || !incomplete || ctx.Err() != nil {
		t.Fatalf("timeout results: issues %+v incomplete %v outer context %v", issues, incomplete, ctx.Err())
	}
}

func TestExternalProbeSSRFAndRedirectGuard(t *testing.T) {
	for _, target := range []string{"http://127.0.0.1/private", "http://169.254.169.254/", "http://user@example.com/", "http://rebind.example/"} {
		t.Run(target, func(t *testing.T) {
			guard := &Guard{Resolver: fakeResolver{addrs: map[string][]string{"rebind.example": {"127.0.0.1"}}}}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			issues, incomplete := runExternalProbes(ctx, guard, []externalProbe{{Target: target, Kind: "link"}}, nil)
			if len(issues) != 0 || !incomplete {
				t.Fatalf("unsafe target produced issues %+v or complete result %v", issues, incomplete)
			}
		})
	}
}

func TestExternalChecksDoNotProbeWithoutRateLimiter(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	checks := NewExternalChecks(nil, nil, nil, &Guard{Client: dialAllTo(server)}, nil, nil)
	issues, incomplete := runExternalProbes(context.Background(), checks.guard, []externalProbe{{Target: "http://outside.example/", Kind: "link"}}, checks.limiter)
	if !incomplete || len(issues) != 0 || requests.Load() != 0 {
		t.Fatalf("without limiter: issues %+v, incomplete %v, requests %d", issues, incomplete, requests.Load())
	}
}

func TestExternalClaimsAreWorkspaceScopedAndFenced(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool := auditTestPool(ctx, t)
	queue, err := jobs.New(pool)
	if err != nil {
		t.Fatal(err)
	}
	checks := NewExternalChecks(queue, pool, NewRepository(pool), NewGuard(), nil, nil)
	name := "external-claim-" + suffix(t)
	inputs := []externalJob{{OrganizationID: name}, {OrganizationID: name}, {OrganizationID: name + "-other"}}
	claimed := make([]jobs.Job, 0, len(inputs))
	for i, input := range inputs {
		raw, _ := json.Marshal(input)
		if _, err := queue.Enqueue(ctx, jobs.EnqueueInput{Queue: ExternalChecksQueue, IdempotencyKey: fmt.Sprintf("%s-%d", name, i), Payload: raw}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM go_jobs WHERE queue=$1 AND idempotency_key LIKE $2`, ExternalChecksQueue, name+"%")
	})
	for range inputs {
		batch, err := queue.Claim(ctx, ExternalChecksQueue, name, time.Minute, 1)
		if err != nil || len(batch) != 1 {
			t.Fatalf("claim = %+v, %v", batch, err)
		}
		claimed = append(claimed, batch[0])
	}
	for i, want := range []bool{true, false, true} {
		started, err := checks.claim(ctx, inputs[i], claimed[i])
		if err != nil || started != want {
			t.Fatalf("workspace %d started = %v err = %v, want %v", i, started, err, want)
		}
	}
	stale := claimed[2]
	stale.ClaimVersion++
	if _, err := checks.claim(ctx, inputs[2], stale); !errors.Is(err, jobs.ErrClaimLost) {
		t.Fatalf("stale claim error = %v", err)
	}
}

func TestExternalInputValidatesStoredPagesAndTenant(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool := auditTestPool(ctx, t)
	repo := NewRepository(pool)
	org := "org-" + suffix(t)
	project := seedProject(ctx, t, pool, org)
	auditID := "audit-" + suffix(t)
	seedRepositoryAudit(ctx, t, repo, auditID, project)
	page := samplePage("page-"+suffix(t), "https://example.com/")
	if err := repo.InsertCrawledBatch(ctx, auditID, []CrawledPageResult{page}, nil); err != nil {
		t.Fatal(err)
	}
	checks := NewExternalChecks(nil, pool, repo, nil, nil, nil)
	input := externalJob{AuditID: auditID, ProjectID: project, OrganizationID: org, StartURL: "https://example.com/", Probes: []externalProbe{{Target: "https://outside.example/", Kind: "link", Sources: []externalProbeSource{{ID: page.ID, URL: page.URL}}}}}
	if valid, err := checks.validateInput(ctx, input); err != nil || !valid {
		t.Fatalf("valid input = %v, %v", valid, err)
	}
	for _, change := range []func(*externalJob){
		func(j *externalJob) { j.OrganizationID = "other" },
		func(j *externalJob) { j.StartURL = "https://other.example/" },
		func(j *externalJob) { j.Probes[0].Sources[0].ID = "other-page" },
		func(j *externalJob) { j.Probes[0].Target = "http://127.0.0.1/" },
	} {
		cloneBytes, _ := json.Marshal(input)
		var clone externalJob
		_ = json.Unmarshal(cloneBytes, &clone)
		change(&clone)
		if valid, err := checks.validateInput(ctx, clone); err != nil || valid {
			t.Fatalf("unsafe input accepted: %+v, %v", clone, err)
		}
	}
}

func TestExternalJobChecksOnceAndRetainsPartialResults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool := auditTestPool(ctx, t)
	repo := NewRepository(pool)
	org := "org-" + suffix(t)
	project := seedProject(ctx, t, pool, org)
	auditID := "audit-" + suffix(t)
	seedRepositoryAudit(ctx, t, repo, auditID, project)
	page := samplePage("page-"+suffix(t), "https://example.com/")
	page.Links = []PageLink{{TargetURL: "http://outside.example/broken"}}
	page.Images = nil
	if err := repo.InsertCrawledBatch(ctx, auditID, []CrawledPageResult{page}, nil); err != nil {
		t.Fatal(err)
	}
	queue, err := jobs.New(pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM go_jobs WHERE queue=$1 AND idempotency_key=$2`, ExternalChecksQueue, auditID)
	})
	var mu sync.Mutex
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nAllow: /"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	rdb := progressRedis(ctx, t)
	checks := NewExternalChecks(queue, pool, repo, &Guard{Client: dialAllTo(server)}, rdb, nil)
	var signals aiSiteSignals
	signals.collectExternalProbes(page)
	input := JobPayload{AuditID: auditID, ProjectID: project, OrganizationID: org, StartURL: page.URL}
	if err := checks.schedule(ctx, input, &signals); err != nil {
		t.Fatal(err)
	}
	batch, err := queue.Claim(ctx, ExternalChecksQueue, "external-test-"+suffix(t), time.Minute, 1)
	if err != nil || len(batch) != 1 {
		t.Fatalf("claim = %+v, %v", batch, err)
	}
	job := batch[0]
	if err := checks.Handler()(ctx, job); err != nil {
		t.Fatal(err)
	}
	var broken, pending, incomplete int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE issue_type=$2), count(*) FILTER (WHERE issue_type=$3), count(*) FILTER (WHERE issue_type=$4)
		FROM go_audit_issues WHERE audit_id=$1`, auditID, IssueBrokenExternalLink, IssueExternalChecksPending, IssueExternalChecksIncomplete).Scan(&broken, &pending, &incomplete); err != nil {
		t.Fatal(err)
	}
	if broken != 1 || pending != 0 || incomplete != 0 {
		t.Fatalf("issues: broken %d pending %d incomplete %d", broken, pending, incomplete)
	}
	var payload map[string]any
	if err := pool.QueryRow(ctx, `SELECT payload FROM go_jobs WHERE id=$1`, job.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if _, retained := payload["probes"]; retained {
		t.Fatal("completed job retained candidate URLs")
	}
	// An ambiguous retry must not make another HEAD or replace saved findings.
	var retry externalJob
	if err := json.Unmarshal(job.Payload, &retry); err != nil {
		t.Fatal(err)
	}
	retry.Started = true
	job.Payload, _ = json.Marshal(retry)
	if err := checks.Handler()(ctx, job); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 2 || requests[0] != "GET /robots.txt" || requests[1] != "HEAD /broken" {
		t.Fatalf("requests on retry = %v", requests)
	}
}
