package gdpr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

var (
	ErrSessionExpired   = errors.New("no valid account deletion session")
	ErrSessionNotRecent = errors.New("account deletion requires a recent sign-in")
	ErrOpenBilling      = errors.New("a workspace has a subscription or open payment")
)

// EraseAccount deletes the account identified by a verified browser cookie's
// token. Session, ownership, billing checks and erasure share one transaction.
func (e Eraser) EraseAccount(ctx context.Context, token string, recentSince time.Time) error {
	return pgdb.InTx(ctx, e.DB, func(tx pgx.Tx) error {
		var userID string
		var createdAt time.Time
		err := tx.QueryRow(ctx, `SELECT u.id, s.created_at FROM "user" u JOIN session s ON s.user_id = u.id
			WHERE s.token = $1 AND s.expires_at > now()
			  AND NOT EXISTS (SELECT 1 FROM go_user_blocks b WHERE b.user_id = u.id)
			FOR UPDATE OF u, s`, token).Scan(&userID, &createdAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionExpired
		}
		if err != nil {
			return fmt.Errorf("lock deletion session: %w", err)
		}
		if createdAt.Before(recentSince) {
			return ErrSessionNotRecent
		}

		orgIDs, err := lockOwnedOrganizations(ctx, tx, userID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT id FROM projects WHERE organization_id = ANY($1)`, orgIDs)
		if err != nil {
			return fmt.Errorf("list projects for account deletion: %w", err)
		}
		projectIDs, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return fmt.Errorf("read projects for account deletion: %w", err)
		}
		return eraseTx(ctx, tx, userID, orgIDs, projectIDs)
	})
}

func lockOwnedOrganizations(ctx context.Context, tx pgx.Tx, userID string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT organization_id, role FROM member WHERE user_id = $1 ORDER BY organization_id FOR UPDATE`, userID)
	if err != nil {
		return nil, fmt.Errorf("lock account memberships: %w", err)
	}
	type membership struct{ organizationID, role string }
	memberships, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (membership, error) {
		var m membership
		return m, row.Scan(&m.organizationID, &m.role)
	})
	if err != nil {
		return nil, fmt.Errorf("read account memberships: %w", err)
	}
	orgIDs := make([]string, 0, len(memberships))
	for _, member := range memberships {
		if member.role != "owner" {
			return nil, ErrSharedOrganization
		}
		orgIDs = append(orgIDs, member.organizationID)
		// Same key as billing's CheckoutPlan. After waiting for a deletion,
		// that flow checks whether the organization still exists.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('checkout:' || $1, 0))`, member.organizationID); err != nil {
			return nil, fmt.Errorf("lock billing checkout for erasure: %w", err)
		}
	}
	if err := lockPrivateOrganizations(ctx, tx, orgIDs, userID); err != nil {
		return nil, err
	}
	var open bool
	if err := tx.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM go_billing_subscriptions WHERE organization_id = ANY($1) AND status NOT IN ('cancelled', 'completed', 'expired'))
		OR EXISTS (SELECT 1 FROM go_billing_topups WHERE organization_id = ANY($1)
			AND (status IN ('creating', 'ready') OR (status = 'failed' AND order_id IS NOT NULL)))`, orgIDs).Scan(&open); err != nil {
		return nil, fmt.Errorf("check billing before erasure: %w", err)
	}
	if open {
		return nil, ErrOpenBilling
	}
	return orgIDs, nil
}
