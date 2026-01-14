package ratelimiter

import "context"

// 定接口
type Limit interface {
	Allow() bool
	AllowCtx(ctx context.Context) bool
}

type noLimit struct{}

func (n *noLimit) Allow() bool {
	return true
}

func (n *noLimit) AllowCtx(ctx context.Context) bool {
	return true
}

type allLimit struct{}

func (a *allLimit) Allow() bool {
	return false
}

func (a *allLimit) AllowCtx(ctx context.Context) bool {
	return false
}
