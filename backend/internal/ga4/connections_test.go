package ga4

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/google"
)

type setupStoreFake struct {
	connection                   ProjectConnection
	connectionErr                error
	grants                       []PropertyGrant
	hasGrant, canManage, deleted bool
	saved                        ProjectConnection
}

func (s *setupStoreFake) GetByProjectID(context.Context, string, string) (ProjectConnection, error) {
	return s.connection, s.connectionErr
}
func (s *setupStoreFake) HasGrant(context.Context, string) (bool, error) { return s.hasGrant, nil }
func (s *setupStoreFake) ListGrants(context.Context, string) ([]PropertyGrant, error) {
	return s.grants, nil
}
func (s *setupStoreFake) CanManage(context.Context, string, string, string) (bool, error) {
	return s.canManage, nil
}
func (s *setupStoreFake) Upsert(_ context.Context, _, _ string, c ProjectConnection) (ProjectConnection, error) {
	s.saved = c
	c.CreatedAt = "2026-01-02T03:04:05.000Z"
	return c, nil
}
func (s *setupStoreFake) Delete(context.Context, string, string) error { s.deleted = true; return nil }

type propertyAdminFake struct {
	properties                   []PropertySummary
	details                      PropertyDetails
	listErr, detailErr, emailErr error
	email                        string
}

func (a propertyAdminFake) ListProperties(context.Context) ([]PropertySummary, error) {
	return a.properties, a.listErr
}
func (a propertyAdminFake) GetProperty(context.Context, string) (PropertyDetails, error) {
	return a.details, a.detailErr
}
func (a propertyAdminFake) UserInfoEmail(context.Context) (string, error) { return a.email, a.emailErr }

func TestListGA4PropertiesMarksSelectionAndProviderState(t *testing.T) {
	store := &setupStoreFake{connection: ProjectConnection{Connection: Connection{PropertyID: "properties/11", GA4AccountID: "acct-a"}}, grants: []PropertyGrant{{AccountID: "acct-a"}, {AccountID: "acct-b"}}}
	ops := &ConnectionOperations{Store: store, NewAdmin: func(_, account string) PropertyAdmin {
		if account == "acct-b" {
			return propertyAdminFake{listErr: errors.New("provider down")}
		}
		return propertyAdminFake{properties: []PropertySummary{{PropertyID: "properties/11", DisplayName: "Site", AccountDisplayName: "Agency"}}, email: "a@example.test"}
	}}
	got, err := ops.ListProperties(context.Background(), "user", "org", "project")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].Properties[0].IsSelected || got[0].Email == nil || *got[0].Email != "a@example.test" || !got[1].PropertiesUnavailable || got[1].RequiresReconnect {
		t.Fatalf("accounts = %+v", got)
	}
}

func TestListGA4PropertiesWithoutOAuthGrantReturnsEmptyList(t *testing.T) {
	ops := &ConnectionOperations{Store: &setupStoreFake{connectionErr: ErrConnectionNotFound}}
	got, err := ops.ListProperties(context.Background(), "user", "org", "project")
	if err != nil || len(got) != 0 {
		t.Fatalf("accounts=%+v err=%v", got, err)
	}
}

func TestListGA4PropertiesMarksRevokedGrantForReconnect(t *testing.T) {
	store := &setupStoreFake{grants: []PropertyGrant{{AccountID: "revoked"}}}
	ops := &ConnectionOperations{Store: store, NewAdmin: func(string, string) PropertyAdmin {
		return propertyAdminFake{listErr: google.APIError{Status: 401}}
	}}
	got, err := ops.ListProperties(context.Background(), "user", "org", "project")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].RequiresReconnect || got[0].PropertiesUnavailable {
		t.Fatalf("account state = %+v", got)
	}
}

func TestSetGA4PropertyVerifiesGrantAndPropertyBeforeSave(t *testing.T) {
	store := &setupStoreFake{canManage: true, grants: []PropertyGrant{{AccountID: "acct-a"}}}
	admin := propertyAdminFake{properties: []PropertySummary{{PropertyID: "properties/11"}}, details: PropertyDetails{Name: "properties/11", DisplayName: "Site", TimeZone: "America/New_York", CurrencyCode: "USD"}}
	ops := &ConnectionOperations{Store: store, NewAdmin: func(string, string) PropertyAdmin { return admin }}
	got, err := ops.SetProperty(context.Background(), "user", "org", "project", "acct-a", "properties/11")
	if err != nil {
		t.Fatal(err)
	}
	if got.PropertyDisplayName != "Site" || got.CreatedAt != "2026-01-02T03:04:05.000Z" || store.saved.PropertyTimeZone != "America/New_York" || store.saved.ConnectedByUserID != "user" {
		t.Fatalf("saved = %+v, returned = %+v", store.saved, got)
	}
	store.saved = ProjectConnection{}
	_, err = ops.SetProperty(context.Background(), "user", "org", "project", "foreign", "properties/11")
	if !errors.Is(err, ErrGrantNotFound) || store.saved.PropertyID != "" {
		t.Fatalf("foreign grant error = %v; saved=%+v", err, store.saved)
	}
	_, err = ops.SetProperty(context.Background(), "user", "org", "project", "acct-a", "properties/12")
	if !errors.Is(err, ErrPropertyNotFound) || store.saved.PropertyID != "" {
		t.Fatalf("foreign property error = %v; saved=%+v", err, store.saved)
	}
}

func TestGA4ConnectionMutationsRequireManager(t *testing.T) {
	store := &setupStoreFake{canManage: false}
	ops := &ConnectionOperations{Store: store}
	if _, err := ops.SetProperty(context.Background(), "user", "org", "project", "acct", "properties/11"); !errors.Is(err, ErrManageForbidden) {
		t.Fatalf("set error = %v", err)
	}
	if err := ops.Disconnect(context.Background(), "user", "org", "project"); !errors.Is(err, ErrManageForbidden) {
		t.Fatalf("disconnect error = %v", err)
	}
	if store.deleted {
		t.Fatal("non-manager disconnected GA4")
	}
}

func TestValidPropertyIDRejectsMalformedProviderInput(t *testing.T) {
	for _, value := range []string{"properties/", "properties/abc", "properties/11/x", "properties/999999999999999999999999999999999"} {
		if validPropertyID(value) {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestListGA4PropertiesSaysWhyPropertiesAreUnavailable(t *testing.T) {
	tests := []struct {
		name    string
		listErr error
		want    string
	}{
		{"analytics permission unticked", google.APIError{Status: 403, Reason: google.ReasonScopeInsufficient}, "missing_permission"},
		{"admin API turned off", google.APIError{Status: 403, Reason: google.ReasonServiceDisabled}, "api_disabled"},
		{"temporary failure", google.APIError{Status: 503}, ""},
		{"not an API error", errors.New("timeout"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs strings.Builder
			ops := &ConnectionOperations{
				Store:  &setupStoreFake{grants: []PropertyGrant{{AccountID: "acct"}}},
				Logger: slog.New(slog.NewTextHandler(&logs, nil)),
				NewAdmin: func(string, string) PropertyAdmin {
					return propertyAdminFake{listErr: tt.listErr}
				},
			}
			got, err := ops.ListProperties(context.Background(), "user", "org", "project")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !got[0].PropertiesUnavailable || got[0].UnavailableReason != tt.want {
				t.Fatalf("account = %+v, want unavailable with reason %q", got, tt.want)
			}
			if !strings.Contains(logs.String(), "list GA4 properties") {
				t.Fatalf("the failure must be logged for the on-call person:\n%s", logs.String())
			}
		})
	}
}

// providerRecorder is a token source that records the provider it was asked
// for and stops the request there.
type providerRecorder struct{ providers []string }

func (r *providerRecorder) AccessToken(_ context.Context, _, provider, _, _ string) (string, error) {
	r.providers = append(r.providers, provider)
	return "", errors.New("stop")
}

func TestGA4GoogleCallsUseTheGA4Provider(t *testing.T) {
	tokens := &providerRecorder{}
	api := &google.APIClient{Tokens: tokens}
	admin := GooglePropertyAdmin{API: api, UserID: "user", AccountID: "acct"}
	_, _ = admin.ListProperties(context.Background())
	_, _ = admin.UserInfoEmail(context.Background())
	if len(tokens.providers) != 2 {
		t.Fatalf("token requests = %v, want 2", tokens.providers)
	}
	for _, provider := range tokens.providers {
		if provider != google.ProviderGA4 {
			t.Fatalf("GA4 asked for provider %q, want %q: the token service rejects any other name", provider, google.ProviderGA4)
		}
	}
}
