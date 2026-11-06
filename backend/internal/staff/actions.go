package staff

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/gdpr"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// Errors the handler maps to 404, 409 and 400 responses.
var (
	// ErrUserNotFound means no user has the id.
	ErrUserNotFound     = errors.New("user not found")
	ErrProtectedUser    = errors.New("staff accounts and your own account cannot be blocked or deleted here")
	ErrLiveSubscription = errors.New("cancel this user's paid plan in Razorpay first")
	ErrConfirmMismatch  = errors.New("type the user's email exactly to confirm")
	ErrSharedWorkspace  = errors.New("this user's workspace has other members; remove them or the user from it first")
)

// Actions changes accounts on a staff member's behalf and logs each change.
type Actions struct {
	DB          *pgxpool.Pool
	StaffEmails []string
}

// target loads the user an action is about and refuses staff and self.
func (a Actions) target(ctx context.Context, actorID, userID string) (string, error) {
	var email string
	err := a.DB.QueryRow(ctx, `SELECT email FROM "user" WHERE id = $1`, userID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load user %s: %w", userID, err)
	}
	if userID == actorID || slices.Contains(a.StaffEmails, strings.ToLower(email)) {
		return "", ErrProtectedUser
	}
	return email, nil
}

// Block stops a user at once: their sessions are deleted and every sign-in,
// session, API key and MCP token check refuses them while the block exists.
// Their data and any paid plan are left as they are.
func (a Actions) Block(ctx context.Context, actorID, userID, reason string) error {
	if _, err := a.target(ctx, actorID, userID); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return pgdb.InTx(ctx, a.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO go_user_blocks(user_id, blocked_by, reason) VALUES($1, $2, $3) ON CONFLICT (user_id) DO NOTHING`, userID, actorID, reason); err != nil {
			return fmt.Errorf("block user %s: %w", userID, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM session WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("end sessions of user %s: %w", userID, err)
		}
		return logTx(ctx, tx, actorID, "block", userID, reason)
	})
}

// Unblock lets a blocked user sign in again.
func (a Actions) Unblock(ctx context.Context, actorID, userID string) error {
	if _, err := a.target(ctx, actorID, userID); err != nil {
		return err
	}
	return pgdb.InTx(ctx, a.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM go_user_blocks WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("unblock user %s: %w", userID, err)
		}
		return logTx(ctx, tx, actorID, "unblock", userID, "")
	})
}

// Delete erases a user and the workspaces only they belong to, keeping
// payment records. It refuses a user with a paid plan still running, so
// Razorpay never keeps charging an account that no longer exists.
func (a Actions) Delete(ctx context.Context, actorID, userID, confirmEmail string) error {
	email, err := a.target(ctx, actorID, userID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(confirmEmail), email) {
		return ErrConfirmMismatch
	}
	var live bool
	if err := a.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM member m JOIN go_billing_subscriptions bs ON bs.organization_id = m.organization_id
		WHERE m.user_id = $1 AND bs.status = ANY($2))`, userID, liveStatuses).Scan(&live); err != nil {
		return fmt.Errorf("check paid plan of user %s: %w", userID, err)
	}
	if live {
		return ErrLiveSubscription
	}
	eraser := gdpr.Eraser{DB: a.DB}
	subject, err := eraser.Find(ctx, "", userID)
	if errors.Is(err, gdpr.ErrUserNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	plan, err := eraser.Plan(ctx, subject)
	if errors.Is(err, gdpr.ErrSharedOrganization) {
		return ErrSharedWorkspace
	}
	if err != nil {
		return err
	}
	if err := eraser.Execute(ctx, plan); err != nil {
		if errors.Is(err, gdpr.ErrSharedOrganization) {
			return ErrSharedWorkspace
		}
		return fmt.Errorf("erase user %s: %w", userID, err)
	}
	// The log keeps only the opaque id: the email was just erased.
	if _, err := a.DB.Exec(ctx, `INSERT INTO go_staff_actions(actor_user_id, action, target_user_id, detail) VALUES($1, 'delete', $2, $3)`,
		actorID, userID, fmt.Sprintf("%d workspace(s)", len(plan.Organizations))); err != nil {
		return fmt.Errorf("log delete of user %s: %w", userID, err)
	}
	return nil
}

func logTx(ctx context.Context, tx pgx.Tx, actor, action, target, detail string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO go_staff_actions(actor_user_id, action, target_user_id, detail) VALUES($1, $2, $3, $4)`, actor, action, target, detail); err != nil {
		return fmt.Errorf("log staff action %s: %w", action, err)
	}
	return nil
}

// LogSync records a Razorpay payment sync.
func (a Actions) LogSync(ctx context.Context, actorID, detail string) error {
	_, err := a.DB.Exec(ctx, `INSERT INTO go_staff_actions(actor_user_id, action, detail) VALUES($1, 'sync_payments', $2)`, actorID, detail)
	if err != nil {
		return fmt.Errorf("log payment sync: %w", err)
	}
	return nil
}

// Action is one logged staff change.
type Action struct {
	ActorEmail  string    `json:"actorEmail"`
	Action      string    `json:"action"`
	TargetEmail string    `json:"targetEmail"`
	TargetID    string    `json:"targetId"`
	Detail      string    `json:"detail"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Recent returns the last 100 staff actions; a deleted user shows by id only.
func (a Actions) Recent(ctx context.Context) ([]Action, error) {
	return pgxCollect(ctx, a.DB, `SELECT coalesce(au.email, ''), sa.action, coalesce(tu.email, ''), sa.target_user_id, sa.detail, sa.created_at
		FROM go_staff_actions sa LEFT JOIN "user" au ON au.id = sa.actor_user_id LEFT JOIN "user" tu ON tu.id = sa.target_user_id
		ORDER BY sa.created_at DESC LIMIT 100`,
		func(r pgx.CollectableRow) (Action, error) {
			var v Action
			return v, r.Scan(&v.ActorEmail, &v.Action, &v.TargetEmail, &v.TargetID, &v.Detail, &v.CreatedAt)
		})
}
