package domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrentDomainColdMissesBuyOneTaskPerOrganization(t *testing.T) {
	for _, kind := range []string{"overview", "keywords", "pages"} {
		for _, separate := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/separate=%t", kind, separate), func(t *testing.T) {
				provider := &fakeProvider{result: func(path string) any {
					time.Sleep(50 * time.Millisecond)
					switch path {
					case domainRankOverviewPath:
						return overviewResult(100, 10)(path)
					case rankedKeywordsPath:
						return rankedKeywordsResult(1, rankedItem("hotel", 1, 20, 100, 0.2, 0, "https://example.com/", "/"))(path)
					default:
						return map[string]any{"total_count": 1, "items": []any{map[string]any{"page_address": "https://example.com/"}}}
					}
				}}
				svc := newTestService(t, provider, &memoryCache{})
				start := make(chan struct{})
				var wg sync.WaitGroup
				for i := range 8 {
					wg.Go(func() {
						<-start
						org := "org"
						if separate && i%2 == 1 {
							org = "other-org"
						}
						var err error
						switch kind {
						case "overview":
							_, err = svc.Overview(t.Context(), org, "project", "example.com", ScopeDomain, 2840, "en")
						case "keywords":
							_, err = svc.KeywordsPage(t.Context(), org, KeywordsPageInput{ProjectID: "project", Domain: "example.com", Scope: ScopeDomain, Page: 1, PageSize: 50})
						case "pages":
							_, err = svc.PagesPage(t.Context(), org, PagesPageInput{ProjectID: "project", Domain: "example.com", Scope: ScopeDomain, Page: 1, PageSize: 50})
						}
						if err != nil {
							t.Errorf("lookup: %v", err)
						}
					})
				}
				close(start)
				wg.Wait()
				want := int32(1)
				if separate {
					want = 2
				}
				if provider.calls.Load() != want {
					t.Fatalf("provider tasks=%d, want %d", provider.calls.Load(), want)
				}
			})
		}
	}
}

func TestCancelledDomainReportBuysNothing(t *testing.T) {
	provider := &fakeProvider{result: overviewResult(1, 1)}
	svc := newTestService(t, provider, &memoryCache{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _ = svc.Overview(ctx, "org", "project", "example.com", ScopeDomain, 2840, "en")
	_, _ = svc.KeywordsPage(ctx, "org", KeywordsPageInput{ProjectID: "project", Domain: "example.com", Scope: ScopeDomain})
	_, _ = svc.PagesPage(ctx, "org", PagesPageInput{ProjectID: "project", Domain: "example.com", Scope: ScopeDomain})
	if provider.calls.Load() != 0 {
		t.Fatalf("cancelled reports bought %d tasks", provider.calls.Load())
	}
}

func TestDomainReportFinishesForOtherWaitersAfterFirstCallerLeaves(t *testing.T) {
	ctx, timeout := context.WithTimeout(t.Context(), 2*time.Second)
	defer timeout()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	provider := &fakeProvider{result: func(path string) any {
		entered <- struct{}{}
		<-release
		return overviewResult(100, 10)(path)
	}}
	svc := newTestService(t, provider, &memoryCache{})
	firstCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	first := make(chan error, 1)
	go func() {
		_, err := svc.Overview(firstCtx, "org", "project", "example.com", ScopeDomain, 2840, "en")
		first <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled waiter: %v", err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	second := make(chan error, 1)
	go func() {
		result, err := svc.Overview(ctx, "org", "project", "example.com/blog", ScopeSubfolder, 2840, "en")
		if err == nil && (!result.HasData || result.Scope != ScopeSubfolder || result.DisplayTarget != "example.com/blog") {
			err = fmt.Errorf("wrong shared report or caller scope: %+v", result)
		}
		second <- err
	}()
	unblock()
	select {
	case err := <-second:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if provider.calls.Load() != 1 {
		t.Fatalf("waiter cancellation bought %d tasks", provider.calls.Load())
	}
}
