package sam

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

type distributedTurns struct {
	db    *pgxpool.Pool
	locks *pgxpool.Pool
}

func turnKey(sessionID string) string { return "sam-turn:" + sessionID }

func turnChannel(sessionID string) string {
	digest := sha256.Sum256([]byte(sessionID))
	return fmt.Sprintf("sam_turn_%x", digest[:24])
}

func (t distributedTurns) begin(parent context.Context, sessionID string) (context.Context, func(), error) {
	conn, release, err := pgdb.TrySessionLock(parent, t.locks, turnKey(sessionID))
	if err != nil || conn == nil {
		return nil, nil, err
	}
	channel := turnChannel(sessionID)
	// The channel contains only the fixed prefix and lowercase hex.
	if _, err := conn.Exec(parent, "LISTEN "+channel); err != nil {
		release()
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	listening, stopListening := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// A lost connection also cancels the turn since it owned the lock.
		_, _ = conn.Conn().WaitForNotification(listening)
		cancel()
	}()
	return ctx, func() {
		stopListening()
		<-done
		release()
	}, nil
}

func (t distributedTurns) rewind(parent context.Context, sessionID string) (func(), error) {
	ctx, cancel := context.WithTimeout(parent, rewindWait)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		// Rewind uses a short lock from the application pool. It must be able
		// to signal the turn even when all streaming lock slots are occupied.
		_, release, err := pgdb.TrySessionLock(ctx, t.db, turnKey(sessionID))
		if err != nil || release != nil {
			return release, err
		}
		if _, err := t.db.Exec(ctx, "SELECT pg_notify($1, '')", turnChannel(sessionID)); err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
