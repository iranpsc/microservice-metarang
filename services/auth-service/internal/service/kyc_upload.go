package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"metarang/auth-service/internal/models"
)

const (
	kycMelliCardMaxSize = 5 * 1024 * 1024
	kycUploadPath       = "/uploads/kyc"
)

// KYCSubmission holds raw KYC form data including file payloads for upload processing.
type KYCSubmission struct {
	Fname                string
	Lname                string
	MelliCode            string
	Birthdate            string
	Province             string
	Gender               string
	VerifyTextID         uint64
	MelliCardData        []byte
	MelliCardFilename    string
	MelliCardContentType string
	VideoPath            string
	VideoName            string
}

func (s *kycService) SubmitKYC(ctx context.Context, userID uint64, input KYCSubmission) (*models.KYC, error) {
	if err := validateMelliCardFile(input.MelliCardData, input.MelliCardFilename, input.MelliCardContentType); err != nil {
		return nil, err
	}
	if input.VideoPath == "" || input.VideoName == "" {
		return nil, ErrVideoRequired
	}
	if s.fileStorage == nil {
		return nil, ErrStorageUnavailable
	}

	melliCardURL, err := s.fileStorage.UploadChunk(
		ctx,
		NewUploadID("kyc_melli_card", userID),
		kycUploadPath,
		input.MelliCardFilename,
		input.MelliCardContentType,
		input.MelliCardData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload melli_card: %w", err)
	}

	videoURL, err := s.promoteKYCVideo(ctx, input.VideoPath, input.VideoName)
	if err != nil {
		return nil, fmt.Errorf("failed to promote kyc video: %w", err)
	}

	melliCardURL = PrependGatewayURL(s.apiGatewayURL, melliCardURL)
	videoURL = PrependGatewayURL(s.apiGatewayURL, videoURL)

	return s.UpdateKYC(
		ctx,
		userID,
		input.Fname,
		input.Lname,
		input.MelliCode,
		input.Birthdate,
		input.Province,
		melliCardURL,
		videoURL,
		input.VerifyTextID,
		input.Gender,
	)
}

func validateMelliCardFile(data []byte, filename, contentType string) error {
	if len(data) == 0 {
		return ErrMelliCardRequired
	}
	if filename == "" {
		return ErrMelliCardFilenameRequired
	}
	if contentType == "" {
		return ErrMelliCardContentTypeRequired
	}
	if len(data) > kycMelliCardMaxSize {
		return ErrMelliCardTooLarge
	}

	contentType = strings.ToLower(contentType)
	if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/jpg" {
		return ErrInvalidMelliCardType
	}

	filenameLower := strings.ToLower(filename)
	if !strings.HasSuffix(filenameLower, ".png") && !strings.HasSuffix(filenameLower, ".jpg") && !strings.HasSuffix(filenameLower, ".jpeg") {
		return ErrInvalidMelliCardExtension
	}
	return nil
}

func (s *kycService) promoteKYCVideo(ctx context.Context, videoPath, videoName string) (string, error) {
	fileData, contentType, err := s.readStagedVideo(ctx, videoPath, videoName)
	if err != nil {
		return "", err
	}

	finalName := filepath.Base(videoName)
	return s.fileStorage.UploadChunk(
		ctx,
		fmt.Sprintf("kyc_video_%d", time.Now().UnixNano()),
		kycUploadPath,
		finalName,
		contentType,
		fileData,
	)
}

func (s *kycService) readStagedVideo(ctx context.Context, videoPath, videoName string) ([]byte, string, error) {
	contentType := contentTypeFromFilename(videoName)
	for _, sourcePath := range stagedVideoPaths(videoPath, videoName) {
		data, ct, err := s.fileStorage.ReadFile(ctx, sourcePath)
		if err != nil {
			continue
		}
		if ct != "" {
			contentType = ct
		}
		return data, contentType, nil
	}
	return nil, "", fmt.Errorf("staged video not found at path %q name %q", videoPath, videoName)
}
