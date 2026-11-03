// Package support saves authenticated customer reports before notifying support.
package support

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // registers JPEG for image.DecodeConfig
	_ "image/png"  // registers PNG for image.DecodeConfig
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/ids"
)

// MaxScreenshotBytes caps one attached screenshot (1 MiB).
const MaxScreenshotBytes = 1 << 20

var uuidPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

// Errors the handler maps to HTTP responses.
var (
	// ErrInvalid is a report that fails validation.
	ErrInvalid   = errors.New("invalid support report")
	ErrConflict  = errors.New("submission ID already used for another report")
	ErrRateLimit = errors.New("support report limit reached")
	ErrNotFound  = errors.New("support report not found")
)

// Input contains only explicitly submitted diagnostics, never session credentials.
type Input struct {
	Kind         string `json:"kind"`
	Description  string `json:"description"`
	PagePath     string `json:"pagePath"`
	AppVersion   string `json:"appVersion"`
	RequestID    string `json:"requestId"`
	SubmissionID string `json:"submissionId"`
	Screenshot   string `json:"screenshot,omitempty"`
}

// Service saves reports and notifies support.
type Service struct {
	DB *pgxpool.Pool
	// StaffEmails is the server-configured list, normalized by config.Load.
	StaffEmails []string
	TemplateID  string
	Mailer      Mailer
	PublicURL   string
}

func validate(in Input) ([]byte, string, error) {
	if in.Kind != "bug" && in.Kind != "billing" && in.Kind != "suggestion" {
		return nil, "", ErrInvalid
	}
	n := utf8.RuneCountInString(strings.TrimSpace(in.Description))
	if n < 10 || n > 5000 || !utf8.ValidString(in.Description) || !uuidPattern.MatchString(in.SubmissionID) || len(in.AppVersion) > 100 || len(in.RequestID) > 128 {
		return nil, "", ErrInvalid
	}
	u, err := url.Parse(in.PagePath)
	if err != nil || len(in.PagePath) > 500 || !strings.HasPrefix(in.PagePath, "/") || strings.HasPrefix(in.PagePath, "//") || strings.ContainsAny(in.PagePath, "?#\\\r\n") || u.Host != "" || u.Scheme != "" || strings.ContainsRune(u.Path, '\\') || strings.ContainsFunc(u.Path, unicode.IsControl) {
		return nil, "", ErrInvalid
	}
	if strings.ContainsAny(in.AppVersion+in.RequestID, "\r\n\x00") {
		return nil, "", ErrInvalid
	}
	if in.Screenshot == "" {
		return nil, "", nil
	}
	mime, encoded, ok := strings.Cut(in.Screenshot, ",")
	if !ok || (mime != "data:image/png;base64" && mime != "data:image/jpeg;base64") || len(encoded) > base64.StdEncoding.EncodedLen(MaxScreenshotBytes) {
		return nil, "", ErrInvalid
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) > MaxScreenshotBytes {
		return nil, "", ErrInvalid
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 || "data:image/"+format+";base64" != mime {
		return nil, "", ErrInvalid
	}
	return data, "image/" + format, nil
}

// Create persists once per member/workspace submission, before email delivery.
func (s *Service) Create(ctx context.Context, user auth.User, in Input) (string, error) {
	screenshot, mime, err := validate(in)
	if err != nil {
		return "", err
	}
	if user.ID == "" || user.OrganizationID == "" {
		return "", ErrInvalid
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin support report: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// One user's parallel submissions share the hourly quota across workspaces.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('support:' || $1,0))`, user.ID); err != nil {
		return "", err
	}
	var id string
	var prior []byte
	err = tx.QueryRow(ctx, `SELECT id,payload_hash FROM go_support_reports WHERE organization_id=$1 AND user_id=$2 AND submission_id=$3`, user.OrganizationID, user.ID, in.SubmissionID).Scan(&id, &prior)
	if err == nil {
		if !bytes.Equal(prior, digest[:]) {
			return "", ErrConflict
		}
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM go_support_reports WHERE user_id=$1 AND created_at>now()-interval '1 hour'`, user.ID).Scan(&count); err != nil {
		return "", err
	}
	if count >= 5 {
		return "", ErrRateLimit
	}
	id, err = ids.New()
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO go_support_reports(id,organization_id,user_id,submission_id,payload_hash,kind,description,page_path,app_version,request_id,screenshot,screenshot_mime) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, user.OrganizationID, user.ID, in.SubmissionID, digest[:], in.Kind, strings.TrimSpace(in.Description), in.PagePath, in.AppVersion, in.RequestID, screenshot, mime)
	if err != nil {
		return "", fmt.Errorf("save support report: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit support report: %w", err)
	}
	return id, nil
}

// Screenshot returns a private attachment to its submitter or verified staff.
func (s *Service) Screenshot(ctx context.Context, user auth.User, id string) ([]byte, string, error) {
	var data []byte
	var mime string
	err := s.DB.QueryRow(ctx, `SELECT screenshot,screenshot_mime FROM go_support_reports WHERE id=$1 AND ((organization_id=$2 AND user_id=$3) OR EXISTS(SELECT 1 FROM "user" u JOIN member m ON m.user_id=u.id WHERE u.id=$3 AND u.email_verified AND lower(u.email)=ANY($4) AND m.organization_id=$2 AND m.role='owner')) AND screenshot IS NOT NULL`, id, user.OrganizationID, user.ID, s.StaffEmails).Scan(&data, &mime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return data, mime, err
}
