package ranktracking

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// MaxPixelKeywords bounds one pixel check: each keyword is one live SERP task
// at twice the normal price (DataForSEO bills rectangles as one more page).
const (
	MaxPixelKeywords = 10
	pixelTTL         = 24 * time.Hour
	pixelParallel    = 4
)

// PixelBlock is one element of the results page and where it sits, in CSS
// pixels from the top of the page.
type PixelBlock struct {
	Type   string `json:"type"`
	Y      int    `json:"y"`
	Height int    `json:"height"`
	Rank   int    `json:"rank,omitempty"`
	Ours   bool   `json:"ours,omitempty"`
}

// PixelKeyword is one keyword's page: the customer's organic position, how
// far down it starts, and the blocks to draw the page.
type PixelKeyword struct {
	Keyword  string       `json:"keyword"`
	Position *int         `json:"position"`
	Y        *int         `json:"y"`
	Blocks   []PixelBlock `json:"blocks"`
	// Error is set when this keyword's lookup failed; the others still count.
	Error string `json:"error,omitempty"`
}

// PixelProvider runs a top-10 SERP with pixel rectangles.
type PixelProvider interface {
	Pixels(ctx context.Context, organizationID string, req CheckRequest) (PixelKeyword, error)
}

// PixelCache keeps a bought page for a day, so repeating a check is free.
type PixelCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// PixelInput asks for some of a config's keywords on one device.
type PixelInput struct {
	OrganizationID string
	ProjectID      string
	ConfigID       string
	Keywords       []string
	Device         string
}

// Pixels checks where the customer's result sits on screen for up to
// MaxPixelKeywords of a config's keywords. Only keywords tracked in the config
// are accepted, the config's validated market is used (an invalid one would be
// billed as a failed task), and each page is cached for a day.
func (c *Checks) Pixels(ctx context.Context, in PixelInput) ([]PixelKeyword, error) {
	if c.Pixel == nil {
		return nil, ErrChecksUnavailable
	}
	if c.Plans != nil {
		paid, err := c.Plans.HasPaidPlan(ctx, in.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("check pixel plan: %w", err)
		}
		if !paid {
			return nil, ErrPaymentRequired
		}
	}
	if len(in.Keywords) == 0 || len(in.Keywords) > MaxPixelKeywords {
		return nil, ValidationError(fmt.Sprintf("Choose 1 to %d keywords.", MaxPixelKeywords))
	}
	cfg, err := c.Repo.GetConfig(ctx, in.ProjectID, in.ConfigID)
	if err != nil {
		return nil, err
	}
	if in.Device != "desktop" && in.Device != "mobile" {
		return nil, ValidationError("Choose desktop or mobile.")
	}
	if cfg.Devices != "both" && cfg.Devices != in.Device {
		return nil, ValidationError("This tracker does not check that device.")
	}
	tracked, err := c.Repo.Keywords(ctx, cfg.ID)
	if err != nil {
		return nil, fmt.Errorf("load tracked keywords: %w", err)
	}
	var keywords []string
	for _, k := range in.Keywords {
		i := slices.IndexFunc(tracked, func(t Keyword) bool { return strings.EqualFold(t.Keyword, strings.TrimSpace(k)) })
		if i < 0 {
			return nil, ValidationError(fmt.Sprintf("%q is not tracked here.", k))
		}
		if !slices.Contains(keywords, tracked[i].Keyword) {
			keywords = append(keywords, tracked[i].Keyword)
		}
	}

	out := make([]PixelKeyword, len(keywords))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(pixelParallel)
	for i, kw := range keywords {
		g.Go(func() error {
			req := CheckRequest{Keyword: kw, Device: in.Device, LocationCode: cfg.LocationCode, LanguageCode: cfg.LanguageCode, LocationName: cfg.LocationName, TargetDomain: cfg.Domain, Depth: 10}
			key := pixelKey(in.OrganizationID, cfg, req)
			if c.PixelCache != nil {
				if raw, err := c.PixelCache.Get(gctx, key); err == nil && json.Unmarshal(raw, &out[i]) == nil {
					return nil
				}
			}
			got, err := c.Pixel.Pixels(gctx, in.OrganizationID, req)
			if err != nil {
				// One failed keyword is reported on its row; the others were
				// bought and are kept. Out of credits stops the whole check.
				if errors.Is(err, spend.ErrNoCredits) {
					return err
				}
				out[i] = PixelKeyword{Keyword: kw, Blocks: []PixelBlock{}, Error: "This keyword could not be checked. Try again later."}
				return nil
			}
			got.Keyword = kw
			out[i] = got
			if c.PixelCache != nil {
				if raw, err := json.Marshal(got); err == nil {
					_ = c.PixelCache.Set(context.WithoutCancel(gctx), key, raw, pixelTTL)
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func pixelKey(org string, cfg Config, req CheckRequest) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{cfg.ID, cfg.Domain, req.Device, strings.ToLower(req.Keyword)}, "\n")))
	return "rank-pixels:" + org + ":" + hex.EncodeToString(sum[:16])
}

// Pixels asks for the top 10 with rectangles. It does not stop the crawl at
// the target: the blocks above it are the point.
func (p DataForSEOSerp) Pixels(ctx context.Context, organizationID string, req CheckRequest) (PixelKeyword, error) {
	task := map[string]any{
		"keyword":              req.Keyword,
		"language_code":        req.LanguageCode,
		"device":               req.Device,
		"os":                   map[string]string{"desktop": "windows", "mobile": "android"}[req.Device],
		"depth":                10,
		"calculate_rectangles": true,
	}
	if req.LocationName != nil && *req.LocationName != "" {
		task["location_name"] = *req.LocationName
	} else {
		task["location_code"] = req.LocationCode
	}
	body, err := json.Marshal([]map[string]any{task})
	if err != nil {
		return PixelKeyword{}, fmt.Errorf("encode pixel request: %w", err)
	}
	response, err := p.Client.Do(ctx, organizationID, spend.RankTracking, http.MethodPost, serpLivePath, body, false)
	if err != nil {
		return PixelKeyword{}, fmt.Errorf("DataForSEO %s: %w", serpLivePath, err)
	}
	results, err := dataforseo.Results(response)
	if taskErr, ok := errors.AsType[*dataforseo.TaskError](err); ok && taskErr.StatusCode == noResultsStatus {
		return PixelKeyword{Blocks: []PixelBlock{}}, nil
	}
	if err != nil {
		return PixelKeyword{}, fmt.Errorf("DataForSEO %s: %w", serpLivePath, err)
	}
	return parsePixels(results, req.TargetDomain)
}

func parsePixels(results []json.RawMessage, targetDomain string) (PixelKeyword, error) {
	out := PixelKeyword{Blocks: []PixelBlock{}}
	if len(results) == 0 {
		return out, nil
	}
	var result struct {
		Items []struct {
			Type      string  `json:"type"`
			RankGroup *int    `json:"rank_group"`
			Domain    *string `json:"domain"`
			Rectangle *struct {
				Y      float64 `json:"y"`
				Height float64 `json:"height"`
			} `json:"rectangle"`
		} `json:"items"`
	}
	if err := json.Unmarshal(results[0], &result); err != nil {
		return PixelKeyword{}, fmt.Errorf("decode pixel SERP: %w", err)
	}
	target := strings.ToLower(targetDomain)
	for _, item := range result.Items {
		if item.Rectangle == nil || item.Rectangle.Height <= 0 {
			continue
		}
		b := PixelBlock{Type: item.Type, Y: int(item.Rectangle.Y), Height: int(item.Rectangle.Height)}
		if item.Type == "organic" && item.RankGroup != nil {
			b.Rank = *item.RankGroup
			if item.Domain != nil && out.Y == nil {
				d := strings.ToLower(*item.Domain)
				if d == target || strings.HasSuffix(d, "."+target) {
					b.Ours = true
					y := b.Y
					out.Y, out.Position = &y, item.RankGroup
				}
			}
		}
		out.Blocks = append(out.Blocks, b)
	}
	slices.SortStableFunc(out.Blocks, func(a, b PixelBlock) int { return a.Y - b.Y })
	return out, nil
}

// RedisPixelCache is PixelCache on Redis.
type RedisPixelCache struct{ Client *redis.Client }

// Get reads a cached page.
func (c RedisPixelCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.Client.Get(ctx, key).Bytes()
}

// Set stores a page for ttl.
func (c RedisPixelCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.Client.Set(ctx, key, value, ttl).Err()
}
