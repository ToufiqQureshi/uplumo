package citefix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// storeTimeout bounds one report's reads. A 10,000-page audit reads in well
// under a second; this only stops a stuck database from holding the request.
const storeTimeout = 45 * time.Second

// issueTypes are the stored audit issues the checks read.
var issueTypes = []string{
	string(audit.IssueAICrawlersBlocked), string(audit.IssueMissingLLMsTxt),
	string(audit.IssueMissingBusinessSchema), string(audit.IssueSeveralPhoneNumbers),
	string(audit.IssueJavaScriptRenderingSuspected), string(audit.IssueDuplicateContent),
}

// Store reads audits from Postgres. Every query joins the project to the
// calling workspace, so another workspace's audit reads as "no scan".
type Store struct {
	DB                  *pgxpool.Pool
	Audit               AuditResults
	GoogleAIOConfigured bool
	Guard               *audit.Guard
	HTTP                *http.Client
	Now                 func() time.Time
}

// Report builds the project's Cite Fix report from its latest completed audit.
func (s Store) Report(ctx context.Context, organizationID, projectID string) (Report, error) {
	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	running, err := s.running(ctx, organizationID, projectID)
	if err != nil {
		return Report{}, err
	}
	proof, err := s.proof(ctx, organizationID, projectID)
	if err != nil {
		return Report{}, err
	}
	mentions, err := s.mentions(ctx, organizationID, projectID)
	if err != nil {
		return Report{}, err
	}
	googleAIOverview, err := s.googleAIOverview(ctx, organizationID, projectID)
	if err != nil {
		return Report{}, err
	}
	var scan Scan
	err = s.DB.QueryRow(ctx, `SELECT a.id, a.start_url, a.completed_at FROM go_audits a
		JOIN projects p ON p.id = a.project_id AND p.organization_id = $1
		WHERE a.project_id = $2 AND a.status = 'completed' AND a.completed_at IS NOT NULL
		ORDER BY a.started_at DESC LIMIT 1`, organizationID, projectID).Scan(&scan.AuditID, &scan.StartURL, &scan.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		report := emptyReport(StateNoScan)
		report.Running, report.Proof, report.Mentions, report.GoogleAIOverview = running, proof, mentions, googleAIOverview
		report.Rivals, err = s.rivalComparisons(ctx, organizationID, projectID, proof, mentions, nil)
		if err != nil {
			return Report{}, err
		}
		return report, nil
	}
	if err != nil {
		return Report{}, fmt.Errorf("find latest completed audit: %w", err)
	}

	pages, legacy, err := s.pages(ctx, scan.AuditID)
	if err != nil {
		return Report{}, err
	}
	if legacy {
		report := emptyReport(StateNeedsRescan)
		report.Running, report.Scan, report.PagesTotal, report.Proof, report.Mentions, report.GoogleAIOverview = running, &scan, len(pages), proof, mentions, googleAIOverview
		report.Rivals, err = s.rivalComparisons(ctx, organizationID, projectID, proof, mentions, pages)
		if err != nil {
			return Report{}, err
		}
		report.SEOIssues, err = s.seoIssues(ctx, scan.AuditID, projectID)
		if err != nil {
			return Report{}, err
		}
		return report, nil
	}
	issues, err := s.issues(ctx, scan.AuditID)
	if err != nil {
		return Report{}, err
	}
	report := Build(scan, pages, issues)
	report.Running, report.Proof, report.Mentions, report.GoogleAIOverview = running, proof, mentions, googleAIOverview
	report.Rivals, err = s.rivalComparisons(ctx, organizationID, projectID, proof, mentions, pages)
	if err != nil {
		return Report{}, err
	}
	report.SEOIssues, err = s.seoIssues(ctx, scan.AuditID, projectID)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

// running returns the project's newest unfinished audit, if any.
func (s Store) running(ctx context.Context, organizationID, projectID string) (*Running, error) {
	var running Running
	err := s.DB.QueryRow(ctx, `SELECT a.id, a.status FROM go_audits a
		JOIN projects p ON p.id = a.project_id AND p.organization_id = $1
		WHERE a.project_id = $2 AND a.status = 'running'
		ORDER BY a.started_at DESC LIMIT 1`, organizationID, projectID).Scan(&running.AuditID, &running.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find running audit: %w", err)
	}
	return &running, nil
}

// pages loads an audit's pages. legacy is true when any page was stored
// without citation signals: the audit predates them.
func (s Store) pages(ctx context.Context, auditID string) ([]Page, bool, error) {
	rows, err := s.DB.Query(ctx, `SELECT url, COALESCE(title, ''), COALESCE(status_code, 0), is_indexable,
			word_count, external_link_count, citation_signals_json
		FROM go_audit_pages WHERE audit_id = $1 ORDER BY url LIMIT $2`, auditID, audit.PaidMaxAuditPages)
	if err != nil {
		return nil, false, fmt.Errorf("list audit pages: %w", err)
	}
	defer rows.Close()
	pages := []Page{}
	legacy := false
	for rows.Next() {
		var page Page
		var signalsJSON *string
		if err := rows.Scan(&page.URL, &page.Title, &page.StatusCode, &page.Indexable, &page.WordCount,
			&page.ExternalLinks, &signalsJSON); err != nil {
			return nil, false, fmt.Errorf("scan audit page: %w", err)
		}
		if signalsJSON == nil {
			legacy = true
		} else {
			var signals audit.AIPageSignals
			if err := json.Unmarshal([]byte(*signalsJSON), &signals); err != nil {
				return nil, false, fmt.Errorf("decode citation signals for %s: %w", page.URL, err)
			}
			if signals.Version < audit.CitationSignalsVersion {
				legacy = true
			}
			page.Signals = &signals
		}
		pages = append(pages, page)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("read audit pages: %w", err)
	}
	return pages, legacy, nil
}

func (s Store) issues(ctx context.Context, auditID string) ([]Issue, error) {
	rows, err := s.DB.Query(ctx, `SELECT issue_type, page_url, details_json FROM go_audit_issues
		WHERE audit_id = $1 AND issue_type = ANY($2)`, auditID, issueTypes)
	if err != nil {
		return nil, fmt.Errorf("list audit issues: %w", err)
	}
	defer rows.Close()
	issues := []Issue{}
	for rows.Next() {
		var issue Issue
		var detailsJSON *string
		if err := rows.Scan(&issue.Type, &issue.PageURL, &detailsJSON); err != nil {
			return nil, fmt.Errorf("scan audit issue: %w", err)
		}
		if detailsJSON != nil {
			if err := json.Unmarshal([]byte(*detailsJSON), &issue.Details); err != nil {
				return nil, fmt.Errorf("decode %s details for %s: %w", issue.Type, issue.PageURL, err)
			}
		}
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read audit issues: %w", err)
	}
	return issues, nil
}
