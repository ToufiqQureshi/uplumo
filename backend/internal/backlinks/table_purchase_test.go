package backlinks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func TestCancelledTablePreparationDoesNotReservePaidWork(t *testing.T) {
	for _, kind := range []string{"rows", "domains", "pages"} {
		t.Run(kind, func(t *testing.T) {
			guard := &spendtest.Guard{}
			svc, _, calls := newOverviewService(t, 90, http.StatusOK, guard)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			svc.cache = &spendtest.CancelOnReadCache{Cancel: cancel}
			in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
			page := pageInput{Page: 1, PageSize: 50, Sort: "rank,desc", Mode: "as_is"}
			var err error
			switch kind {
			case "rows":
				_, err = svc.Rows(ctx, "org", in, page, rowsFilters{}, false)
			case "domains":
				_, err = svc.Domains(ctx, "org", in, page, domainsFilters{})
			case "pages":
				_, err = svc.Pages(ctx, "org", in, page, pagesFilters{})
			}
			if !errors.Is(err, context.Canceled) || len(guard.Held()) != 0 || calls.Load() != 0 {
				t.Fatalf("cancelled table: err=%v holds=%d calls=%d", err, len(guard.Held()), calls.Load())
			}
		})
	}
}

func TestUndercountedWholeListReusesThePurchasedFirstBlock(t *testing.T) {
	for _, kind := range []string{"rows", "domains"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				items := make([]any, blockSize)
				for i := range items {
					items[i] = map[string]any{"domain": fmt.Sprintf("s%d.example", i), "url_from": fmt.Sprintf("https://s%d.example/", i)}
				}
				providerOK(w, r.URL.Path, map[string]any{"items": items, "total_count": 2000})
			})
			svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
			target, err := normalizeTarget("example.com", string(ScopeDomain))
			if err != nil {
				t.Fatal(err)
			}
			key, err := cacheKey("overview", "org", target)
			if err != nil {
				t.Fatal(err)
			}
			count := float64(10)
			svc.writeCache(t.Context(), key, Overview{Summary: Summary{Backlinks: &count, ReferringDomains: &count}})
			for _, size := range []int{50, 100} {
				page := pageInput{Page: 1, PageSize: size, Sort: "rank,desc", Mode: "as_is"}
				if kind == "rows" {
					got, err := svc.Rows(t.Context(), "org", lookupInput{Target: "example.com", Scope: string(ScopeDomain)}, page, rowsFilters{}, false)
					if err != nil || len(got.Rows) != size {
						t.Fatalf("Rows=%d, %v", len(got.Rows), err)
					}
				} else {
					page.Sort = "backlinks,desc"
					got, err := svc.Domains(t.Context(), "org", lookupInput{Target: "example.com", Scope: string(ScopeDomain)}, page, domainsFilters{})
					if err != nil || len(got.Rows) != size {
						t.Fatalf("Domains=%d, %v", len(got.Rows), err)
					}
				}
			}
			if calls.Load() != 1 {
				t.Fatalf("provider calls=%d, want one preserved first block despite stale overview", calls.Load())
			}
		})
	}
}

func TestConcurrentTableMissesShareOnePurchasePerOrganization(t *testing.T) {
	for _, kind := range []string{"rows", "domains", "pages", "whole-rows", "whole-domains"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var tasks []map[string]any
				if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil {
					t.Error(err)
				}
				time.Sleep(50 * time.Millisecond)
				providerOK(w, r.URL.Path, map[string]any{"items": []any{map[string]any{"domain": "source.example", "url_from": "https://source.example/", "page": "https://example.com/"}}, "total_count": 1})
			})
			svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
			if kind == "whole-rows" || kind == "whole-domains" {
				target, _ := normalizeTarget("example.com", string(ScopeDomain))
				for _, org := range []string{"org", "other-org"} {
					key, _ := cacheKey("overview", org, target)
					count := float64(1)
					svc.writeCache(t.Context(), key, Overview{Summary: Summary{Backlinks: &count, ReferringDomains: &count}})
				}
			}
			start := make(chan struct{})
			var workers sync.WaitGroup
			for i := range 8 {
				workers.Go(func() {
					<-start
					org := "org"
					if i%2 == 1 {
						org = "other-org"
					}
					in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
					page := pageInput{Page: 1, PageSize: 50, Sort: "rank,desc", Mode: "as_is"}
					var err error
					switch kind {
					case "rows", "whole-rows":
						_, err = svc.Rows(context.Background(), org, in, page, rowsFilters{}, false)
					case "domains", "whole-domains":
						_, err = svc.Domains(context.Background(), org, in, page, domainsFilters{})
					case "pages":
						_, err = svc.Pages(context.Background(), org, in, page, pagesFilters{})
					}
					if err != nil {
						t.Error(err)
					}
				})
			}
			close(start)
			workers.Wait()
			if calls.Load() != 2 {
				t.Fatalf("provider calls=%d, want one per organization", calls.Load())
			}
		})
	}
}
