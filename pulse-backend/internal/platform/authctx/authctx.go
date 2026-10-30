// Package authctx carries the authenticated user's ID through a request context.
package authctx

import "context"

type ctxKey struct{}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// UserID returns the authenticated user's ID. Only call it from handlers
// wrapped by the auth middleware; it returns 0 otherwise.
func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(ctxKey{}).(int64)
	return id
}
