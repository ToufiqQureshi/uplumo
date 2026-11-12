package pgdb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TrySessionLock holds a database-wide lock until release. A nil connection
// means another request owns it. Use a separate pool for long-lived locks so
// the locked work can still use the application pool.
func TrySessionLock(ctx context.Context, pool *pgxpool.Pool, key string) (*pgxpool.Conn, func(), error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtextextended($1, 0))", key).Scan(&locked); err != nil {
		closeSession(conn)
		return nil, nil, fmt.Errorf("claim session lock: %w", err)
	}
	if !locked {
		conn.Release()
		return nil, nil, nil
	}
	// End the session to release the lock and discard any buffered LISTEN
	// notifications before another request can borrow a connection.
	return conn, func() { closeSession(conn) }, nil
}

func closeSession(conn *pgxpool.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = conn.Hijack().Close(ctx)
}
