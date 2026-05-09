package api

import "context"

type fraudScoreLimiter struct {
	tokens chan struct{}
}

func newFraudScoreLimiter(limit int) *fraudScoreLimiter {
	if limit <= 0 {
		return nil
	}

	return &fraudScoreLimiter{
		tokens: make(chan struct{}, limit),
	}
}

func (l *fraudScoreLimiter) Acquire(ctx context.Context) error {
	if l == nil {
		return nil
	}

	select {
	case l.tokens <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *fraudScoreLimiter) Release() {
	if l == nil {
		return
	}

	<-l.tokens
}
