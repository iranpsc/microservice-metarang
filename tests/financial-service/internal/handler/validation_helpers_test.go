package handler_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"

	"metarang/financial-service/internal/handler"
)

func TestGetLocaleFromContext(t *testing.T) {
	t.Run("defaults to en", func(t *testing.T) {
		if got := handler.GetLocaleFromContext(context.Background()); got != "en" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("fa from grpcgateway prefix", func(t *testing.T) {
		md := metadata.Pairs("grpcgateway-accept-language", "fa-IR,en;q=0.9")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := handler.GetLocaleFromContext(ctx); got != "fa" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("fa from accept-language", func(t *testing.T) {
		md := metadata.Pairs("accept-language", "fa")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := handler.GetLocaleFromContext(ctx); got != "fa" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("en from primary tag", func(t *testing.T) {
		md := metadata.Pairs("accept-language", "en-US")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := handler.GetLocaleFromContext(ctx); got != "en" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("empty metadata values default to en", func(t *testing.T) {
		md := metadata.MD{}
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := handler.GetLocaleFromContext(ctx); got != "en" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("Accept-Language header key", func(t *testing.T) {
		md := metadata.Pairs("Accept-Language", "fa")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := handler.GetLocaleFromContext(ctx); got != "fa" {
			t.Fatalf("got %q", got)
		}
	})
}
