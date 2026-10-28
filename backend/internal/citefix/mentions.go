package citefix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
)

func (s Store) mentions(ctx context.Context, organizationID, projectID string) (*MentionsSnapshot, error) {
	var snapshot MentionsSnapshot
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT m.fetched_at,m.result_json
		FROM go_citefix_ai_mentions m JOIN projects p ON p.id=m.project_id AND p.organization_id=$1
		WHERE m.project_id=$2 AND m.organization_id=$1`, organizationID, projectID).Scan(&snapshot.FetchedAt, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read saved AI mentions for Cite Fix: %w", err)
	}
	if err := json.Unmarshal(raw, &snapshot.Result); err != nil {
		return nil, fmt.Errorf("decode saved AI mentions for Cite Fix: %w", err)
	}
	if snapshot.Result.SourceSites == nil {
		snapshot.Result.SourceSites = []aisearch.SiteMentions{}
	}
	if snapshot.Result.TopPages == nil {
		snapshot.Result.TopPages = []aisearch.TopPage{}
	}
	if snapshot.Result.TopQueries == nil {
		snapshot.Result.TopQueries = []aisearch.TopQuery{}
	}
	if snapshot.Result.PerPlatform == nil {
		snapshot.Result.PerPlatform = []aisearch.PlatformBreakdown{}
	}
	if snapshot.Result.MonthlyVolume == nil {
		snapshot.Result.MonthlyVolume = []aisearch.MonthlyVolume{}
	}
	if snapshot.Result.RetrievedSites == nil {
		snapshot.Result.RetrievedSites = []aisearch.SiteMentions{}
	}
	if snapshot.Result.BrandsAlongside == nil {
		snapshot.Result.BrandsAlongside = []aisearch.BrandMention{}
	}
	return &snapshot, nil
}
