package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type contextKey string

const (
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUsername  contextKey = "username"
	ContextKeyRoleID    contextKey = "role_id"
	ContextKeySessionID contextKey = "session_id"
)

// TokenValidator is a minimal interface the Authenticator middleware relies on.
// It is satisfied by auth.Service without creating an import cycle.
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, tokenStr string) (*Claims, error)
}

// Claims holds the decoded token values placed in the request context.
type Claims struct {
	UserID    uuid.UUID
	Username  string
	RoleID    uuid.UUID
	SessionID uuid.UUID
}

// Authenticator is a middleware that validates JWT Bearer tokens.
func Authenticator(validator TokenValidator, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				WriteError(w, apperr.Unauthorized("missing authorization header"))
				return
			}

			claims, err := validator.ValidateAccessToken(r.Context(), token)
			if err != nil {
				WriteError(w, apperr.Unauthorized("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyUsername, claims.Username)
			ctx = context.WithValue(ctx, ContextKeyRoleID, claims.RoleID)
			ctx = context.WithValue(ctx, ContextKeySessionID, claims.SessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromCtx extracts the authenticated user ID from context.
func UserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ContextKeyUserID).(uuid.UUID)
	return v, ok
}

// UsernameFromCtx extracts the authenticated username from context.
func UsernameFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyUsername).(string)
	return v, ok
}

// RoleIDFromCtx extracts the role ID from context.
func RoleIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ContextKeyRoleID).(uuid.UUID)
	return v, ok
}

// SessionIDFromCtx extracts the session ID from context.
func SessionIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ContextKeySessionID).(uuid.UUID)
	return v, ok
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
