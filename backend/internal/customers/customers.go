// Package customers lists every signed-up user with their onboarding answers,
// plan and site usage for Uplumo staff, as JSON or a CSV export.
package customers

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
)

// ErrForbidden means the caller is not verified staff.
var ErrForbidden = errors.New("staff only")

// Customer is one user as staff see them. Onboarding fields are empty until answered.
type Customer struct {
	UserID                string    `json:"userId"`
	Email                 string    `json:"email"`
	Name                  string    `json:"name"`
	EmailVerified         bool      `json:"emailVerified"`
	Blocked               bool      `json:"blocked"`
	SignedUpAt            time.Time `json:"signedUpAt"`
	Workspace             string    `json:"workspace"`
	Role                  string    `json:"role"`
	Plan                  string    `json:"plan"`
	SubscriptionStatus    string    `json:"subscriptionStatus"`
	ActiveSites           int       `json:"activeSites"`
	Domains               []string  `json:"domains"`
	GSCConnected          bool      `json:"gscConnected"`
	GA4Connected          bool      `json:"ga4Connected"`
	InterestedFeatures    string    `json:"interestedFeatures"`
	WorkFor               string    `json:"workFor"`
	ClientWebsiteCount    string    `json:"clientWebsiteCount"`
	FoundVia              string    `json:"foundVia"`
	MCPSetupIntent        string    `json:"mcpSetupIntent"`
	OnboardingCompletedAt string    `json:"onboardingCompletedAt"`
}

type Service struct {
	DB *pgxpool.Pool
	// StaffEmails is the server-configured list, normalized by config.Load.
	StaffEmails []string
}

// IsStaff reports whether user is a verified account on the staff list.
func (s *Service) IsStaff(ctx context.Context, user auth.User) (bool, error) {
	if user.ID == "" || len(s.StaffEmails) == 0 {
		return false, nil
	}
	var ok bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND email_verified AND lower(email)=ANY($2))`, user.ID, s.StaffEmails).Scan(&ok)
	return ok, err
}

// List returns every user, newest first. The workspace is the one the user
// owns (else the first they joined); the "Default" placeholder is not a site.
func (s *Service) List(ctx context.Context, user auth.User) ([]Customer, error) {
	staff, err := s.IsStaff(ctx, user)
	if err != nil {
		return nil, err
	}
	if !staff {
		return nil, ErrForbidden
	}
	rows, err := s.DB.Query(ctx, `
		SELECT u.id, u.email, coalesce(u.name,''), u.email_verified, EXISTS(SELECT 1 FROM go_user_blocks b WHERE b.user_id=u.id), u.created_at,
		       coalesce(o.name,''), coalesce(m.role,''),
		       coalesce(CASE WHEN bs.status IN ('authenticated','active','pending') THEN bs.plan END,'free'),
		       coalesce(bs.status,''),
		       coalesce(sites.n,0), coalesce(sites.domains,'{}'),
		       EXISTS(SELECT 1 FROM gsc_connections g WHERE g.organization_id=m.organization_id),
		       EXISTS(SELECT 1 FROM ga4_connections g WHERE g.organization_id=m.organization_id),
		       coalesce(a.interested_features,''), coalesce(a.work_for,''), coalesce(a.client_website_count,''),
		       coalesce(a.found_via,''), coalesce(a.mcp_setup_intent,''), coalesce(a.completed_at,'')
		FROM "user" u
		LEFT JOIN LATERAL (
			SELECT organization_id, role FROM member WHERE user_id=u.id
			ORDER BY (role='owner') DESC, created_at LIMIT 1
		) m ON true
		LEFT JOIN organization o ON o.id=m.organization_id
		LEFT JOIN go_billing_subscriptions bs ON bs.organization_id=m.organization_id
		LEFT JOIN LATERAL (
			SELECT count(*) AS n, array_agg(p.domain ORDER BY p.created_at) FILTER (WHERE p.domain IS NOT NULL) AS domains
			FROM projects p WHERE p.organization_id=m.organization_id AND p.archived_at IS NULL
			AND NOT (p.name='Default' AND p.domain IS NULL)
		) sites ON true
		LEFT JOIN user_onboarding_answers a ON a.user_id=u.id
		ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()
	out := []Customer{}
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.UserID, &c.Email, &c.Name, &c.EmailVerified, &c.Blocked, &c.SignedUpAt, &c.Workspace, &c.Role,
			&c.Plan, &c.SubscriptionStatus, &c.ActiveSites, &c.Domains, &c.GSCConnected, &c.GA4Connected,
			&c.InterestedFeatures, &c.WorkFor, &c.ClientWebsiteCount, &c.FoundVia, &c.MCPSetupIntent, &c.OnboardingCompletedAt); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var csvHeader = []string{"email", "name", "email_verified", "blocked", "signed_up_at", "workspace", "role", "plan", "subscription_status",
	"active_sites", "domains", "gsc_connected", "ga4_connected", "interested_features", "work_for",
	"client_website_count", "found_via", "mcp_setup_intent", "onboarding_completed_at"}

// WriteCSV writes customers for spreadsheets and marketing tools.
func WriteCSV(w io.Writer, list []Customer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, c := range list {
		row := []string{c.Email, c.Name, strconv.FormatBool(c.EmailVerified), strconv.FormatBool(c.Blocked), c.SignedUpAt.UTC().Format(time.RFC3339),
			c.Workspace, c.Role, c.Plan, c.SubscriptionStatus, strconv.Itoa(c.ActiveSites), strings.Join(c.Domains, " "),
			strconv.FormatBool(c.GSCConnected), strconv.FormatBool(c.GA4Connected), c.InterestedFeatures, c.WorkFor,
			c.ClientWebsiteCount, c.FoundVia, c.MCPSetupIntent, c.OnboardingCompletedAt}
		for i := range row {
			row[i] = safeCell(row[i])
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// safeCell stops user-typed text (names, "Other" answers) from running as a
// spreadsheet formula when the export is opened in Excel or Sheets.
func safeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
