package spend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestScopedKeyHasNoKeyWithoutAnActionKey(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"no key attached": context.Background(),
		"empty key":       WithIdempotencyKey(context.Background(), ""),
	} {
		if got := ScopedKey(ctx, []byte("POST"), []byte("/path"), []byte("{}")); got != "" {
			t.Errorf("%s: ScopedKey() = %q, want none so background work is never deduplicated", name, got)
		}
	}
}

func TestScopedKeyIsStableForTheSameWorkAndDifferentForOtherWork(t *testing.T) {
	first := WithIdempotencyKey(context.Background(), "click-1")
	replay := WithIdempotencyKey(context.Background(), "click-1")
	other := WithIdempotencyKey(context.Background(), "click-2")

	a := ScopedKey(first, []byte("POST"), []byte("/summary"), []byte(`[{"t":"a.com"}]`))
	if got := ScopedKey(replay, []byte("POST"), []byte("/summary"), []byte(`[{"t":"a.com"}]`)); got != a {
		t.Errorf("a replay of the action got %q, want the same key %q", got, a)
	}
	for name, got := range map[string]string{
		"other path":         ScopedKey(first, []byte("POST"), []byte("/history"), []byte(`[{"t":"a.com"}]`)),
		"other body":         ScopedKey(first, []byte("POST"), []byte("/summary"), []byte(`[{"t":"b.com"}]`)),
		"other action":       ScopedKey(other, []byte("POST"), []byte("/summary"), []byte(`[{"t":"a.com"}]`)),
		"parts run together": ScopedKey(first, []byte("POST/summary"), []byte(`[{"t":"a.com"}]`)),
		"shifted boundary":   ScopedKey(first, []byte("POS"), []byte("T/summary"), []byte(`[{"t":"a.com"}]`)),
	} {
		if got == a || got == "" {
			t.Errorf("%s: key %q must differ from %q", name, got, a)
		}
	}
}

func TestScopedKeyNumbersRepeatedWorkAndStaysUnderTheHoldKeyLimit(t *testing.T) {
	ctx := WithIdempotencyKey(context.Background(), "click-1")
	sameWork := func(c context.Context) string { return ScopedKey(c, []byte("POST"), []byte("/summary"), []byte("{}")) }
	first, second := sameWork(ctx), sameWork(ctx)
	if first == second {
		t.Fatalf("the second identical call in one action got the first's key %q", first)
	}
	replay := WithIdempotencyKey(context.Background(), "click-1")
	if sameWork(replay) != first || sameWork(replay) != second {
		t.Error("a replay must get the same two keys in the same order")
	}
	// Derived contexts share the action's numbering.
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	if third := sameWork(child); third == first || third == second {
		t.Errorf("a derived context reused key %q", third)
	}
	// The hold rejects keys over 128 bytes, however long the browser's key is.
	long := make([]byte, 4096)
	for i := range long {
		long[i] = 'k'
	}
	if got := ScopedKey(WithIdempotencyKey(context.Background(), string(long)), long); len(got) > 128 {
		t.Errorf("key is %d bytes, over the hold limit of 128", len(got))
	}
}

func TestScopedKeyIsSafeForParallelLegs(t *testing.T) {
	ctx := WithIdempotencyKey(context.Background(), "click-1")
	keys := make([]string, 50)
	var wg sync.WaitGroup
	for i := range keys {
		wg.Go(func() { keys[i] = ScopedKey(ctx, []byte("POST"), []byte("/serp"), []byte("{}")) })
	}
	wg.Wait()
	seen := map[string]bool{}
	for _, key := range keys {
		if key == "" || seen[key] {
			t.Fatalf("parallel identical calls produced a missing or repeated key %q", key)
		}
		seen[key] = true
	}
}

func TestWriteHTTPErrorAnswersOnlyRefusals(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		status   int
		code     string
		answered bool
	}{
		{"no credits", fmt.Errorf("hold: %w", ErrNoCredits), http.StatusPaymentRequired, "insufficient_credits", true},
		{"paused", fmt.Errorf("hold: %w", ErrPaused), http.StatusServiceUnavailable, "usage_paused", true},
		{"workspace day used", fmt.Errorf("hold: %w", ErrWorkspaceDailyLimit), http.StatusTooManyRequests, "daily_credit_limit", true},
		{"duplicate request", fmt.Errorf("hold: %w", ErrIdempotentReplay), http.StatusConflict, "duplicate_request", true},
		{"anything else", errors.New("provider down"), 0, "", false},
		{"nil", nil, 0, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			if got := WriteHTTPError(recorder, tc.err); got != tc.answered {
				t.Fatalf("WriteHTTPError() = %v, want %v", got, tc.answered)
			}
			if Refused(tc.err) != tc.answered || (Message(tc.err) != "") != tc.answered {
				t.Fatalf("Refused/Message disagree with WriteHTTPError for %v", tc.err)
			}
			if !tc.answered {
				if recorder.Body.Len() != 0 {
					t.Fatalf("wrote %q for an error it does not own", recorder.Body.String())
				}
				return
			}
			var body struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("body %q: %v", recorder.Body.String(), err)
			}
			if recorder.Code != tc.status || body.Error.Code != tc.code || body.Error.Message != Message(tc.err) {
				t.Fatalf("got %d %+v", recorder.Code, body)
			}
		})
	}
}

func TestFeaturesListsEveryPaidFeatureOnce(t *testing.T) {
	seen := map[Feature]bool{}
	for _, f := range Features() {
		if f == "" || f == Reference || seen[f] {
			t.Fatalf("Features() has %q twice, empty or free", f)
		}
		seen[f] = true
	}
	if len(seen) != 9 {
		t.Fatalf("Features() has %d features, want 9", len(seen))
	}
}
