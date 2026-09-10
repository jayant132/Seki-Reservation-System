package middleware

import (
	"context"

	"github.com/google/uuid"
)

func UserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxUserID).(uuid.UUID)
	return id
}

func UserRole(ctx context.Context) string {
	role, _ := ctx.Value(ctxRole).(string)
	return role
}
