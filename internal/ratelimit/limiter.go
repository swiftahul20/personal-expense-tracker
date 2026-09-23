package ratelimit

import (
	"sync"
	"time"
)

// type
type visitor struct {
	attempts  int
	resetTime time.Time
}

type Limiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	maxTries int
	window   time.Duration
}

// ========================

func New(maxTries int, window time.Duration) *Limiter {
	return &Limiter{
		visitors: make(map[string]*visitor),
		maxTries: maxTries,
		window:   window,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, exists := l.visitors[key]

	if !exists || now.After(v.resetTime) {
		l.visitors[key] = &visitor{
			attempts:  1,
			resetTime: now.Add(l.window),
		}
		return true
	}

	if v.attempts >= l.maxTries {
		return false
	}

	v.attempts++
	return true
}
