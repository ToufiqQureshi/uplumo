package emailauth

import (
	"net/http"
	"testing"
)

func TestDeleteAccountRequiresOriginConfirmationAndSignedSession(t *testing.T) {
	e := newAuthEnv(t, envOptions{bypass: true})
	signup := e.request(http.MethodPost, "/api/auth/sign-up/email", signUpBody("erasure@example.com"), originHeaders())
	cookie := signup.sessionCookie()
	if cookie == nil {
		t.Fatalf("signup did not set a session: %s", signup.raw)
	}
	const path = "/api/auth/delete-account"
	confirm := map[string]bool{"confirm": true}
	if resp := e.request(http.MethodPost, path, confirm, map[string]string{"Origin": "https://other.example"}, cookie); resp.StatusCode != http.StatusForbidden || resp.errCode() != "INVALID_ORIGIN" {
		t.Fatalf("cross-origin deletion = %d %s", resp.StatusCode, resp.raw)
	}
	if resp := e.request(http.MethodPost, path, map[string]bool{}, originHeaders(), cookie); resp.StatusCode != http.StatusBadRequest || resp.errCode() != "CONFIRMATION_REQUIRED" {
		t.Fatalf("unconfirmed deletion = %d %s", resp.StatusCode, resp.raw)
	}
	if resp := e.request(http.MethodPost, path, confirm, originHeaders()); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous deletion = %d %s", resp.StatusCode, resp.raw)
	}
	if resp := e.request(http.MethodPost, path, map[string]any{"confirm": true, "reason": "free text"}, originHeaders(), cookie); resp.StatusCode != http.StatusBadRequest || resp.errCode() != "INVALID_DELETION_REASON" {
		t.Fatalf("unrecognized reason = %d %s", resp.StatusCode, resp.raw)
	}
	if _, err := e.authenticate(cookie); err != nil {
		t.Fatalf("refused deletion ended session: %v", err)
	}
}
