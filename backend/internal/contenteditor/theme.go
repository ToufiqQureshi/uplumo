package contenteditor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// MaxThemeCSS is the largest blog theme a project can save (100 KB).
const MaxThemeCSS = 100 << 10

// Theme is a project's own blog CSS; an empty CSS means none is saved.
type Theme struct {
	CSS       string     `json:"css"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

// cleanThemeCSS checks pasted CSS. The CSS is placed inside a <style> tag in
// the preview frame and in exported files, so any "<" could close that tag
// and inject HTML; plain CSS never needs one outside a string.
func cleanThemeCSS(css string) (string, error) {
	css = strings.TrimSpace(css)
	switch {
	case len(css) > MaxThemeCSS:
		return "", &ValidationError{Message: "This theme is over 100 KB. Remove rules your blog posts do not use and try again."}
	case strings.Contains(css, "<"):
		return "", &ValidationError{Message: "Paste CSS only. Remove HTML tags such as <style> or <link>."}
	}
	return css, nil
}

// GetTheme returns the project's saved blog CSS, or an empty Theme.
func (r Repository) GetTheme(ctx context.Context, projectID string) (Theme, error) {
	var theme Theme
	var updated time.Time
	err := r.DB.QueryRow(ctx, `SELECT css, updated_at FROM go_content_blog_themes WHERE project_id=$1`, projectID).Scan(&theme.CSS, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return Theme{}, nil
	}
	if err != nil {
		return Theme{}, fmt.Errorf("get blog theme: %w", err)
	}
	theme.UpdatedAt = &updated
	return theme, nil
}

// SaveTheme stores the project's blog CSS; empty CSS removes it.
func (r Repository) SaveTheme(ctx context.Context, projectID, css string) (Theme, error) {
	css, err := cleanThemeCSS(css)
	if err != nil {
		return Theme{}, err
	}
	if css == "" {
		if _, err := r.DB.Exec(ctx, `DELETE FROM go_content_blog_themes WHERE project_id=$1`, projectID); err != nil {
			return Theme{}, fmt.Errorf("remove blog theme: %w", err)
		}
		return Theme{}, nil
	}
	var updated time.Time
	err = r.DB.QueryRow(ctx, `INSERT INTO go_content_blog_themes (project_id, css) VALUES ($1, $2)
		ON CONFLICT (project_id) DO UPDATE SET css=EXCLUDED.css, updated_at=now() RETURNING updated_at`, projectID, css).Scan(&updated)
	if err != nil {
		return Theme{}, fmt.Errorf("save blog theme: %w", err)
	}
	return Theme{CSS: css, UpdatedAt: &updated}, nil
}

func (h Handler) getTheme(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "content_editor_unavailable", "The content editor is not available on this server.")
		return
	}
	var empty map[string]any
	if !decode(w, r, &empty) {
		return
	}
	if len(empty) != 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Choose a valid blog theme request.")
		return
	}
	theme, err := h.Store.GetTheme(r.Context(), r.PathValue("projectId"))
	if err != nil {
		h.internalError(w, r, "get blog theme", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, theme)
}

func (h Handler) saveTheme(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "content_editor_unavailable", "The content editor is not available on this server.")
		return
	}
	var input struct {
		CSS *string `json:"css"`
	}
	if !decodeLimit(w, r, &input, MaxThemeCSS+(16<<10)) {
		return
	}
	if input.CSS == nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Choose a valid blog theme request.")
		return
	}
	css, err := cleanThemeCSS(*input.CSS)
	if err == nil {
		var theme Theme
		theme, err = h.Store.SaveTheme(r.Context(), r.PathValue("projectId"), css)
		if err == nil {
			httpx.WriteJSON(w, http.StatusOK, theme)
			return
		}
	}
	var validationError *ValidationError
	if errors.As(err, &validationError) {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", validationError.Error())
		return
	}
	h.internalError(w, r, "save blog theme", err)
}
