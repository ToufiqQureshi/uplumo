package ranktracking

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

type pixelRepo struct {
	Repository
	cfg      Config
	keywords []Keyword
}

func (r pixelRepo) GetConfig(context.Context, string, string) (Config, error) { return r.cfg, nil }
func (r pixelRepo) Keywords(context.Context, string) ([]Keyword, error)       { return r.keywords, nil }

type pixelFake struct {
	calls atomic.Int32
	fail  map[string]error
}

func (f *pixelFake) Pixels(_ context.Context, _ string, req CheckRequest) (PixelKeyword, error) {
	f.calls.Add(1)
	if err := f.fail[req.Keyword]; err != nil {
		return PixelKeyword{}, err
	}
	y := 1200
	return PixelKeyword{Position: new(2), Y: &y, Blocks: []PixelBlock{{Type: "ai_overview", Y: 0, Height: 800}}}, nil
}

type pixelMemCache struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (c *pixelMemCache) Get(_ context.Context, k string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.m[k]; ok {
		return v, nil
	}
	return nil, errors.New("miss")
}
func (c *pixelMemCache) Set(_ context.Context, k string, v []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k] = v
	return nil
}

func TestPixelsOnlyBuysTrackedKeywordsOnceAndCachesThem(t *testing.T) {
	repo := pixelRepo{cfg: Config{ID: "cfg", Domain: "sunrise.in", Devices: "mobile", LocationCode: 2356, LanguageCode: "en"},
		keywords: []Keyword{{Keyword: "cafe near lake"}, {Keyword: "cold coffee"}}}
	fake := &pixelFake{}
	c := &Checks{Repo: repo, Pixel: fake, PixelCache: &pixelMemCache{m: map[string][]byte{}}, Plans: fakePlans{paid: true}}
	ctx := t.Context()
	in := func(device string, kws ...string) PixelInput {
		return PixelInput{OrganizationID: "org", ProjectID: "p", ConfigID: "cfg", Device: device, Keywords: kws}
	}
	for name, bad := range map[string]PixelInput{
		"untracked keyword": in("mobile", "pizza"),
		"device not tracked": in("desktop", "cold coffee"),
		"no keywords":        in("mobile"),
		"over the limit":     in("mobile", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"),
	} {
		if _, err := c.Pixels(ctx, bad); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := (&Checks{Repo: repo, Pixel: fake, Plans: fakePlans{paid: false}}).Pixels(ctx, in("mobile", "cold coffee")); !errors.Is(err, ErrPaymentRequired) {
		t.Errorf("free plan: err = %v", err)
	}
	if fake.calls.Load() != 0 {
		t.Fatalf("refused requests reached the provider %d times", fake.calls.Load())
	}

	got, err := c.Pixels(ctx, in("mobile", "Cold Coffee", "cold coffee", "cafe near lake"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || fake.calls.Load() != 2 || got[0].Keyword != "cold coffee" || *got[0].Y != 1200 {
		t.Fatalf("got %+v with %d calls; want one call per distinct tracked keyword", got, fake.calls.Load())
	}
	if _, err := c.Pixels(ctx, in("mobile", "cold coffee", "cafe near lake")); err != nil || fake.calls.Load() != 2 {
		t.Fatalf("repeat: err=%v calls=%d, want served from cache", err, fake.calls.Load())
	}

	fake.fail = map[string]error{"cafe near lake": errors.New("boom")}
	c.PixelCache = &pixelMemCache{m: map[string][]byte{}}
	got, err = c.Pixels(ctx, in("mobile", "cold coffee", "cafe near lake"))
	if err != nil || got[1].Error == "" || got[0].Error != "" {
		t.Fatalf("one failed keyword: %+v, %v; want the other kept", got, err)
	}
	fake.fail = map[string]error{"cold coffee": spend.ErrNoCredits}
	c.PixelCache = &pixelMemCache{m: map[string][]byte{}}
	if _, err := c.Pixels(ctx, in("mobile", "cold coffee")); !errors.Is(err, spend.ErrNoCredits) {
		t.Fatalf("out of credits: err = %v, want it surfaced", err)
	}
}

func TestParsePixelsFindsOurResultAndOrdersBlocks(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"items": []any{
		map[string]any{"type": "organic", "rank_group": 1, "domain": "rival.in", "rectangle": map[string]any{"y": 900.0, "height": 120.0}},
		map[string]any{"type": "ai_overview", "rectangle": map[string]any{"y": 200.0, "height": 600.0}},
		map[string]any{"type": "organic", "rank_group": 2, "domain": "www.sunrise.in", "rectangle": map[string]any{"y": 1030.0, "height": 120.0}},
		map[string]any{"type": "people_also_ask"},
	}})
	got, err := parsePixels([]json.RawMessage{raw}, "sunrise.in")
	if err != nil {
		t.Fatal(err)
	}
	if got.Y == nil || *got.Y != 1030 || *got.Position != 2 || len(got.Blocks) != 3 || got.Blocks[0].Type != "ai_overview" || !got.Blocks[2].Ours {
		t.Fatalf("parsed = %+v", got)
	}
}
