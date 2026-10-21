package domain

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestConcurrentSuggestionsBuyOneTaskAndKeepOrganizationsSeparate(t *testing.T) {
	for _, cacheDown := range []bool{false, true} {
		t.Run(map[bool]string{false: "cache available", true: "cache unavailable"}[cacheDown], func(t *testing.T) {
			provider := &fakeProvider{result: func(path string) any {
				time.Sleep(100 * time.Millisecond)
				return rankedKeywordsResult(1, rankedItem("seo", 1, 2, 3, 4, 5, "https://example.com/p", ""))(path)
			}}
			svc := newTestService(t, provider, &memoryCache{fail: cacheDown})
			start := make(chan struct{})
			var group sync.WaitGroup
			for range 8 {
				group.Go(func() {
					<-start
					rows, err := svc.KeywordSuggestions(t.Context(), "org-a", "project", "example.com", ScopeDomain, 2840, "en")
					if err != nil || len(rows) != 1 {
						t.Errorf("suggestions=%v err=%v", rows, err)
					}
				})
			}
			close(start)
			group.Wait()
			if provider.calls.Load() != 1 {
				t.Fatalf("eight concurrent requests purchased %d tasks, want one", provider.calls.Load())
			}
			if _, err := svc.KeywordSuggestions(t.Context(), "org-b", "project", "example.com", ScopeDomain, 2840, "en"); err != nil {
				t.Fatal(err)
			}
			if provider.calls.Load() != 2 {
				t.Fatalf("another organization reused private work: purchases=%d", provider.calls.Load())
			}
		})
	}
}

func TestCancelledSuggestionsDoNotStartPaidWork(t *testing.T) {
	provider := &fakeProvider{result: rankedKeywordsResult(0)}
	svc := newTestService(t, provider, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := svc.KeywordSuggestions(ctx, "org", "project", "example.com", ScopeDomain, 2840, "en"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled suggestions=%v", err)
	}
	if provider.calls.Load() != 0 {
		t.Fatal("cancelled request purchased a task")
	}
}

func TestDifferentOrganizationsDoNotShareInflightSuggestions(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	provider := &fakeProvider{result: func(path string) any {
		entered <- struct{}{}
		<-release
		return rankedKeywordsResult(0)(path)
	}}
	svc := newTestService(t, provider, nil)
	var group sync.WaitGroup
	for _, org := range []string{"org-a", "org-b"} {
		group.Go(func() {
			if _, err := svc.KeywordSuggestions(ctx, org, "project", "example.com", ScopeDomain, 2840, "en"); err != nil {
				t.Error(err)
			}
		})
	}
	for range 2 {
		select {
		case <-entered:
		case <-ctx.Done():
			unblock()
			group.Wait()
			t.Fatal("one organization joined another organization's in-flight purchase")
		}
	}
	unblock()
	group.Wait()
	if provider.calls.Load() != 2 {
		t.Fatalf("independent organization purchases=%d", provider.calls.Load())
	}
}

func TestSuggestionsKeepSharedPurchaseWhenFirstWaiterLeaves(t *testing.T) {
	ctx, timeout := context.WithTimeout(t.Context(), 2*time.Second)
	defer timeout()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	provider := &fakeProvider{result: func(path string) any {
		entered <- struct{}{}
		<-release
		return rankedKeywordsResult(1, rankedItem("seo", 1, 2, 3, 4, 5, "https://example.com/p", ""))(path)
	}}
	svc := newTestService(t, provider, &memoryCache{})
	firstCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	first := make(chan error, 1)
	go func() {
		_, err := svc.KeywordSuggestions(firstCtx, "org", "project", "example.com", ScopeDomain, 2840, "en")
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
			t.Fatalf("first waiter=%v", err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	second := make(chan error, 1)
	go func() {
		rows, err := svc.KeywordSuggestions(ctx, "org", "project", "example.com", ScopeDomain, 2840, "en")
		if err == nil && len(rows) != 1 {
			err = errors.New("missing shared suggestions")
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
		t.Fatalf("waiter cancellation bought another task: %d", provider.calls.Load())
	}
}
