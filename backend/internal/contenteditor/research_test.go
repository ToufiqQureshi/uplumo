package contenteditor

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
)

func TestBriefKeywordsOneCallPreservesUnknownAndDeduplicates(t *testing.T) {
	calls := 0
	request := func(method, path string, body []byte, retry bool) (json.RawMessage, error) {
		calls++
		if retry || method != "POST" || !strings.HasSuffix(path, "/keyword_suggestions/live") {
			t.Fatal("unexpected request")
		}
		var tasks []map[string]any
		if err := json.Unmarshal(body, &tasks); err != nil {
			t.Fatal(err)
		}
		if tasks[0]["limit"] != float64(billing.ContentBriefKeywordLimit) || tasks[0]["include_seed_keyword"] != true || tasks[0]["include_clickstream_data"] != false {
			t.Fatalf("bad request %s", body)
		}
		return []byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"seed_keyword_data":{"keyword":"seo","keyword_info":{"search_volume":0},"search_intent_info":{"main_intent":"informational"}},"items":[{"keyword":"SEO"},{"keyword":"seo guide","keyword_properties":{"keyword_difficulty":22}}]}]}]}`), nil
	}
	rows, status := fetchBriefKeywords(context.Background(), request, BriefInput{Keyword: "seo", LocationCode: 2840, LanguageCode: "en"})
	if calls != 1 || status != "available" || len(rows) != 2 || rows[0].SearchVolume == nil || *rows[0].SearchVolume != 0 || rows[1].SearchVolume != nil || *rows[1].Difficulty != 22 {
		t.Fatalf("%+v %s calls=%d", rows, status, calls)
	}
}

func TestReviewKeyExcludesTransportKeyButIncludesEvidence(t *testing.T) {
	input := WriterInput{Action: WriterAnalyze, Content: "draft", Brief: "research", Mode: "seo", IdempotencyKey: "first"}
	first := analysisKey(input)
	input.IdempotencyKey = "second"
	if first != analysisKey(input) {
		t.Fatal("transport key bypasses cache")
	}
	input.Brief = "new research"
	if first == analysisKey(input) {
		t.Fatal("changed evidence reused")
	}
}

func TestReviewCacheCoalescesAndIsolatesWorkspaces(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool := contentTestPool(ctx, t)
	cache := PostgresAnalysisCache{DB: pool}
	suffix := time.Now().Format("150405.000000000")
	orgs := []string{"review-a-" + suffix, "review-b-" + suffix}
	for _, org := range orgs {
		if _, err := pool.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,$1,$1,now())`, org); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org) })
	}
	var calls atomic.Int32
	build := func() (WriterOutput, error) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		return WriterOutput{Content: "private findings"}, nil
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			out, err := cache.GetOrBuild(ctx, orgs[0], strings.Repeat("c", 64), build)
			if err != nil || out.Content != "private findings" {
				t.Errorf("%+v %v", out, err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("duplicate paid reviews: %d", calls.Load())
	}
	if _, err := cache.GetOrBuild(ctx, orgs[1], strings.Repeat("c", 64), build); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("private report leaked between workspaces")
	}
}

func TestConcurrentBriefRefreshUsesOneBuild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool := contentTestPool(ctx, t)
	cache := PostgresBriefCache{DB: pool}
	key := strings.Repeat("d", 64)
	_, _ = pool.Exec(ctx, `DELETE FROM go_content_brief_cache WHERE cache_key=$1`, key)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM go_content_brief_cache WHERE cache_key=$1`, key)
	})
	var calls atomic.Int32
	build := func() (ContentBrief, error) {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
		return ContentBrief{FetchedAt: time.Now().UTC()}, nil
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if _, err := cache.GetOrBuild(ctx, key, true, build); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("refresh builds=%d", calls.Load())
	}
}

type reviewMemoryCache struct {
	out   WriterOutput
	calls int
}

func (c *reviewMemoryCache) GetOrBuild(_ context.Context, _, _ string, build func() (WriterOutput, error)) (WriterOutput, error) {
	if c.out.Content != "" {
		return c.out, nil
	}
	c.calls++
	out, err := build()
	if err == nil {
		c.out = out
	}
	return out, err
}
func TestAnalyzeUsesCacheAndNeverRewrites(t *testing.T) {
	cache := &reviewMemoryCache{}
	llm := &fakeWriterLLM{result: openrouter.Result{Text: "Missing evidence in section two"}}
	agent := &WriterAgent{LLM: llm, Reviews: cache}
	input := WriterInput{Action: WriterAnalyze, Mode: "seo", Content: "Article text", IdempotencyKey: "one"}
	for range 2 {
		if _, err := agent.Generate(context.Background(), "org", input); err != nil {
			t.Fatal(err)
		}
	}
	if cache.calls != 1 {
		t.Fatal("review was regenerated")
	}
	if !strings.Contains(llm.request.Messages[0].Content, "not replacement article text") {
		t.Fatal("review prompt lost its read-only contract")
	}
}
