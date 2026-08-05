package handler

import (
	"fmt"
	"os"
	"strings"

	commonpb "metarang/shared/pb/common"
	trainingpb "metarang/shared/pb/training"
	"metarang/training-service/internal/service"
)

func VideoDetailsToProto(video *service.VideoDetails) (*trainingpb.VideoResponse, error) {
	if video == nil || video.Video == nil {
		return nil, fmt.Errorf("invalid video data")
	}

	resp := &trainingpb.VideoResponse{
		Id:          video.Video.ID,
		Title:       video.Video.Title,
		Slug:        stringValue(video.Video.Slug),
		Description: video.Video.Description,
		FileName:    video.Video.FileName,
		CreatorCode: video.Video.CreatorCode,
		CreatedAt:   video.CreatedAtJalali,
	}

	appURL := strings.TrimSuffix(os.Getenv("APP_URL"), "/")

	if video.Video.Image != "" {
		if strings.HasPrefix(video.Video.Image, "http://") || strings.HasPrefix(video.Video.Image, "https://") {
			resp.ImageUrl = video.Video.Image
		} else if appURL != "" {
			imagePath := strings.TrimPrefix(video.Video.Image, "/")
			if !strings.HasPrefix(imagePath, "uploads/") {
				imagePath = "uploads/" + imagePath
			}
			resp.ImageUrl = fmt.Sprintf("%s/%s", appURL, imagePath)
		} else {
			imagePath := strings.TrimPrefix(video.Video.Image, "/")
			if !strings.HasPrefix(imagePath, "uploads/") {
				imagePath = "uploads/" + imagePath
			}
			resp.ImageUrl = "/" + imagePath
		}
	} else {
		resp.ImageUrl = ""
	}

	if video.Video.FileName != "" {
		videoPath := "/uploads/videos/" + video.Video.FileName
		if appURL != "" {
			resp.VideoUrl = fmt.Sprintf("%s%s", appURL, videoPath)
		} else {
			resp.VideoUrl = videoPath
		}
	} else {
		resp.VideoUrl = ""
	}

	if video.Creator != nil {
		resp.Creator = &commonpb.UserBasic{
			Id:    video.Creator.ID,
			Name:  video.Creator.Name,
			Code:  video.Creator.Code,
			Email: video.Creator.Email,
		}
		if video.Creator.ProfilePhoto != "" {
			resp.Creator.ProfilePhoto = video.Creator.ProfilePhoto
		}
	}

	if video.Category != nil {
		resp.Category = &trainingpb.CategoryInfo{
			Id:   video.Category.ID,
			Name: video.Category.Name,
			Slug: video.Category.Slug,
		}
	}
	if video.SubCategory != nil {
		resp.SubCategory = &trainingpb.SubCategoryInfo{
			Id:   video.SubCategory.ID,
			Name: video.SubCategory.Name,
			Slug: video.SubCategory.Slug,
		}
	}

	if video.Stats != nil {
		resp.Stats = &trainingpb.VideoStats{
			ViewsCount:    video.Stats.ViewsCount,
			LikesCount:    video.Stats.LikesCount,
			DislikesCount: video.Stats.DislikesCount,
			CommentsCount: video.Stats.CommentsCount,
		}
	}

	return resp, nil
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
