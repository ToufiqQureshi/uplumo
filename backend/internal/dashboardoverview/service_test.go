package dashboardoverview

import (
	"context"
	"errors"
	"testing"
	"time"
)

type storeStub struct {
	domain                *string
	audit                 *AuditSummary
	snapshot              BacklinkSummary
	found                 bool
	project, organization string
}

func (s *storeStub) ProjectDomain(_ context.Context, p, o string) (*string, error) {
	s.project, s.organization = p, o
	return s.domain, nil
}
func (s *storeStub) LatestAudit(context.Context, string) (*AuditSummary, error) { return s.audit, nil }
func (s *storeStub) LatestBacklinkSnapshot(context.Context, string, string) (BacklinkSummary, bool, error) {
	return s.snapshot, s.found, nil
}
func TestServiceScopesProjectAndMarksSnapshotStaleness(t *testing.T) {
	domain := "example.com"
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	store := &storeStub{domain: &domain, audit: &AuditSummary{Status: "completed", TopIssues: []TopIssue{}}, snapshot: BacklinkSummary{Domain: domain, CapturedAt: now.Add(-25 * time.Hour).Format(time.RFC3339Nano)}, found: true}
	svc := &Service{Store: store, Now: func() time.Time { return now }}
	got, err := svc.Get(context.Background(), "org-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if store.organization != "org-1" || store.project != "project-1" || !got.Backlinks.Stale || got.Audit == nil {
		t.Fatalf("overview=%+v scope=%s/%s", got, store.organization, store.project)
	}
	store.snapshot.Domain = "another.example"
	got, err = svc.Get(context.Background(), "org-1", "project-1")
	if err != nil || got.Backlinks != nil {
		t.Fatalf("foreign-domain snapshot=%+v err=%v", got.Backlinks, err)
	}
}
func TestSortIssuesMatchesSeverityThenPageCount(t *testing.T) {
	rows := []TopIssue{{IssueType: "few-critical", Severity: "critical", Count: 1}, {IssueType: "warning", Severity: "warning", Count: 40}, {IssueType: "many-critical", Severity: "critical", Count: 8}, {IssueType: "info", Severity: "info", Count: 100}}
	sortIssues(rows)
	if rows[0].IssueType != "many-critical" || rows[1].IssueType != "few-critical" || rows[2].IssueType != "warning" {
		t.Fatalf("sorted issues=%+v", rows)
	}
}
func TestRefreshWithoutBacklinkProviderReportsUnavailable(t *testing.T) {
	domain := "example.com"
	svc := &Service{Store: writableStore{&storeStub{domain: &domain}}}
	got, err := svc.Get(t.Context(), "org-1", "project-1")
	if err != nil || got.BacklinkRefreshAvailable {
		t.Fatalf("unavailable provider advertised: %+v, %v", got, err)
	}
	if err := svc.RefreshBacklinkSnapshot(context.Background(), "org-1", "project-1"); !errors.Is(err, ErrBacklinksUnavailable) {
		t.Fatalf("refresh without provider: %v", err)
	}
}

type cancelProjectLookup struct {
	writableStore
	cancel context.CancelFunc
}

func (s cancelProjectLookup) ProjectDomain(ctx context.Context, projectID, org string) (*string, error) {
	domain, err := s.writableStore.ProjectDomain(ctx, projectID, org)
	s.cancel()
	return domain, err
}

func TestRefreshCancelledDuringProjectLookupDoesNotBuyTasks(t *testing.T) {
	domain := "example.com"
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	provider := &snapshotProvider{}
	svc := &Service{Store: cancelProjectLookup{writableStore{&storeStub{domain: &domain}}, cancel}, Backlinks: provider}
	err := svc.RefreshBacklinkSnapshot(ctx, "org", "project")
	_, _, _ = svc.refreshes.Do("org\x00project\x00example.com", func() (any, error) { return nil, nil })
	if !errors.Is(err, context.Canceled) || provider.calls.Load() != 0 {
		t.Fatalf("cancelled refresh=%v purchases=%d", err, provider.calls.Load())
	}
}
