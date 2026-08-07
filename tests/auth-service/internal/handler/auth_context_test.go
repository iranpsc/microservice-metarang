package handler_test

import (
	"context"

	"google.golang.org/grpc/metadata"

	sharedauth "metarang/shared/pkg/auth"
)

func authenticatedContext(userID uint64) context.Context {
	return context.WithValue(context.Background(), sharedauth.UserContextKey{}, &sharedauth.UserContext{
		UserID: userID,
		Email:  "test@example.com",
	})
}

func serviceTokenContext(secret string) context.Context {
	md := metadata.Pairs(sharedauth.ServiceTokenMetadataKey, secret)
	return metadata.NewIncomingContext(context.Background(), md)
}
