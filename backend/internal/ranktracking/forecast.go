package ranktracking

import (
	"context"
	"fmt"
	"time"
)

// ScheduledCredits estimates queued checks due before the current credit refill.
// It uses each saved keyword/device/depth, including advanced operator charges.
func (s Store) ScheduledCredits(ctx context.Context, org string, until time.Time) (int64, error) {
	rows, err := s.DB.Query(ctx, `SELECT c.devices,c.serp_depth,c.schedule_interval,c.next_check_at,
 COALESCE(array_agg(k.keyword) FILTER (WHERE k.id IS NOT NULL),'{}')
 FROM go_rank_tracking_configs c JOIN projects p ON p.id=c.project_id LEFT JOIN go_rank_tracking_keywords k ON k.config_id=c.id
 WHERE p.organization_id=$1 AND p.archived_at IS NULL AND c.is_active AND c.next_check_at<$2 GROUP BY c.id`, org, until)
	if err != nil {
		return 0, fmt.Errorf("load scheduled rank budget: %w", err)
	}
	defer rows.Close()
	var total int64
	for rows.Next() {
		var devices, interval string
		var depth int
		var next time.Time
		var keywords []string
		if err := rows.Scan(&devices, &depth, &interval, &next, &keywords); err != nil {
			return 0, err
		}
		if len(keywords) == 0 {
			continue
		}
		cost := int64(EstimateCheck(keywords, devices, depth, Queued).CostCredits)
		for n := 0; next.Before(until) && n < 366; n++ {
			total += cost
			switch interval {
			case "daily":
				next = next.AddDate(0, 0, 1)
			case "weekly":
				next = next.AddDate(0, 0, 7)
			case "monthly":
				anchor := next
				next = (Scheduler{Now: func() time.Time { return anchor }}).Next(Monthly, &anchor, nil)
			default:
				return 0, fmt.Errorf("unexpected scheduled interval %q", interval)
			}
		}
	}
	return total, rows.Err()
}
