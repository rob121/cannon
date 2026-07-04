package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey int

const (
	ctxKeyCredential contextKey = iota + 1
	ctxKeyUserID
	ctxKeyViewerGroups
)

// WithCredentialID stores the validated API credential id on the context.
func WithCredentialID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, ctxKeyCredential, id)
}

// CredentialID returns the API credential id when present.
func CredentialID(ctx context.Context) (uint, bool) {
	v, ok := ctx.Value(ctxKeyCredential).(uint)
	return v, ok && v > 0
}

// WithUserID stores the JWT-authenticated user id.
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// UserID returns the JWT user id when present.
func UserID(ctx context.Context) (uint, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(uint)
	return v, ok && v > 0
}

// WithViewerGroups caches resolved viewer group ids for the request.
func WithViewerGroups(ctx context.Context, ids []uint) context.Context {
	return context.WithValue(ctx, ctxKeyViewerGroups, ids)
}

// ViewerGroups returns cached viewer groups when set.
func ViewerGroups(ctx context.Context) ([]uint, bool) {
	v, ok := ctx.Value(ctxKeyViewerGroups).([]uint)
	return v, ok
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
