package emailauth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/gdpr"
)

const deletionSessionAge = 15 * time.Minute

func (s *Service) deleteAccount(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		Confirm bool   `json:"confirm"`
		Reason  string `json:"reason"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	if !input.Confirm {
		return &apiError{http.StatusBadRequest, "CONFIRMATION_REQUIRED", "Confirm that you want to delete your account."}
	}
	cookie, err := r.Cookie(securePrefix + sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		cookie, err = r.Cookie(sessionCookieName)
	}
	if err != nil {
		return errUnauthorized
	}
	token, ok := s.jar.verify(cookie.Value)
	if !ok {
		return errUnauthorized
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	eraser := gdpr.Eraser{DB: s.DB}
	switch err := eraser.EraseAccount(ctx, token, s.Now().Add(-deletionSessionAge), input.Reason); {
	case errors.Is(err, gdpr.ErrInvalidReason):
		return &apiError{http.StatusBadRequest, "INVALID_DELETION_REASON", "Choose a reason from the list or leave it blank."}
	case errors.Is(err, gdpr.ErrSessionExpired):
		return errUnauthorized
	case errors.Is(err, gdpr.ErrSessionNotRecent):
		return &apiError{http.StatusForbidden, "RECENT_SIGN_IN_REQUIRED", "Sign in again, then confirm account deletion."}
	case errors.Is(err, gdpr.ErrSharedOrganization):
		return &apiError{http.StatusConflict, "SHARED_WORKSPACE", "Transfer ownership or leave shared workspaces before deleting your account. Contact support if you need help."}
	case errors.Is(err, gdpr.ErrOpenBilling):
		return &apiError{http.StatusConflict, "OPEN_BILLING", "End any active subscription and contact support if a payment is still open. Then try deleting your account again."}
	case err != nil:
		return err
	}

	for _, base := range []string{sessionCookieName, sessionDataCookieName, dontRememberCookieName} {
		for _, prefix := range []string{securePrefix, ""} {
			// #nosec G124 -- clears both the Secure and legacy cookie variants.
			http.SetCookie(w, &http.Cookie{
				Name: prefix + base, Path: "/", MaxAge: -1,
				HttpOnly: true, Secure: prefix == securePrefix, SameSite: http.SameSiteLaxMode,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	return nil
}
