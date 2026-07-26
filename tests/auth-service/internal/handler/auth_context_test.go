package handler_test

import (
	"context"

	sharedauth "metarang/shared/pkg/auth"
)

func authenticatedContext(userID uint64) context.Context {
	return context.WithValue(context.Background(), sharedauth.UserContextKey{}, &sharedauth.UserContext{
		UserID: userID,
		Email:  "test@example.com",
	})
}
