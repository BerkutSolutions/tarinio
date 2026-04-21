package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	rediscoord "waf/control-plane/internal/coordination/redis"
	"waf/control-plane/internal/telemetry"
)

type DistributedCoordinator interface {
	Enabled() bool
	NodeID() string
	OperationTTL() time.Duration
	LeaderTTL() time.Duration
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) error
	TryRunLeader(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) (bool, error)
}

type noopDistributedCoordinator struct{}

func NewNoopDistributedCoordinator() DistributedCoordinator {
	return noopDistributedCoordinator{}
}

func (noopDistributedCoordinator) Enabled() bool               { return false }
func (noopDistributedCoordinator) NodeID() string              { return "single-node" }
func (noopDistributedCoordinator) OperationTTL() time.Duration { return 2 * time.Minute }
func (noopDistributedCoordinator) LeaderTTL() time.Duration    { return 30 * time.Second }
func (noopDistributedCoordinator) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) error {
	return fn(ctx)
}
func (noopDistributedCoordinator) TryRunLeader(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) (bool, error) {
	return true, fn(ctx)
}

type RedisDistributedCoordinator struct {
	backend      *rediscoord.Backend
	nodeID       string
	operationTTL time.Duration
	leaderTTL    time.Duration
}

func NewRedisDistributedCoordinator(backend *rediscoord.Backend, nodeID string, operationTTL time.Duration, leaderTTL time.Duration) DistributedCoordinator {
	if backend == nil || backend.Client() == nil {
		return NewNoopDistributedCoordinator()
	}
	if operationTTL <= 0 {
		operationTTL = 2 * time.Minute
	}
	if leaderTTL <= 0 {
		leaderTTL = 30 * time.Second
	}
	return &RedisDistributedCoordinator{
		backend:      backend,
		nodeID:       nodeID,
		operationTTL: operationTTL,
		leaderTTL:    leaderTTL,
	}
}

func (c *RedisDistributedCoordinator) Enabled() bool {
	return c != nil && c.backend != nil && c.backend.Client() != nil
}

func (c *RedisDistributedCoordinator) NodeID() string {
	if c == nil || c.nodeID == "" {
		return "unknown-node"
	}
	return c.nodeID
}

func (c *RedisDistributedCoordinator) OperationTTL() time.Duration {
	if c == nil || c.operationTTL <= 0 {
		return 2 * time.Minute
	}
	return c.operationTTL
}

func (c *RedisDistributedCoordinator) LeaderTTL() time.Duration {
	if c == nil || c.leaderTTL <= 0 {
		return 30 * time.Second
	}
	return c.leaderTTL
}

func (c *RedisDistributedCoordinator) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) error {
	if !c.Enabled() {
		return fn(ctx)
	}
	start := time.Now()
	lock, err := c.acquireWithRetry(ctx, key, ttl, ttl)
	if err != nil {
		if errors.Is(err, rediscoord.ErrLockNotAcquired) {
			telemetry.Default().RecordHALock(c.NodeID(), key, "busy", time.Since(start))
			return fmt.Errorf("ha lock busy for %s", key)
		}
		telemetry.Default().RecordHALock(c.NodeID(), key, "error", time.Since(start))
		return err
	}
	telemetry.Default().RecordHALock(c.NodeID(), key, "acquired", time.Since(start))
	return runWithRefreshedLock(ctx, lock, ttl, fn)
}

func (c *RedisDistributedCoordinator) TryRunLeader(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error) (bool, error) {
	if !c.Enabled() {
		return true, fn(ctx)
	}
	start := time.Now()
	lock, err := c.backend.Locks().Acquire(ctx, key, ttl)
	if err != nil {
		if errors.Is(err, rediscoord.ErrLockNotAcquired) {
			telemetry.Default().RecordLeaderRun(c.NodeID(), key, "skipped")
			return false, nil
		}
		telemetry.Default().RecordLeaderRun(c.NodeID(), key, "error")
		return false, err
	}
	telemetry.Default().RecordHALock(c.NodeID(), key, "acquired", time.Since(start))
	if err := runWithRefreshedLock(ctx, lock, ttl, fn); err != nil {
		telemetry.Default().RecordLeaderRun(c.NodeID(), key, "error")
		return true, err
	}
	telemetry.Default().RecordLeaderRun(c.NodeID(), key, "ran")
	return true, nil
}

func (c *RedisDistributedCoordinator) acquireWithRetry(ctx context.Context, key string, ttl time.Duration, maxWait time.Duration) (rediscoord.HeldLock, error) {
	if maxWait <= 0 {
		maxWait = 15 * time.Second
	}
	if maxWait > 2*time.Minute {
		maxWait = 2 * time.Minute
	}
	deadline := time.Now().Add(maxWait)
	backoff := 250 * time.Millisecond
	for {
		lock, err := c.backend.Locks().Acquire(ctx, key, ttl)
		if err == nil {
			return lock, nil
		}
		if !errors.Is(err, rediscoord.ErrLockNotAcquired) {
			return nil, err
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if time.Now().After(deadline) {
			return nil, err
		}

		wait := backoff
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		if backoff < 2*time.Second {
			backoff *= 2
			if backoff > 2*time.Second {
				backoff = 2 * time.Second
			}
		}
	}
}

func runWithRefreshedLock(ctx context.Context, lock rediscoord.HeldLock, ttl time.Duration, fn func(context.Context) error) error {
	refreshCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(ttl / 3)
		defer ticker.Stop()
		for {
			select {
			case <-refreshCtx.Done():
				errCh <- nil
				return
			case <-ticker.C:
				if err := lock.Refresh(context.Background(), ttl); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	runErr := fn(ctx)
	cancel()
	refreshErr := <-errCh
	releaseErr := lock.Release(context.Background())

	if runErr != nil {
		return runErr
	}
	if refreshErr != nil {
		return refreshErr
	}
	if releaseErr != nil && !errors.Is(releaseErr, rediscoord.ErrLockNotAcquired) {
		return releaseErr
	}
	return nil
}
