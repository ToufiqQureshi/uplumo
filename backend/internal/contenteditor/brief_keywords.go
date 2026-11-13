package contenteditor

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/market"
)

// BriefKeyword preserves missing metrics as null, not a fabricated zero.
type BriefKeyword struct {
	Keyword      string `json:"keyword"`
	SearchVolume *int   `json:"searchVolume"`
	Difficulty   *int   `json:"difficulty"`
	Intent       string `json:"intent"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

func fetchBriefKeywords(ctx context.Context, request dataforseo.DoBudgetedRequest, input BriefInput) ([]BriefKeyword, string) {
	if !market.IsLabsLocationCode(input.LocationCode) || !market.IsLanguageServedForLocation(input.LocationCode, input.LanguageCode) {
		return nil, "unsupported_market"
	}
	if ctx.Err() != nil {
		return nil, "unavailable"
	}
	body, _ := json.Marshal([]map[string]any{{
		"keyword": input.Keyword, "location_code": input.LocationCode,
		"language_code": input.LanguageCode, "limit": billing.ContentBriefKeywordLimit,
		"include_seed_keyword": true, "include_clickstream_data": false,
		"exact_match": false, "ignore_synonyms": false,
	}})
	payload, err := request(http.MethodPost, "/v3/dataforseo_labs/google/keyword_suggestions/live", body, false)
	if err != nil {
		return nil, "unavailable"
	}
	result, err := firstTaskResult(payload)
	if err != nil {
		return nil, "unavailable"
	}
	type item struct {
		Keyword string `json:"keyword"`
		Info    struct {
			Volume  *int   `json:"search_volume"`
			Updated string `json:"last_updated_time"`
		} `json:"keyword_info"`
		Properties struct {
			Difficulty    *int `json:"keyword_difficulty"`
			OtherLanguage bool `json:"is_another_language"`
		} `json:"keyword_properties"`
		Intent struct {
			Main string `json:"main_intent"`
		} `json:"search_intent_info"`
	}
	var wire struct {
		Items []item `json:"items"`
		Seed  *item  `json:"seed_keyword_data"`
	}
	if json.Unmarshal(result, &wire) != nil {
		return nil, "unavailable"
	}
	if wire.Seed != nil {
		wire.Items = append([]item{*wire.Seed}, wire.Items...)
	}
	var out []BriefKeyword
	seen := map[string]bool{}
	for _, row := range wire.Items {
		key := strings.ToLower(strings.Join(strings.Fields(row.Keyword), " "))
		if key == "" || seen[key] || row.Properties.OtherLanguage {
			continue
		}
		seen[key] = true
		if row.Info.Volume != nil && *row.Info.Volume < 0 {
			row.Info.Volume = nil
		}
		if row.Properties.Difficulty != nil && (*row.Properties.Difficulty < 0 || *row.Properties.Difficulty > 100) {
			row.Properties.Difficulty = nil
		}
		switch row.Intent.Main {
		case "informational", "commercial", "transactional", "navigational":
		default:
			row.Intent.Main = ""
		}
		out = append(out, BriefKeyword{row.Keyword, row.Info.Volume, row.Properties.Difficulty, row.Intent.Main, row.Info.Updated})
		if len(out) == billing.ContentBriefKeywordLimit+1 {
			break
		}
	}
	if len(out) == 0 {
		return out, "no_data"
	}
	return out, "available"
}
