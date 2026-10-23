package crawleraccess

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
)

func TestSaveRequiresManagerRoleInTheProjectsWorkspace(t *testing.T) {
	for _, tc := range []struct {
		name, activeOrg, projectOrg, role string
		want                              int
	}{
		{"manager in another workspace", "org-a", "org-b", "owner", http.StatusForbidden},
		{"member in this workspace", "org-b", "org-b", "member", http.StatusForbidden},
		{"manager in this workspace", "org-b", "org-b", "admin", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			identity := func(next http.Handler) http.Handler { return next }
			Mount(mux, &Service{}, identity, identity)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/projects/project-b/crawler-access/save", strings.NewReader(`{}`))
			ctx := auth.WithUser(req.Context(), auth.User{ID: "user", OrganizationID: tc.activeOrg, Role: tc.role})
			ctx = auth.WithProjectOrganization(ctx, tc.projectOrg)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req.WithContext(ctx))
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}
