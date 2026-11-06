// Package staff serves the staff console: business numbers (users, revenue,
// subscriptions, marketing, provider cost) and account actions (block,
// unblock, delete). Every route requires a verified STAFF_EMAILS account.
package staff

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

// liveStatuses are the Razorpay states that grant a paid plan (billing.proStatuses).
var liveStatuses = []string{"authenticated", "active", "pending"}

// Money is an amount in a currency's subunits (paise or cents).
type Money struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

// Count is a labelled tally.
type Count struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// DayCount is signups on one day.
type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// Overview is the headline numbers.
type Overview struct {
	Users              int        `json:"users"`
	VerifiedUsers      int        `json:"verifiedUsers"`
	BlockedUsers       int        `json:"blockedUsers"`
	NewToday           int        `json:"newToday"`
	New7Days           int        `json:"new7Days"`
	New30Days          int        `json:"new30Days"`
	PayingWorkspaces   int        `json:"payingWorkspaces"`
	PlanCounts         []Count    `json:"planCounts"`
	MRR                []Money    `json:"mrr"`
	CollectedThisMonth []Money    `json:"collectedThisMonth"`
	CollectedLastMonth []Money    `json:"collectedLastMonth"`
	CancelledLast30    int        `json:"cancelledLast30Days"`
	EndingNext7        int        `json:"endingNext7Days"`
	PaymentFailing     int        `json:"paymentFailing"`
	SignupsByDay       []DayCount `json:"signupsByDay"`
}

// Stats is reads the console numbers from Postgres.
type Stats struct{ DB *pgxpool.Pool }

// Overview counts users, paying workspaces, MRR and collected payments.
func (s Stats) Overview(ctx context.Context) (Overview, error) {
	var o Overview
	err := s.DB.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM "user"),
		(SELECT count(*) FROM "user" WHERE email_verified),
		(SELECT count(*) FROM go_user_blocks),
		(SELECT count(*) FROM "user" WHERE created_at >= date_trunc('day', now())),
		(SELECT count(*) FROM "user" WHERE created_at >= now() - interval '7 days'),
		(SELECT count(*) FROM "user" WHERE created_at >= now() - interval '30 days'),
		(SELECT count(*) FROM go_billing_subscriptions WHERE status = ANY($1)),
		(SELECT count(*) FROM go_billing_subscriptions WHERE status = 'cancelled' AND updated_at >= now() - interval '30 days'),
		(SELECT count(*) FROM go_billing_subscriptions WHERE status = ANY($1) AND cancel_at_period_end AND current_period_end <= now() + interval '7 days'),
		(SELECT count(*) FROM go_billing_subscriptions WHERE status IN ('halted','pending'))`, liveStatuses,
	).Scan(&o.Users, &o.VerifiedUsers, &o.BlockedUsers, &o.NewToday, &o.New7Days, &o.New30Days,
		&o.PayingWorkspaces, &o.CancelledLast30, &o.EndingNext7, &o.PaymentFailing)
	if err != nil {
		return o, fmt.Errorf("overview counts: %w", err)
	}
	if o.PlanCounts, err = counts(ctx, s.DB, `SELECT coalesce(CASE WHEN bs.status = ANY($1) THEN bs.plan END, 'free'), count(*)
		FROM organization o LEFT JOIN go_billing_subscriptions bs ON bs.organization_id = o.id GROUP BY 1 ORDER BY 2 DESC`, liveStatuses); err != nil {
		return o, err
	}
	// MRR is each active subscription at its catalog price in its own currency.
	rows, err := s.DB.Query(ctx, `SELECT plan, currency, count(*) FROM go_billing_subscriptions WHERE status = 'active' GROUP BY 1, 2`)
	if err != nil {
		return o, fmt.Errorf("mrr: %w", err)
	}
	mrr := map[string]int64{}
	for rows.Next() {
		var plan, currency string
		var n int64
		if err := rows.Scan(&plan, &currency, &n); err != nil {
			rows.Close()
			return o, fmt.Errorf("scan mrr: %w", err)
		}
		if d, ok := plans.Get(plan); ok {
			mrr[currency] += int64(d.Price(currency)) * 100 * n
		}
	}
	rows.Close()
	for _, c := range []string{plans.INR, plans.USD} {
		o.MRR = append(o.MRR, Money{Currency: c, Amount: mrr[c]})
	}
	if o.CollectedThisMonth, err = collected(ctx, s.DB, `paid_at >= date_trunc('month', now())`); err != nil {
		return o, err
	}
	if o.CollectedLastMonth, err = collected(ctx, s.DB, `paid_at >= date_trunc('month', now()) - interval '1 month' AND paid_at < date_trunc('month', now())`); err != nil {
		return o, err
	}
	o.SignupsByDay, err = pgxCollect(ctx, s.DB, `SELECT to_char(d, 'YYYY-MM-DD'), (SELECT count(*) FROM "user" WHERE created_at >= d AND created_at < d + interval '1 day')
		FROM generate_series(date_trunc('day', now()) - interval '29 days', date_trunc('day', now()), interval '1 day') d ORDER BY d`,
		func(r pgx.CollectableRow) (DayCount, error) {
			var d DayCount
			return d, r.Scan(&d.Day, &d.Count)
		})
	return o, err
}

// collected is net payments (after refunds) in each currency matching where.
func collected(ctx context.Context, db *pgxpool.Pool, where string) ([]Money, error) {
	out := []Money{}
	for _, c := range []string{plans.INR, plans.USD} {
		m := Money{Currency: c}
		if err := db.QueryRow(ctx, `SELECT coalesce(sum(amount - amount_refunded), 0) FROM go_billing_payments WHERE currency = $1 AND `+where, c).Scan(&m.Amount); err != nil {
			return nil, fmt.Errorf("collected %s: %w", c, err)
		}
		out = append(out, m)
	}
	return out, nil
}

func counts(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) ([]Count, error) {
	return pgxCollect(ctx, db, sql, func(r pgx.CollectableRow) (Count, error) {
		var c Count
		return c, r.Scan(&c.Label, &c.Count)
	}, args...)
}

func pgxCollect[T any](ctx context.Context, db *pgxpool.Pool, sql string, scan pgx.RowToFunc[T], args ...any) ([]T, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("staff query: %w", err)
	}
	out, err := pgx.CollectRows(rows, scan)
	if err != nil {
		return nil, fmt.Errorf("staff query: %w", err)
	}
	if out == nil {
		out = []T{}
	}
	return out, nil
}

// Subscription is one workspace's subscription as staff see it.
type Subscription struct {
	OrganizationID    string     `json:"organizationId"`
	Workspace         string     `json:"workspace"`
	OwnerEmail        string     `json:"ownerEmail"`
	Plan              string     `json:"plan"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	CurrentPeriodEnd  *time.Time `json:"currentPeriodEnd"`
	CancelAtPeriodEnd bool       `json:"cancelAtPeriodEnd"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	PaidTotal         int64      `json:"paidTotal"`
}

// Subscriptions lists every workspace that ever subscribed, soonest renewal first.
func (s Stats) Subscriptions(ctx context.Context) ([]Subscription, error) {
	return pgxCollect(ctx, s.DB, `SELECT bs.organization_id, coalesce(o.name, ''),
			coalesce((SELECT u.email FROM member m JOIN "user" u ON u.id = m.user_id WHERE m.organization_id = bs.organization_id AND m.role = 'owner' ORDER BY m.created_at LIMIT 1), ''),
			bs.plan, bs.currency, bs.status, bs.current_period_end, bs.cancel_at_period_end, bs.updated_at,
			coalesce((SELECT sum(p.amount - p.amount_refunded) FROM go_billing_payments p WHERE p.organization_id = bs.organization_id AND p.currency = bs.currency), 0)
		FROM go_billing_subscriptions bs LEFT JOIN organization o ON o.id = bs.organization_id
		ORDER BY (bs.status = ANY($1)) DESC, bs.current_period_end NULLS LAST, bs.updated_at DESC`,
		func(r pgx.CollectableRow) (Subscription, error) {
			var v Subscription
			return v, r.Scan(&v.OrganizationID, &v.Workspace, &v.OwnerEmail, &v.Plan, &v.Currency, &v.Status,
				&v.CurrentPeriodEnd, &v.CancelAtPeriodEnd, &v.UpdatedAt, &v.PaidTotal)
		}, liveStatuses)
}

// Marketing is onboarding answers and the signup funnel.
type Marketing struct {
	FoundVia  []Count `json:"foundVia"`
	Interests []Count `json:"interests"`
	WorkFor   []Count `json:"workFor"`
	Funnel    []Count `json:"funnel"`
}

// Marketing tallies onboarding answers and the signup-to-paying funnel.
func (s Stats) Marketing(ctx context.Context) (Marketing, error) {
	var m Marketing
	var err error
	if m.FoundVia, err = counts(ctx, s.DB, `SELECT coalesce(nullif(found_via, ''), 'Not answered'), count(*) FROM "user" u LEFT JOIN user_onboarding_answers a ON a.user_id = u.id GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return m, err
	}
	if m.WorkFor, err = counts(ctx, s.DB, `SELECT coalesce(nullif(work_for, ''), 'Not answered'), count(*) FROM "user" u LEFT JOIN user_onboarding_answers a ON a.user_id = u.id GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return m, err
	}
	// interested_features is a JSON array; rows that are not are skipped.
	if m.Interests, err = counts(ctx, s.DB, `SELECT f, count(*) FROM user_onboarding_answers a,
		LATERAL jsonb_array_elements_text(CASE WHEN a.interested_features ~ '^\s*\[' THEN a.interested_features::jsonb ELSE '[]'::jsonb END) f
		GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return m, err
	}
	var signed, verified, onboarded, withSite, withGoogle, paid int
	err = s.DB.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM "user"),
		(SELECT count(*) FROM "user" WHERE email_verified),
		(SELECT count(*) FROM user_onboarding_answers WHERE coalesce(completed_at, '') <> ''),
		(SELECT count(DISTINCT m.user_id) FROM member m JOIN projects p ON p.organization_id = m.organization_id AND p.archived_at IS NULL AND p.domain IS NOT NULL),
		(SELECT count(DISTINCT m.user_id) FROM member m WHERE EXISTS (SELECT 1 FROM gsc_connections g WHERE g.organization_id = m.organization_id) OR EXISTS (SELECT 1 FROM ga4_connections g WHERE g.organization_id = m.organization_id)),
		(SELECT count(DISTINCT m.user_id) FROM member m JOIN go_billing_subscriptions bs ON bs.organization_id = m.organization_id AND bs.status = ANY($1))`, liveStatuses,
	).Scan(&signed, &verified, &onboarded, &withSite, &withGoogle, &paid)
	if err != nil {
		return m, fmt.Errorf("funnel: %w", err)
	}
	m.Funnel = []Count{{"Signed up", signed}, {"Verified email", verified}, {"Finished onboarding", onboarded}, {"Added a website", withSite}, {"Connected Google", withGoogle}, {"Paying", paid}}
	return m, nil
}

// MonthMoney is one month of payments in one currency.
type MonthMoney struct {
	Month    string `json:"month"`
	Currency string `json:"currency"`
	Gross    int64  `json:"gross"`
	Refunded int64  `json:"refunded"`
	Payments int    `json:"payments"`
}

// MonthCost is one month of provider cost in US dollars.
type MonthCost struct {
	Month   string  `json:"month"`
	CostUSD float64 `json:"costUsd"`
}

// WorkspaceCost is a workspace's provider cost.
type WorkspaceCost struct {
	OrganizationID string  `json:"organizationId"`
	Workspace      string  `json:"workspace"`
	OwnerEmail     string  `json:"ownerEmail"`
	Plan           string  `json:"plan"`
	CostUSD        float64 `json:"costUsd"`
}

// Finance is revenue and provider cost.
type Finance struct {
	Revenue          []MonthMoney    `json:"revenue"`
	ProviderCost     []MonthCost     `json:"providerCost"`
	CostByFeature    []FeatureCost   `json:"costByFeature30Days"`
	TopCostWorkspace []WorkspaceCost `json:"topCostWorkspaces30Days"`
}

// FeatureCost is provider cost of one feature.
type FeatureCost struct {
	Feature string  `json:"feature"`
	CostUSD float64 `json:"costUsd"`
}

// Finance covers the last 12 months: real payments per currency and every
// recorded provider cost (go_provider_spend, in US dollars).
func (s Stats) Finance(ctx context.Context) (Finance, error) {
	var f Finance
	var err error
	if f.Revenue, err = pgxCollect(ctx, s.DB, `SELECT to_char(date_trunc('month', paid_at), 'YYYY-MM'), currency, sum(amount), sum(amount_refunded), count(*)
		FROM go_billing_payments WHERE paid_at >= date_trunc('month', now()) - interval '11 months' GROUP BY 1, 2 ORDER BY 1, 2`,
		func(r pgx.CollectableRow) (MonthMoney, error) {
			var v MonthMoney
			return v, r.Scan(&v.Month, &v.Currency, &v.Gross, &v.Refunded, &v.Payments)
		}); err != nil {
		return f, err
	}
	if f.ProviderCost, err = pgxCollect(ctx, s.DB, `SELECT to_char(date_trunc('month', created_at), 'YYYY-MM'), sum(cost_usd)::float8
		FROM go_provider_spend WHERE created_at >= date_trunc('month', now()) - interval '11 months' GROUP BY 1 ORDER BY 1`,
		func(r pgx.CollectableRow) (MonthCost, error) {
			var v MonthCost
			return v, r.Scan(&v.Month, &v.CostUSD)
		}); err != nil {
		return f, err
	}
	if f.CostByFeature, err = pgxCollect(ctx, s.DB, `SELECT feature, sum(cost_usd)::float8 FROM go_provider_spend WHERE created_at >= now() - interval '30 days' GROUP BY 1 ORDER BY 2 DESC`,
		func(r pgx.CollectableRow) (FeatureCost, error) {
			var v FeatureCost
			return v, r.Scan(&v.Feature, &v.CostUSD)
		}); err != nil {
		return f, err
	}
	f.TopCostWorkspace, err = pgxCollect(ctx, s.DB, `SELECT ps.organization_id, coalesce(o.name, ''),
			coalesce((SELECT u.email FROM member m JOIN "user" u ON u.id = m.user_id WHERE m.organization_id = ps.organization_id AND m.role = 'owner' ORDER BY m.created_at LIMIT 1), ''),
			coalesce((SELECT CASE WHEN bs.status = ANY($1) THEN bs.plan END FROM go_billing_subscriptions bs WHERE bs.organization_id = ps.organization_id), 'free'),
			sum(ps.cost_usd)::float8
		FROM go_provider_spend ps LEFT JOIN organization o ON o.id = ps.organization_id
		WHERE ps.created_at >= now() - interval '30 days' GROUP BY 1, 2 ORDER BY 5 DESC LIMIT 10`,
		func(r pgx.CollectableRow) (WorkspaceCost, error) {
			var v WorkspaceCost
			return v, r.Scan(&v.OrganizationID, &v.Workspace, &v.OwnerEmail, &v.Plan, &v.CostUSD)
		}, liveStatuses)
	return f, err
}
