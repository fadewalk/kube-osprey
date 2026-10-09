package mcpserver

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/model"
)

type userContextKey struct{}

func userFromGin(c *gin.Context) (model.User, bool) {
	raw, ok := c.Get("user")
	if !ok {
		return model.User{}, false
	}
	user, ok := raw.(model.User)
	return user, ok
}

func userFromContext(ctx context.Context) (model.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(model.User)
	return user, ok
}

// mcpRateLimit returns the runtime per-user tool call budget per minute.
func mcpRateLimit() int {
	setting, err := model.GetGeneralSetting()
	if err != nil || setting.MCPRateLimitPerMinute <= 0 {
		return 60
	}
	return setting.MCPRateLimitPerMinute
}

// rateLimiter is a per-user sliding window; entries outside the window are
// dropped lazily on access, so memory is bounded by the active user count.
type rateLimiter struct {
	mu   sync.Mutex
	hits map[uint][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{hits: make(map[uint][]time.Time)}
}

func (l *rateLimiter) allow(userID uint, limit int) bool {
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	recent := l.hits[userID][:0]
	for _, ts := range l.hits[userID] {
		if ts.After(windowStart) {
			recent = append(recent, ts)
		}
	}
	if len(recent) >= limit {
		l.hits[userID] = recent
		return false
	}
	l.hits[userID] = append(recent, now)
	return true
}
