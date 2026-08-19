// Package worker owns the golangci dashboard's job queue and per-repo/
// branch lock. It uses a self-contained Redis client rather than
// frontiir/cache — see golangci/plans/2026-08-04-golangci-m2-implementation.md
// for why: this dashboard keeps its own infra fully separate from the
// main app's, same reasoning as the M1 DB-isolation decision.
package worker

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// Lock is a Redis-based distributed lock enforcing Rule BR-1
// (golangci/plans/10-business-rules.md): only one scan or fix may run at
// a time per repo+branch.
type Lock struct {
	rdb *redis.Client
}

// NewLock opens the dashboard's own Redis connection (GOLANGCI_REDIS_*),
// independent of the main app's REDIS_* connection.
func NewLock(addr, password string, db int) *Lock {
	return &Lock{rdb: redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})}
}

// Client exposes the underlying redis.Client so the queue can share the
// same connection rather than opening a second one.
func (l *Lock) Client() *redis.Client {
	return l.rdb
}

// TryLock claims key for ttl. claimed is false (nil error) if another
// holder already owns it.
func (l *Lock) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return l.rdb.SetNX(ctx, key, "1", ttl).Result()
}

// Unlock releases key immediately.
func (l *Lock) Unlock(ctx context.Context, key string) error {
	return l.rdb.Del(ctx, key).Err()
}

// LockKey returns the per-repo/branch lock key for Rule BR-1.
func LockKey(repoRef, branch string) string {
	return "golangci:lock:" + repoRef + ":" + branch
}
