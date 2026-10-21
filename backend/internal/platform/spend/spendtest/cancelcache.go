package spendtest

import (
	"context"
	"sync"
	"time"
)

// CancelOnReadCache cancels the caller during its first cache read. It always
// returns a miss; Key records the first read so a test can drain a shared flight
// before asserting that no asynchronous provider work escaped cancellation.
// This is deterministic test support, never a production cache.
type CancelOnReadCache struct {
	Cancel context.CancelFunc
	Key    string
	once   sync.Once
}

// Get simulates a cache read interrupted by caller cancellation.
func (c *CancelOnReadCache) Get(_ context.Context, key string) ([]byte, error) {
	c.once.Do(func() { c.Key = key; c.Cancel() })
	return nil, context.Canceled
}

// Set intentionally stores nothing.
func (*CancelOnReadCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
