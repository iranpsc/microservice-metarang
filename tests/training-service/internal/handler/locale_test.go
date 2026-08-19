package handler_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/handler"
	"metarang/training-service/internal/service"
	"metarang/training-service/tests/internal/testutil"
)

func TestLocale_InvalidMetadataFallsThrough(t *testing.T) {
	handler.SetProjectLocale("en")
	svc := service.NewVideoService(&testutil.MockVideoRepo{}, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterVideoHandler(s, svc)
	})
	defer cleanup()
	client := trainingpb.NewVideoServiceClient(conn)

	md := metadata.Pairs("locale", "de", "accept-language", "en-US")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := client.SearchVideos(ctx, &trainingpb.SearchVideosRequest{Query: ""})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestLocale_SearchVideosUsesMetadataLocale(t *testing.T) {
	handler.SetProjectLocale("en")
	svc := service.NewVideoService(&testutil.MockVideoRepo{}, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterVideoHandler(s, svc)
	})
	defer cleanup()
	client := trainingpb.NewVideoServiceClient(conn)

	md := metadata.Pairs("locale", "fa")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := client.SearchVideos(ctx, &trainingpb.SearchVideosRequest{Query: ""})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
	if st.Message() == "" {
		t.Fatal("expected validation message")
	}

	md = metadata.Pairs("accept-language", "fa-IR,en")
	ctx = metadata.NewOutgoingContext(context.Background(), md)
	_, err = client.SearchVideos(ctx, &trainingpb.SearchVideosRequest{Query: ""})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("accept-language got %v", err)
	}
}

func TestLocale_SetProjectLocaleFallback(t *testing.T) {
	handler.SetProjectLocale("  FA  ")
	t.Cleanup(func() { handler.SetProjectLocale("en") })

	svc := service.NewVideoService(&testutil.MockVideoRepo{}, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterVideoHandler(s, svc)
	})
	defer cleanup()
	client := trainingpb.NewVideoServiceClient(conn)
	_, err := client.SearchVideos(context.Background(), &trainingpb.SearchVideosRequest{Query: ""})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}

	handler.SetProjectLocale("nope")
	_, err = client.SearchVideos(context.Background(), &trainingpb.SearchVideosRequest{Query: ""})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("invalid locale fallback got %v", err)
	}
}

func TestLocale_ProjectLocaleEnv(t *testing.T) {
	handler.SetProjectLocale("")
	t.Setenv("PROJECT_LOCALE", "fa")
	svc := service.NewVideoService(&testutil.MockVideoRepo{}, &testutil.MockCategoryRepo{}, &testutil.MockUserRepo{})
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterVideoHandler(s, svc)
	})
	defer cleanup()
	client := trainingpb.NewVideoServiceClient(conn)
	_, err := client.SearchVideos(context.Background(), &trainingpb.SearchVideosRequest{Query: ""})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}
