package dashboard

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Next Actions statuses. "todo" is never stored: it is the absence of a row,
// so setting an action back to todo deletes its row.
const (
	ActionTodo    = "todo"
	ActionDone    = "done"
	ActionIgnored = "ignored"
)

// MaxActionStates bounds how many decisions one project keeps. The page builds
// at most a few dozen actions at a time; the cap stops a buggy or hostile
// client from growing the table without limit.
const MaxActionStates = 500

// actionKey is the stable id the page derives from the action's source, such
// as "audit:missing_meta_description" or "gsc:q:1a2b3c4d".
var actionKey = regexp.MustCompile(`^[a-z][a-z0-9_:.-]{0,119}$`)

// ErrInvalidAction is a key or status the page must never send.
var ErrInvalidAction = errors.New("invalid next action")

// ErrTooManyActions is a project that already holds MaxActionStates decisions.
var ErrTooManyActions = errors.New("too many saved next actions")

// ActionState is one decision on one action.
type ActionState struct {
	Key       string    `json:"key"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ActionStore keeps the project team's Next Actions decisions.
type ActionStore interface {
	Actions(ctx context.Context, organizationID, projectID string) ([]ActionState, error)
	SetAction(ctx context.Context, organizationID, projectID, userID, key, status string) error
}

// ValidateAction checks a key and status before anything is stored.
func ValidateAction(key, status string) error {
	if !actionKey.MatchString(key) {
		return fmt.Errorf("%w: key %q", ErrInvalidAction, key)
	}
	switch status {
	case ActionTodo, ActionDone, ActionIgnored:
		return nil
	}
	return fmt.Errorf("%w: status %q", ErrInvalidAction, status)
}

// Actions lists the project's stored decisions, newest first. The
// organization is matched too, so a project id from another workspace reads
// nothing even if middleware were bypassed.
func (s Store) Actions(ctx context.Context, organizationID, projectID string) ([]ActionState, error) {
	rows, err := s.DB.Query(ctx, `SELECT action_key, status, updated_at FROM go_next_action_states
		WHERE organization_id=$1 AND project_id=$2 ORDER BY updated_at DESC LIMIT $3`, organizationID, projectID, MaxActionStates)
	if err != nil {
		return nil, fmt.Errorf("load next actions: %w", err)
	}
	defer rows.Close()
	out := []ActionState{}
	for rows.Next() {
		var a ActionState
		if err := rows.Scan(&a.Key, &a.Status, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("read next action: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read next actions: %w", err)
	}
	return out, nil
}

// SetAction records a decision. Setting "todo" removes it. A new key is
// refused once the project holds MaxActionStates decisions; changing an
// existing one always works.
func (s Store) SetAction(ctx context.Context, organizationID, projectID, userID, key, status string) error {
	if err := ValidateAction(key, status); err != nil {
		return err
	}
	if status == ActionTodo {
		if _, err := s.DB.Exec(ctx, `DELETE FROM go_next_action_states WHERE organization_id=$1 AND project_id=$2 AND action_key=$3`, organizationID, projectID, key); err != nil {
			return fmt.Errorf("reset next action: %w", err)
		}
		return nil
	}
	// The project row is locked so two members adding their 500th decision at
	// once cannot both pass the count.
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin next action: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var owned bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE id=$1 AND organization_id=$2 FOR UPDATE)`, projectID, organizationID).Scan(&owned); err != nil {
		return fmt.Errorf("lock project for next action: %w", err)
	}
	if !owned {
		return fmt.Errorf("%w: project not in workspace", ErrInvalidAction)
	}
	tag, err := tx.Exec(ctx, `INSERT INTO go_next_action_states (project_id, organization_id, action_key, status, updated_by)
		SELECT $1, $2, $3, $4, $5
		WHERE EXISTS (SELECT 1 FROM go_next_action_states WHERE project_id=$1 AND action_key=$3)
		   OR (SELECT count(*) FROM go_next_action_states WHERE project_id=$1) < $6
		ON CONFLICT (project_id, action_key) DO UPDATE SET status=EXCLUDED.status, updated_by=EXCLUDED.updated_by, updated_at=now()`,
		projectID, organizationID, key, status, userID, MaxActionStates)
	if err != nil {
		return fmt.Errorf("save next action: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTooManyActions
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit next action: %w", err)
	}
	return nil
}
