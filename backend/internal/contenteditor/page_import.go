package contenteditor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
	"golang.org/x/sync/singleflight"
)

const (
	pageImportTimeout   = 10 * time.Second
	pageImportMaxBytes  = 2 << 20
	pageImportRedirects = 3
	maxImportURLLength  = 2048
)

var (
	// ErrImportNoDomain means the project has no website to import from.
	ErrImportNoDomain = errors.New("project has no website domain")
	// ErrImportOtherSite means the URL is not on the project's website.
	ErrImportOtherSite = errors.New("page is not on the project's website")
	// ErrImportUnreadable means the page could not be fetched or had no text.
	ErrImportUnreadable = errors.New("page could not be read")
)

// PageImporter turns a page of the project's own website into a draft. It
// spends nothing: the server fetches the page itself through Client, which
// must refuse private and internal addresses (audit.Guard.NewClient).
type PageImporter struct {
	Drafts  Repository
	Client  *http.Client
	Logger  *slog.Logger
	flights singleflight.Group
}

// Import returns the project's draft of rawURL, creating it on first use.
// Importing the same page again opens the existing draft, edits intact.
func (p *PageImporter) Import(ctx context.Context, projectID, rawURL string) (Draft, error) {
	if len(rawURL) > maxImportURLLength {
		return Draft{}, ErrImportOtherSite
	}
	var domain *string
	err := p.Drafts.DB.QueryRow(ctx, `SELECT domain FROM projects WHERE id=$1 AND archived_at IS NULL`, projectID).Scan(&domain)
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, ErrNotFound
	}
	if err != nil {
		return Draft{}, fmt.Errorf("read project domain: %w", err)
	}
	if domain == nil || *domain == "" {
		return Draft{}, ErrImportNoDomain
	}
	page, err := siteURL(rawURL, *domain)
	if err != nil {
		return Draft{}, err
	}
	result, err, _ := p.flights.Do(projectID+"\x00"+page, func() (any, error) {
		if draft, ok, err := p.existing(ctx, projectID, page); err != nil || ok {
			return draft, err
		}
		return p.create(ctx, projectID, page, *domain)
	})
	if err != nil {
		return Draft{}, err
	}
	return result.(Draft), nil
}

// siteURL accepts http(s) URLs on the project's domain or its subdomains and
// returns them without a fragment.
func siteURL(raw, domain string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return "", ErrImportOtherSite
	}
	if !onSite(u.Hostname(), domain) {
		return "", ErrImportOtherSite
	}
	u.Fragment = ""
	u.RawFragment = ""
	return u.String(), nil
}

func onSite(host, domain string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	domain = strings.TrimPrefix(strings.ToLower(domain), "www.")
	host = strings.TrimPrefix(host, "www.")
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func (p *PageImporter) existing(ctx context.Context, projectID, page string) (Draft, bool, error) {
	var id string
	err := p.Drafts.DB.QueryRow(ctx, `SELECT id FROM go_content_drafts WHERE project_id=$1 AND source_url=$2`, projectID, page).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, false, nil
	}
	if err != nil {
		return Draft{}, false, fmt.Errorf("find imported draft: %w", err)
	}
	draft, err := p.Drafts.Get(ctx, projectID, id)
	return draft, err == nil, err
}

func (p *PageImporter) create(ctx context.Context, projectID, page, domain string) (Draft, error) {
	document, finalURL, err := p.fetch(ctx, page, domain)
	if err != nil {
		if p.Logger != nil {
			p.Logger.InfoContext(ctx, "page import fetch failed", "url", page, "err", err)
		}
		return Draft{}, ErrImportUnreadable
	}
	content := pageToMarkdown(document, finalURL)
	if content.Markdown == "" {
		return Draft{}, ErrImportUnreadable
	}
	slug := ""
	if u, err := url.Parse(finalURL); err == nil {
		slug = strings.Trim(u.Path, "/")
	}
	draft, err := p.Drafts.Save(ctx, projectID, Input{
		Title:           truncateRunes(content.Title, MaxTitleLength),
		Slug:            truncateRunes(slug, MaxSlugLength),
		MetaDescription: truncateRunes(content.MetaDescription, MaxMetaLength),
		Content:         truncateMarkdown(content.Markdown, MaxContentLength),
	})
	if err != nil {
		return Draft{}, err
	}
	_, err = p.Drafts.DB.Exec(ctx, `UPDATE go_content_drafts SET source_url=$3 WHERE id=$1 AND project_id=$2`, draft.ID, projectID, page)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		// Another server imported this page first: keep theirs.
		_ = p.Drafts.Delete(ctx, projectID, draft.ID)
		existing, _, findErr := p.existing(ctx, projectID, page)
		return existing, findErr
	}
	if err != nil {
		return Draft{}, fmt.Errorf("record imported page: %w", err)
	}
	return draft, nil
}

// fetch reads the page, following at most three redirects that stay on the
// project's site, and returns its HTML and final address.
func (p *PageImporter) fetch(ctx context.Context, page, domain string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, pageImportTimeout)
	defer cancel()
	current := page
	for range pageImportRedirects + 1 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("User-Agent", "UplumoBot/1.0 (+https://uplumo.com)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		resp, err := p.Client.Do(req)
		if err != nil {
			return "", "", err
		}
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			_ = resp.Body.Close()
			next, err := url.Parse(location)
			if err != nil || location == "" {
				return "", "", fmt.Errorf("bad redirect %q", location)
			}
			base, _ := url.Parse(current)
			current, err = siteURL(base.ResolveReference(next).String(), domain)
			if err != nil {
				return "", "", fmt.Errorf("redirect leaves the site: %w", err)
			}
			continue
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("status %d", resp.StatusCode)
		}
		if media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); media != "text/html" && media != "application/xhtml+xml" {
			return "", "", fmt.Errorf("not html: %q", media)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, pageImportMaxBytes))
		if err != nil {
			return "", "", err
		}
		return string(body), current, nil
	}
	return "", "", errors.New("too many redirects")
}

// truncateMarkdown keeps whole paragraphs within limit bytes.
func truncateMarkdown(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := strings.LastIndex(s[:limit], "\n\n")
	if cut <= 0 {
		cut = limit
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
	}
	return strings.TrimSpace(s[:cut])
}

// MountImport registers the page import endpoint.
func MountImport(mux *http.ServeMux, importer *PageImporter, withSession, withProjectAccess func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/projects/{projectId}/content/drafts/import", withSession(withProjectAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.ProjectOrganizationFromContext(r.Context()); !ok {
			httpx.WriteError(w, http.StatusNotFound, "project_not_found", "Project not found.")
			return
		}
		var in struct {
			URL string `json:"url"`
		}
		if err := httpx.DecodeJSONLimit(r.Body, 4<<10, &in); err != nil || strings.TrimSpace(in.URL) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Send the page URL to edit.")
			return
		}
		draft, err := importer.Import(r.Context(), r.PathValue("projectId"), in.URL)
		var validation *ValidationError
		switch {
		case err == nil:
			httpx.WriteJSON(w, http.StatusOK, draft)
		case errors.Is(err, ErrImportNoDomain):
			httpx.WriteError(w, http.StatusBadRequest, "no_domain", "Add your website's domain in project settings first.")
		case errors.Is(err, ErrImportOtherSite):
			httpx.WriteError(w, http.StatusBadRequest, "other_site", "Only pages on this project's website can be opened in the editor.")
		case errors.Is(err, ErrImportUnreadable):
			httpx.WriteError(w, http.StatusUnprocessableEntity, "page_unreadable", "We could not read text from that page. Check that it loads and is public.")
		case errors.Is(err, ErrDraftLimit):
			httpx.WriteError(w, http.StatusConflict, "draft_limit", fmt.Sprintf("This project already has %d drafts. Delete one to open another page.", MaxDraftsPerProject))
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "project_not_found", "Project not found.")
		case errors.As(err, &validation):
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", validation.Message)
		default:
			if importer.Logger != nil {
				importer.Logger.ErrorContext(r.Context(), "import page into editor", "err", err)
			}
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Could not open the page. Try again.")
		}
	}))))
}
