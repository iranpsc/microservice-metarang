package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"metarang/dynasty-service/internal/client"
	pb "metarang/shared/pb/levels"
)

type fakeLevelServiceClient struct {
	pb.LevelServiceClient
	resp    *pb.UserLevelResponse
	err     error
	gemResp *pb.LevelGemResponse
	gemErr  error
	last    *pb.GetUserLevelRequest
	lastGem *pb.GetLevelGemRequest
}

func (f *fakeLevelServiceClient) GetUserLevel(_ context.Context, in *pb.GetUserLevelRequest, _ ...grpc.CallOption) (*pb.UserLevelResponse, error) {
	f.last = in
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func (f *fakeLevelServiceClient) GetLevelGem(_ context.Context, in *pb.GetLevelGemRequest, _ ...grpc.CallOption) (*pb.LevelGemResponse, error) {
	f.lastGem = in
	if f.gemErr != nil {
		return nil, f.gemErr
	}
	return f.gemResp, nil
}

func TestLevelsClient_GetUserLevel_Success(t *testing.T) {
	fake := &fakeLevelServiceClient{
		resp: &pb.UserLevelResponse{
			LatestLevel: &pb.Level{Id: 2, Name: "Silver"},
			UserScore:   10,
		},
	}
	c := client.NewLevelsClientFromGRPC(fake, nil)

	resp, err := c.GetUserLevel(context.Background(), 99)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, uint64(99), fake.last.UserId)
	assert.Equal(t, "Silver", resp.LatestLevel.Name)
	assert.NoError(t, c.Close())
}

func TestLevelsClient_GetUserLevel_Error(t *testing.T) {
	fake := &fakeLevelServiceClient{err: errors.New("boom")}
	c := client.NewLevelsClientFromGRPC(fake, nil)
	_, err := c.GetUserLevel(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user level")
}

func TestLevelsClient_GetLevelGem_Success(t *testing.T) {
	fake := &fakeLevelServiceClient{
		gemResp: &pb.LevelGemResponse{
			Gem: &pb.LevelGem{Id: 5, Name: "Ruby", PngFile: "ruby.png"},
		},
	}
	c := client.NewLevelsClientFromGRPC(fake, nil)
	gem, err := c.GetLevelGem(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, gem)
	assert.Equal(t, uint64(3), fake.lastGem.LevelId)
	assert.Equal(t, "Ruby", gem.Name)
	assert.Equal(t, "ruby.png", gem.PngFile)
}

func TestLevelsClient_GetLevelGem_Error(t *testing.T) {
	fake := &fakeLevelServiceClient{gemErr: errors.New("gem boom")}
	c := client.NewLevelsClientFromGRPC(fake, nil)
	_, err := c.GetLevelGem(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get level gem")
}
