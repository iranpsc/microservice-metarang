package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UserContextKey is the key for user data in context
type UserContextKey struct{}

// UserContext holds authenticated user information
type UserContext struct {
	UserID uint64
	Email  string
	Token  string
}

// TokenValidator interface for validating tokens
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*UserContext, error)
}

// UnaryServerInterceptor returns a new unary server interceptor for authentication
func UnaryServerInterceptor(validator TokenValidator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication for certain methods (e.g., health checks, public endpoints)
		if shouldSkipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token := extractToken(authHeader[0])
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate token
		userCtx, err := validator.ValidateToken(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("invalid token: %v", err))
		}

		// Add user context
		ctx = context.WithValue(ctx, UserContextKey{}, userCtx)

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new stream server interceptor for authentication
func StreamServerInterceptor(validator TokenValidator) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Skip authentication for certain methods
		if shouldSkipAuth(info.FullMethod) {
			return handler(srv, stream)
		}

		ctx := stream.Context()

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token := extractToken(authHeader[0])
		if token == "" {
			return status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate token
		userCtx, err := validator.ValidateToken(ctx, token)
		if err != nil {
			return status.Error(codes.Unauthenticated, fmt.Sprintf("invalid token: %v", err))
		}

		// Add user context
		ctx = context.WithValue(ctx, UserContextKey{}, userCtx)

		// Wrap stream with new context
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// extractToken extracts the token from "Bearer <token>" format
func extractToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

// shouldSkipAuth checks if authentication should be skipped for a method
func shouldSkipAuth(fullMethod string) bool {
	// List of methods that don't require authentication
	publicMethods := []string{
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
		"/auth.AuthService/Register",
		"/auth.AuthService/Login",
		"/auth.AuthService/Callback",
	}

	for _, method := range publicMethods {
		if fullMethod == method {
			return true
		}
	}
	return false
}

// GetUserFromContext retrieves user context from the context
func GetUserFromContext(ctx context.Context) (*UserContext, error) {
	userCtx, ok := ctx.Value(UserContextKey{}).(*UserContext)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user context not found")
	}
	return userCtx, nil
}

// wrappedServerStream wraps grpc.ServerStream to override context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

