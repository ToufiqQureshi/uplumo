package localseo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/keywords"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

type fakeData struct {
	calls   atomic.Int32
	fail    bool
	billing bool
	// creditsFor, when not zero, is how many searches are paid for before the
	// credit guard refuses the rest; a negative value refuses every search.
	creditsFor int32
}

type fixedPlans struct {
	allowed bool
	err     error
}

func (f fixedPlans) HasPaidPlan(context.Context, string) (bool, error) { return f.allowed, f.err }

func TestLocalSEORoutesCheckPlanBeforeProvider(t *testing.T) {
	for _, tt := range []struct {
		plans  fixedPlans
		status int
	}{
		{fixedPlans{allowed: false}, http.StatusPaymentRequired},
		{fixedPlans{err: errors.New("database unavailable")}, http.StatusServiceUnavailable},
	} {
		provider := &fakeData{}
		mux := http.NewServeMux()
		Mount(mux, &Service{Data: provider, Plans: tt.plans}, func(h http.Handler) http.Handler { return h }, func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1")))
			})
		})
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/projects/p/local-seo/rank-grid", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tt.status || provider.calls.Load() != 0 {
			t.Fatalf("status=%d provider calls=%d, want %d and 0", w.Code, provider.calls.Load(), tt.status)
		}
	}
}

func (f *fakeData) LocalSERP(_ context.Context, org string, in keywords.LocalSERPInput) ([]map[string]any, error) {
	if org != "org-1" || in.Depth != 20 || in.SearchType != "maps" {
		return nil, errors.New("bad provider arguments")
	}
	n := f.calls.Add(1)
	if f.creditsFor != 0 && n > max(f.creditsFor, 0) {
		return nil, fmt.Errorf("%w: %w", dataforseo.ErrBillingIssue, spend.ErrNoCredits)
	}
	if f.billing {
		return nil, dataforseo.ErrBillingIssue
	}
	if f.fail {
		return nil, errors.New("provider unavailable")
	}
	return []map[string]any{{"title": "Harbor Dental", "cid": "target", "rank_absolute": float64(2)}}, nil
}
func (f *fakeData) BusinessListingsSearch(context.Context, string, keywords.BusinessListingsSearchInput) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakeData) BusinessInfo(context.Context, string, keywords.BusinessInfoInput) (map[string]any, error) {
	return map[string]any{"title": "Harbor Dental"}, nil
}

func TestRankGridBoundsWorkAndSummarizesMatches(t *testing.T) {
	provider := &fakeData{}
	s := &Service{Data: provider}
	body := `{"keyword":"dentist","target":{"cid":"target"},"center":{"latitude":19.07,"longitude":72.87},"gridSize":3,"spacingKm":1,"device":"mobile"}`
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(body))
	r = r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1"))
	w := httptest.NewRecorder()
	s.rankGrid(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Grid    []localPoint `json:"grid"`
		Summary struct {
			PointsSearched int      `json:"pointsSearched"`
			PointsFound    int      `json:"pointsFound"`
			AverageRank    *float64 `json:"averageRank"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != 9 || provider.calls.Load() != 9 || got.Summary.PointsSearched != 9 || got.Summary.PointsFound != 9 || got.Summary.AverageRank == nil || *got.Summary.AverageRank != 2 {
		t.Fatalf("unexpected grid result: %+v calls=%d", got, provider.calls.Load())
	}
	if got.Grid[0].Latitude <= got.Grid[6].Latitude {
		t.Fatal("grid rows should descend from north to south")
	}
}

func TestRankGridRejectsInvalidInputBeforeProviderCall(t *testing.T) {
	provider := &fakeData{}
	s := &Service{Data: provider}
	for _, body := range []string{
		`{"keyword":"dentist","target":{"cid":"target"},"center":{"latitude":91,"longitude":0}}`,
		`{"keyword":"dentist","target":{"cid":"target"},"center":{"longitude":0}}`,
		`{"keyword":"dentist","target":{"cid":"target"},"center":{"latitude":0,"longitude":0},"gridSize":4}`,
		`{"keyword":"dentist","target":{},"center":{"latitude":0,"longitude":0}}`,
	} {
		r := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(body))
		r = r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1"))
		w := httptest.NewRecorder()
		s.rankGrid(w, r)
		if w.Code != 400 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	if provider.calls.Load() != 0 {
		t.Fatalf("invalid requests reached paid provider %d times", provider.calls.Load())
	}
}

func TestRankGridAllProviderFailuresReturnGatewayError(t *testing.T) {
	provider := &fakeData{fail: true}
	s := &Service{Data: provider}
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(`{"keyword":"dentist","target":{"name":"Harbor Dental"},"center":{"latitude":0,"longitude":0},"gridSize":3}`))
	r = r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1"))
	w := httptest.NewRecorder()
	s.rankGrid(w, r)
	if w.Code != 502 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if provider.calls.Load() != 9 {
		t.Fatalf("calls=%d", provider.calls.Load())
	}
}

// Credits running out part-way must not throw away the searches already paid
// for: the finished points are shown, the rest are skipped and counted.
func TestRankGridKeepsPaidPointsWhenCreditsRunOutMidGrid(t *testing.T) {
	provider := &fakeData{creditsFor: 3}
	s := &Service{Data: provider}
	body := `{"keyword":"dentist","target":{"cid":"target"},"center":{"latitude":19.07,"longitude":72.87},"gridSize":3,"spacingKm":1,"device":"mobile"}`
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(body))
	r = r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1"))
	w := httptest.NewRecorder()
	s.rankGrid(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s, want the 3 paid points answered", w.Code, w.Body.String())
	}
	var got struct {
		Grid    []localPoint `json:"grid"`
		Summary struct {
			PointsFound   int    `json:"pointsFound"`
			FailedPoints  int    `json:"failedPoints"`
			StoppedReason string `json:"stoppedReason"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary.PointsFound != 3 || got.Summary.FailedPoints != 6 || len(got.Grid) != 9 {
		t.Fatalf("summary = %+v, want 3 found and 6 skipped of 9", got.Summary)
	}
	if got.Summary.StoppedReason != spend.Message(spend.ErrNoCredits) {
		t.Fatalf("stoppedReason = %q, want the customer's out-of-credits message", got.Summary.StoppedReason)
	}
	if calls := provider.calls.Load(); calls != 6 {
		t.Fatalf("provider calls = %d, want 6: three paid, three refused, and no search started after the refusal", calls)
	}
}

func TestRankGridWithNothingPaidStillReportsWhyCreditsRefused(t *testing.T) {
	provider := &fakeData{creditsFor: -1}
	s := &Service{Data: provider}
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(`{"keyword":"dentist","target":{"cid":"target"},"center":{"latitude":19.07,"longitude":72.87},"gridSize":3}`))
	r = r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1"))
	w := httptest.NewRecorder()
	s.rankGrid(w, r)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status=%d body=%s, want 402 when no point was paid", w.Code, w.Body.String())
	}
}

func TestRankGridStopsAfterProviderBillingFailure(t *testing.T) {
	provider := &fakeData{billing: true}
	s := &Service{Data: provider}
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(`{"keyword":"dentist","target":{"name":"Harbor Dental"},"center":{"latitude":0,"longitude":0},"gridSize":5}`))
	r = r.WithContext(auth.WithProjectOrganization(r.Context(), "org-1"))
	w := httptest.NewRecorder()
	s.rankGrid(w, r)
	if w.Code != 402 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if provider.calls.Load() > 3 {
		t.Fatalf("billing failure should stop subsequent grid batches; provider calls=%d", provider.calls.Load())
	}
}
