// Package dashboard stores each member's dashboard layout: which cards show on
// a project's dashboard and in what order. It holds no card data; every card
// reads its own feature API.
package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxCards bounds a saved layout. The client has under a dozen cards; the cap
// only stops a crafted request from storing an unbounded document.
const MaxCards = 30

var cardID = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// ErrInvalidLayout reports a layout that fails validation.
var ErrInvalidLayout = errors.New("invalid dashboard layout")

// Card is one dashboard card in a saved layout.
type Card struct {
	ID     string `json:"id"`
	Hidden bool   `json:"hidden"`
}

// Validate checks a layout before it is stored. Card ids are not checked
// against the client's card list on purpose: the client drops ids it does
// not know, so adding or retiring a card needs no server release.
func Validate(cards []Card) error {
	if len(cards) > MaxCards {
		return fmt.Errorf("%w: at most %d cards", ErrInvalidLayout, MaxCards)
	}
	seen := make(map[string]bool, len(cards))
	for _, c := range cards {
		if !cardID.MatchString(c.ID) {
			return fmt.Errorf("%w: card id %q", ErrInvalidLayout, c.ID)
		}
		if seen[c.ID] {
			return fmt.Errorf("%w: card %q listed twice", ErrInvalidLayout, c.ID)
		}
		seen[c.ID] = true
	}
	return nil
}

// Store reads and writes saved layouts.
type Store struct{ DB *pgxpool.Pool }

// Get returns the member's saved layout, or nil when none is saved (the
// client then shows its default layout).
func (s Store) Get(ctx context.Context, userID, projectID string) ([]Card, error) {
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT cards FROM go_dashboard_layouts WHERE user_id=$1 AND project_id=$2`, userID, projectID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load dashboard layout: %w", err)
	}
	var cards []Card
	if err := json.Unmarshal(raw, &cards); err != nil {
		return nil, fmt.Errorf("decode dashboard layout: %w", err)
	}
	return cards, nil
}

// Save stores the member's layout. An empty layout deletes the saved one, so
// "Reset" brings back the default, including cards added later.
func (s Store) Save(ctx context.Context, userID, projectID string, cards []Card) error {
	if err := Validate(cards); err != nil {
		return err
	}
	if len(cards) == 0 {
		if _, err := s.DB.Exec(ctx, `DELETE FROM go_dashboard_layouts WHERE user_id=$1 AND project_id=$2`, userID, projectID); err != nil {
			return fmt.Errorf("reset dashboard layout: %w", err)
		}
		return nil
	}
	raw, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("encode dashboard layout: %w", err)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO go_dashboard_layouts (user_id, project_id, cards) VALUES ($1,$2,$3)
		ON CONFLICT (user_id, project_id) DO UPDATE SET cards=EXCLUDED.cards, updated_at=now()`, userID, projectID, raw); err != nil {
		return fmt.Errorf("save dashboard layout: %w", err)
	}
	return nil
}
