package google

import "testing"

// Every provider name callers pass in APIRequest.Provider must resolve to a
// stored grant; "google-analytics" (the provider_id column) did not, and every
// GA4 call failed before reaching Google.
func TestAccountScopeAcceptsEveryCallerProvider(t *testing.T) {
	for provider, wantID := range map[string]string{ProviderGSC: "google-search-console", ProviderGA4: "google-analytics"} {
		if _, id, _, err := accountScope(provider); err != nil || id != wantID {
			t.Errorf("accountScope(%q) = %q, %v; want %q", provider, id, err, wantID)
		}
	}
	if _, _, _, err := accountScope("google-analytics"); err == nil {
		t.Error("the stored provider_id is not a caller provider name")
	}
}
