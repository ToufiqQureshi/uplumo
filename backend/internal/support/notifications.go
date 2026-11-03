package support

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Mailer is the existing transactional transport, injectable for failure tests.
type Mailer interface {
	SendTransactional(context.Context, string, string, map[string]string) error
}

// NotifyPending sends ten due notifications at most. Row locks keep replicas
// from sending concurrently; a crash after provider acceptance can resend an email.
func (s *Service) NotifyPending(ctx context.Context) (int, error) {
	if s.Mailer == nil || s.TemplateID == "" {
		return 0, nil
	}
	sent := 0
	for range 10 {
		tx, err := s.DB.Begin(ctx)
		if err != nil {
			return sent, err
		}
		var id string
		var attempts int
		var hasScreenshot bool
		var kind, description, page, version, request, reporter, org string
		err = tx.QueryRow(ctx, `SELECT r.id,r.kind,r.description,r.page_path,r.app_version,r.request_id,u.email,r.organization_id,r.screenshot IS NOT NULL,r.notification_attempts FROM go_support_reports r JOIN "user" u ON u.id=r.user_id WHERE r.notified_at IS NULL AND r.notification_attempts<10 AND r.next_notification_at<=now() ORDER BY r.next_notification_at,r.id FOR UPDATE OF r SKIP LOCKED LIMIT 1`).Scan(&id, &kind, &description, &page, &version, &request, &reporter, &org, &hasScreenshot, &attempts)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			return sent, nil
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return sent, err
		}
		vars := map[string]string{"ticketId": id, "kind": kind, "description": description, "pagePath": page, "appVersion": version, "requestId": request, "reporterEmail": reporter, "organizationId": org, "screenshotUrl": ""}
		if hasScreenshot {
			vars["screenshotUrl"] = strings.TrimRight(s.PublicURL, "/") + "/api/v1/support/reports/" + id + "/screenshot"
		}
		sendErr := s.Mailer.SendTransactional(ctx, "support@uplumo.com", s.TemplateID, vars)
		if sendErr == nil {
			_, err = tx.Exec(ctx, `UPDATE go_support_reports SET notified_at=now(),notification_attempts=notification_attempts+1 WHERE id=$1`, id)
		} else {
			delay := min(time.Minute*time.Duration(1<<attempts), time.Hour)
			_, err = tx.Exec(ctx, `UPDATE go_support_reports SET notification_attempts=notification_attempts+1,next_notification_at=now()+$2::interval WHERE id=$1`, id, fmt.Sprintf("%d seconds", int(delay.Seconds())))
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return sent, err
		}
		if err = tx.Commit(ctx); err != nil {
			return sent, err
		}
		if sendErr != nil {
			return sent, fmt.Errorf("notify support ticket %s: %w", id, sendErr)
		}
		sent++
	}
	return sent, nil
}

// RunNotifications drains persisted reports once a minute until shutdown.
func (s *Service) RunNotifications(ctx context.Context, logger *slog.Logger) {
	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	for {
		if _, err := s.NotifyPending(ctx); err != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "support notification failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}
