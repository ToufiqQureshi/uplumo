package citefix

import (
	"context"
	"slices"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

const maxSEOIssues = 10

// SEOIssue is a critical or warning finding from the latest Site Audit.
type SEOIssue struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	PageURL  string `json:"pageUrl"`
	HowToFix string `json:"howToFix"`
}

func (s Store) seoIssues(ctx context.Context, auditID, projectID string) ([]SEOIssue, error) {
	if s.Audit == nil {
		return []SEOIssue{}, nil
	}
	results, err := s.Audit.GetResults(ctx, auditID, projectID)
	if err != nil {
		return nil, err
	}
	issues := make([]SEOIssue, 0, len(results.Issues))
	for _, issue := range results.Issues {
		descriptor, ok := audit.DescribeIssueType(issue.IssueType)
		if !ok || (descriptor.Severity != audit.SeverityCritical && descriptor.Severity != audit.SeverityWarning) {
			continue
		}
		issues = append(issues, SEOIssue{
			Type: issue.IssueType, Title: descriptor.Title, Severity: string(descriptor.Severity),
			PageURL: issue.PageURL, HowToFix: descriptor.HowToFix,
		})
	}
	slices.SortStableFunc(issues, func(a, b SEOIssue) int {
		if a.Severity == b.Severity {
			if a.PageURL < b.PageURL {
				return -1
			}
			if a.PageURL > b.PageURL {
				return 1
			}
			if a.Type < b.Type {
				return -1
			}
			if a.Type > b.Type {
				return 1
			}
			return 0
		}
		if a.Severity == string(audit.SeverityCritical) {
			return -1
		}
		if b.Severity == string(audit.SeverityCritical) {
			return 1
		}
		return 0
	})
	if len(issues) > maxSEOIssues {
		issues = issues[:maxSEOIssues]
	}
	return issues, nil
}
