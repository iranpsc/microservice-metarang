package service_test

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"metarang/auth-service/internal/service"
	commercialpb "metarang/shared/pb/commercial"
	levelspb "metarang/shared/pb/levels"
)

type liveLevelServer struct {
	levelspb.UnimplementedLevelServiceServer
}

func (liveLevelServer) GetUserLevel(context.Context, *levelspb.GetUserLevelRequest) (*levelspb.UserLevelResponse, error) {
	return &levelspb.UserLevelResponse{
		LatestLevel: &levelspb.Level{
			Id: 1, Name: "L", Score: 10, Slug: "l",
			GeneralInfo: &levelspb.LevelGeneralInfo{Description: "desc"},
		},
		ScorePercentageToNextLevel: 42,
	}, nil
}

type liveWalletServer struct {
	commercialpb.UnimplementedWalletServiceServer
}

func (liveWalletServer) GetWallet(context.Context, *commercialpb.GetWalletRequest) (*commercialpb.WalletResponse, error) {
	return &commercialpb.WalletResponse{
		Psc: "1", Irr: "2", Red: "3", Blue: "4", Yellow: "5", Satisfaction: "6", Effect: 7.5,
	}, nil
}

func (liveWalletServer) CreateWallet(context.Context, *commercialpb.CreateWalletRequest) (*commercialpb.WalletResponse, error) {
	return &commercialpb.WalletResponse{Psc: "0"}, nil
}

type liveUserVarServer struct {
	commercialpb.UnimplementedUserVariableServiceServer
}

func (liveUserVarServer) CreateUserVariables(context.Context, *commercialpb.CreateUserVariablesRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func TestHelperService_LiveGRPCBackends(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_SECRET", "test-secret")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := "127.0.0.1:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	srv := grpc.NewServer()
	levelspb.RegisterLevelServiceServer(srv, liveLevelServer{})
	commercialpb.RegisterWalletServiceServer(srv, liveWalletServer{})
	commercialpb.RegisterUserVariableServiceServer(srv, liveUserVarServer{})
	go func() { _ = srv.Serve(ln) }()
	defer srv.Stop()

	hs := service.NewHelperService(addr, "", addr)
	defer func() { _ = hs.Close() }()
	ctx := context.Background()

	pct, err := hs.GetScorePercentageToNextLevel(ctx, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 42.0, pct)

	lvl, err := hs.GetUserLevel(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, lvl)
	require.Equal(t, "desc", lvl.Description)

	w, err := hs.GetUserWallet(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "1", w.Psc)
	require.Equal(t, 7.5, w.Effect)

	require.NoError(t, hs.CreateWallet(ctx, 1))
	require.NoError(t, hs.CreateUserVariables(ctx, 1))
}

func TestHelperService_WalletReconnect(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_SECRET", "test-secret")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	_ = ln.Close()

	// Dial fails at construction; addr retained for lazy reconnect.
	hs := service.NewHelperService("", "", addr)
	defer func() { _ = hs.Close() }()

	ln, err = net.Listen("tcp", addr)
	require.NoError(t, err)
	srv := grpc.NewServer()
	commercialpb.RegisterWalletServiceServer(srv, liveWalletServer{})
	commercialpb.RegisterUserVariableServiceServer(srv, liveUserVarServer{})
	go func() { _ = srv.Serve(ln) }()
	defer srv.Stop()

	w, err := hs.GetUserWallet(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "1", w.Psc)
}
