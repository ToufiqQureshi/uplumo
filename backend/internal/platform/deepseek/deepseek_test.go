package deepseek

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// offPeakTime is a Saturday: always off-peak.
var offPeakTime = time.Date(2026, 10, 24, 2, 0, 0, 0, time.UTC)

func serve(t *testing.T, status int, body string, inspect func(*http.Request, []byte)) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		if inspect != nil {
			inspect(r, payload)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return &Client{APIKey: "test-key", BaseURL: server.URL, Now: func() time.Time { return offPeakTime }}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func TestStreamAssemblesTextReasoningAndPricesUsage(t *testing.T) {
	body := strings.Join([]string{
		": keep-alive",
		`data: {"choices":[{"delta":{"reasoning_content":"think "}}]}`,
		`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo ✓"},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":1000,"completion_tokens":400,"prompt_cache_hit_tokens":600,"prompt_cache_miss_tokens":400}}`,
		"data: [DONE]",
		"",
	}, "\n")
	var seenAuth, seenBody string
	client := serve(t, 200, body, func(r *http.Request, payload []byte) {
		seenAuth = r.Header.Get("Authorization")
		seenBody = string(payload)
	})

	var text, reasoning []string
	result, err := client.Stream(context.Background(), Request{
		Model:           "deepseek-flash",
		Messages:        []Message{{Role: "user", Content: "hi"}},
		MaxTokens:       100,
		ReasoningEffort: "low",
	}, Handler{
		OnText:      func(d string) error { text = append(text, d); return nil },
		OnReasoning: func(d string) error { reasoning = append(reasoning, d); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "Hello ✓" || strings.Join(text, "|") != "Hel|lo ✓" {
		t.Fatalf("text = %q deltas = %v", result.Text, text)
	}
	if strings.Join(reasoning, "") != "think " {
		t.Fatalf("reasoning = %v", reasoning)
	}
	// Off-peak flash: 600 cache hits, 400 misses, 400 output tokens.
	want := 600*0.003/1e6 + 400*0.15/1e6 + 400*0.60/1e6
	if result.FinishReason != "stop" || !near(result.Usage.CostUSD, want) || result.Usage.PromptTokens != 1000 {
		t.Fatalf("result = %+v, want cost %v", result, want)
	}
	if seenAuth != "Bearer test-key" {
		t.Fatalf("auth header = %q", seenAuth)
	}
	for _, want := range []string{`"stream":true`, `"stream_options":{"include_usage":true}`, `"thinking":{"type":"enabled"}`, `"reasoning_effort":"low"`, `"max_tokens":100`, `"model":"deepseek-flash"`} {
		if !strings.Contains(seenBody, want) {
			t.Errorf("request body missing %s: %s", want, seenBody)
		}
	}
}

func TestStreamDisablesThinkingByDefault(t *testing.T) {
	for _, effort := range []string{"", "none"} {
		var seen string
		client := serve(t, 200, "data: [DONE]\n", func(_ *http.Request, payload []byte) { seen = string(payload) })
		if _, err := client.Stream(context.Background(), Request{Model: "deepseek-flash", ReasoningEffort: effort}, Handler{}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(seen, `"thinking":{"type":"disabled"}`) || strings.Contains(seen, "reasoning_effort") {
			t.Fatalf("effort %q sent %s", effort, seen)
		}
	}
}

func TestStreamSendsAnOpaqueCacheScope(t *testing.T) {
	var sent []string
	client := serve(t, 200, "data: [DONE]\n", func(_ *http.Request, payload []byte) {
		var body struct {
			UserID string `json:"user_id"`
		}
		_ = json.Unmarshal(payload, &body)
		sent = append(sent, body.UserID)
	})
	for _, scope := range []string{"org-1", "org-1", "org-2", ""} {
		if _, err := client.Stream(context.Background(), Request{Model: "deepseek-flash", CacheScope: scope}, Handler{}); err != nil {
			t.Fatal(err)
		}
	}
	if sent[0] == "" || sent[0] != sent[1] || sent[0] == sent[2] || sent[3] != "" {
		t.Fatalf("cache scopes = %q: same org must match, orgs must differ, none must be empty", sent)
	}
	if strings.Contains(sent[0], "org") || len(sent[0]) != 32 {
		t.Fatalf("cache scope %q leaks the id or has the wrong shape", sent[0])
	}
}

func TestStreamPricesPeakWhenStartOrEndIsPeak(t *testing.T) {
	body := `data: {"choices":[{"delta":{},"finish_reason":"stop","index":0}],"usage":{"prompt_tokens":1000,"completion_tokens":1000}}` + "\ndata: [DONE]\n"
	offPeakCost := 1000*0.15/1e6 + 1000*0.60/1e6
	for _, tc := range []struct {
		name       string
		start, end time.Time
		want       float64
	}{
		{"off-peak", offPeakTime, offPeakTime, offPeakCost},
		{"starts in peak", time.Date(2026, 10, 23, 9, 59, 0, 0, time.UTC), time.Date(2026, 10, 23, 10, 1, 0, 0, time.UTC), 2 * offPeakCost},
		{"ends in peak", time.Date(2026, 10, 23, 5, 59, 0, 0, time.UTC), time.Date(2026, 10, 23, 6, 0, 0, 0, time.UTC), 2 * offPeakCost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := serve(t, 200, body, nil)
			times := []time.Time{tc.start, tc.end}
			client.Now = func() time.Time { now := times[0]; times = times[1:]; return now }
			result, err := client.Stream(context.Background(), Request{Model: "deepseek-flash"}, Handler{})
			if err != nil {
				t.Fatal(err)
			}
			if !near(result.Usage.CostUSD, tc.want) {
				t.Fatalf("cost = %v, want %v", result.Usage.CostUSD, tc.want)
			}
		})
	}
}

func TestIsPeak(t *testing.T) {
	for _, tc := range []struct {
		at   time.Time
		want bool
	}{
		{time.Date(2026, 10, 23, 0, 59, 0, 0, time.UTC), false},
		{time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 10, 23, 3, 59, 0, 0, time.UTC), true},
		{time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 10, 23, 6, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 10, 23, 9, 59, 0, 0, time.UTC), true},
		{time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC), false},
		{time.Date(2026, 10, 24, 7, 0, 0, 0, time.UTC), false}, // Saturday
		{time.Date(2026, 10, 25, 7, 0, 0, 0, time.UTC), false}, // Sunday
		// 07:00 in Kolkata is 01:30 UTC: the zone must not matter.
		{time.Date(2026, 10, 23, 7, 0, 0, 0, time.FixedZone("IST", 5*3600+1800)), true},
	} {
		if got := IsPeak(tc.at); got != tc.want {
			t.Errorf("IsPeak(%s) = %v, want %v", tc.at, got, tc.want)
		}
	}
}

func TestPriceUsage(t *testing.T) {
	flash := priceUsage("deepseek-flash", Usage{PromptTokens: 2000, PromptCacheHitTokens: 1500, PromptCacheMissTokens: 500, CompletionTokens: 100}, false)
	if want := 1500*0.003/1e6 + 500*0.15/1e6 + 100*0.60/1e6; !near(flash, want) {
		t.Fatalf("flash = %v, want %v", flash, want)
	}
	// Only prompt_tokens reported: all of it is priced as cache misses.
	if got, want := priceUsage("deepseek-flash", Usage{PromptTokens: 1000}, true), 1000*0.30/1e6; !near(got, want) {
		t.Fatalf("misses only = %v, want %v", got, want)
	}
	// A model missing from the rate card is priced as the most expensive.
	if got, want := priceUsage("deepseek-typo", Usage{CompletionTokens: 1000}, false), priceUsage("deepseek-v4-pro", Usage{CompletionTokens: 1000}, false); !near(got, want) || want == 0 {
		t.Fatalf("unknown model = %v, want %v", got, want)
	}
	if got := priceUsage("deepseek-flash", Usage{}, true); got != 0 {
		t.Fatalf("no usage = %v", got)
	}
}

func TestStreamAssemblesToolCallsAcrossChunks(t *testing.T) {
	body := strings.Join([]string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"b","function":{"name":"second","arguments":"{\"q\":"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"a","function":{"name":"first","arguments":"{}"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"\"x\"}"}}]},"finish_reason":"tool_calls"}]}`,
		"data: [DONE]",
		"",
	}, "\n")
	client := serve(t, 200, body, nil)
	result, err := client.Stream(context.Background(), Request{Model: "m"}, Handler{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("tool calls = %+v", result.ToolCalls)
	}
	first, second := result.ToolCalls[0], result.ToolCalls[1]
	if first.ID != "a" || first.Function.Name != "first" || first.Type != "function" {
		t.Errorf("first = %+v", first)
	}
	if second.ID != "b" || second.Function.Arguments != `{"q":"x"}` {
		t.Errorf("second = %+v (arguments must join in order)", second)
	}
	if !json.Valid([]byte(second.Function.Arguments)) {
		t.Errorf("arguments are not valid JSON: %s", second.Function.Arguments)
	}
}

func TestStreamSendsToolsWithDefaultSchema(t *testing.T) {
	var seen string
	client := serve(t, 200, "data: [DONE]\n", func(_ *http.Request, payload []byte) { seen = string(payload) })
	_, err := client.Stream(context.Background(), Request{Model: "m", Tools: []Tool{{Name: "ping", Description: "d"}}}, Handler{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(seen, `"name":"ping"`) || !strings.Contains(seen, `"parameters":{"type":"object","properties":{}}`) {
		t.Fatalf("request = %s", seen)
	}
}

func TestStreamReturnsStatusErrorWithBoundedBody(t *testing.T) {
	client := serve(t, 429, strings.Repeat("x", maxErrorBody*3), nil)
	_, err := client.Stream(context.Background(), Request{Model: "m"}, Handler{})
	var status *StatusError
	if !errors.As(err, &status) || status.StatusCode != 429 {
		t.Fatalf("err = %v", err)
	}
	if len(status.Body) != maxErrorBody {
		t.Fatalf("body length = %d, want %d", len(status.Body), maxErrorBody)
	}
}

func TestStreamMidStreamErrorKeepsPartialResult(t *testing.T) {
	body := `data: {"choices":[{"delta":{"content":"par"}}]}` + "\n" +
		`data: {"error":{"message":"upstream went away"}}` + "\n"
	client := serve(t, 200, body, nil)
	result, err := client.Stream(context.Background(), Request{Model: "m"}, Handler{})
	if !errors.Is(err, ErrStreamError) || !strings.Contains(err.Error(), "upstream went away") {
		t.Fatalf("err = %v", err)
	}
	if result.Text != "par" {
		t.Fatalf("partial text = %q", result.Text)
	}
}

func TestStreamRejectsMalformedChunk(t *testing.T) {
	client := serve(t, 200, "data: {not json}\n", nil)
	if _, err := client.Stream(context.Background(), Request{Model: "m"}, Handler{}); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestStreamStopsWhenHandlerFails(t *testing.T) {
	body := `data: {"choices":[{"delta":{"content":"a"}}]}` + "\n" + `data: {"choices":[{"delta":{"content":"b"}}]}` + "\n"
	client := serve(t, 200, body, nil)
	boom := errors.New("client went away")
	calls := 0
	result, err := client.Stream(context.Background(), Request{Model: "m"}, Handler{OnText: func(string) error {
		calls++
		return boom
	}})
	if !errors.Is(err, boom) || calls != 1 {
		t.Fatalf("err = %v calls = %d", err, calls)
	}
	if result.Text != "a" {
		t.Fatalf("partial text = %q", result.Text)
	}
}

func TestStreamHonorsContextCancellation(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-release
	}))
	t.Cleanup(func() { close(release); server.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	client := &Client{APIKey: "k", BaseURL: server.URL}
	if _, err := client.Stream(ctx, Request{Model: "m"}, Handler{}); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestStreamRejectsOversizedLine(t *testing.T) {
	client := serve(t, 200, "data: "+strings.Repeat("x", maxLine+10)+"\n", nil)
	if _, err := client.Stream(context.Background(), Request{Model: "m"}, Handler{}); err == nil {
		t.Fatal("expected scanner error for oversized line")
	}
}

func TestBillableUSDNeverMakesAnAnswerFree(t *testing.T) {
	// The fallback is the model's peak rate; for a model missing from the rate
	// card, deepseek-v4-pro at peak, $1.32/M input and $3.96/M output.
	in, out := 1.32/1e6, 3.96/1e6
	for _, tc := range []struct {
		name          string
		model         string
		usage         Usage
		chars         int
		want          float64
		wantEstimated bool
	}{
		{"priced cost wins", "deepseek-flash", Usage{PromptTokens: 1000, CompletionTokens: 1000, CostUSD: 0.002}, 50, 0.002, false},
		{"no cost, tokens known, flash", "deepseek-flash", Usage{PromptTokens: 1_000_000, CompletionTokens: 1_000_000}, 0, 0.30 + 1.20, true},
		{"no cost, tokens known, unknown model", "nope", Usage{PromptTokens: 1_000_000, CompletionTokens: 1_000_000}, 0, 1.32 + 3.96, true},
		{"no usage at all, text streamed", "nope", Usage{}, 300, 100 * out, true},
		{"nothing happened", "deepseek-flash", Usage{}, 0, 0, false},
		{"negative cost is ignored", "nope", Usage{CompletionTokens: 1000, CostUSD: -1}, 0, 1000 * out, true},
		{"infinite cost is ignored", "nope", Usage{CompletionTokens: 1000, CostUSD: math.Inf(1)}, 0, 1000 * out, true},
		{"input only", "nope", Usage{PromptTokens: 1000}, 0, 1000 * in, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, estimated := tc.usage.BillableUSD(tc.model, tc.chars)
			if !near(got, tc.want) || estimated != tc.wantEstimated {
				t.Fatalf("BillableUSD(%d) = %v, %v; want %v, %v", tc.chars, got, estimated, tc.want, tc.wantEstimated)
			}
		})
	}
}

func TestMaxCostUSD(t *testing.T) {
	if _, err := MaxCostUSD(Request{Model: "deepseek-flash", Messages: []Message{{Content: "hi"}}}); err == nil {
		t.Fatal("a request without MaxTokens must have no ceiling")
	}
	got, err := MaxCostUSD(Request{
		Model:     "deepseek-flash",
		Messages:  []Message{{Role: "user", Content: strings.Repeat("a", 3000)}},
		Tools:     []Tool{{Name: "t", Description: strings.Repeat("d", 2999)}},
		MaxTokens: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 6,000 bytes are 2,000 input tokens at flash's peak $0.30/M plus 1,000
	// output at $1.20/M.
	if want := 2000*0.30/1e6 + 1000*1.20/1e6; !near(got, want) {
		t.Fatalf("MaxCostUSD = %v, want %v", got, want)
	}
	unknown, _ := MaxCostUSD(Request{Model: "nope", Messages: []Message{{Content: "hi"}}, MaxTokens: 1000})
	pro, _ := MaxCostUSD(Request{Model: "deepseek-v4-pro", Messages: []Message{{Content: "hi"}}, MaxTokens: 1000})
	if !near(unknown, pro) {
		t.Fatalf("unknown model ceiling %v, want the most expensive %v", unknown, pro)
	}
	withCall, _ := MaxCostUSD(Request{Model: "deepseek-flash", Messages: []Message{{ToolCalls: []ToolCall{{Function: FunctionCall{Arguments: strings.Repeat("x", 300)}}}}}, MaxTokens: 1})
	if withCall < 100*0.30/1e6 {
		t.Fatalf("tool-call arguments must count as input: %v", withCall)
	}
}
