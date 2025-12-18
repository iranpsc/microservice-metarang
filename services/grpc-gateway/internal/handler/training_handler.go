package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metargb/shared/pb/auth"
	commonpb "metargb/shared/pb/common"
	trainingpb "metargb/shared/pb/training"
)

type TrainingHandler struct {
	trainingClient trainingpb.VideoServiceClient
	categoryClient trainingpb.CategoryServiceClient
	commentClient  trainingpb.CommentServiceClient
	replyClient    trainingpb.ReplyServiceClient
	authClient     pb.AuthServiceClient
}

func NewTrainingHandler(trainingConn *grpc.ClientConn, authConn *grpc.ClientConn) *TrainingHandler {
	return &TrainingHandler{
		trainingClient: trainingpb.NewVideoServiceClient(trainingConn),
		categoryClient: trainingpb.NewCategoryServiceClient(trainingConn),
		commentClient:  trainingpb.NewCommentServiceClient(trainingConn),
		replyClient:    trainingpb.NewReplyServiceClient(trainingConn),
		authClient:     pb.NewAuthServiceClient(authConn),
	}
}

// GetVideos handles GET /api/v2/tutorials
func (h *TrainingHandler) GetVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	page, perPage := parsePagination(r, 1, 18)

	grpcReq := &trainingpb.GetVideosRequest{
		Pagination: &commonpb.PaginationRequest{
			Page:    page,
			PerPage: perPage,
		},
	}

	resp, err := h.trainingClient.GetVideos(r.Context(), grpcReq)
	if err != nil {
		writeGRPCErrorTraining(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildVideosResponse(resp))
}

// GetVideo handles GET /api/v2/tutorials/{slug}
func (h *TrainingHandler) GetVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slug := extractSlugFromPathTraining(r.URL.Path, "/api/v2/tutorials/")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}

	var userID uint64
	token := extractTokenFromHeader(r)
	if token != "" {
		if validateResp, err := h.authClient.ValidateToken(r.Context(), &pb.ValidateTokenRequest{Token: token}); err == nil && validateResp.Valid {
			userID = validateResp.UserId
		}
	}

	ipAddress := getIPAddress(r)

	grpcReq := &trainingpb.GetVideoRequest{
		Slug:      slug,
		UserId:    userID,
		IpAddress: ipAddress,
	}

	resp, err := h.trainingClient.GetVideo(r.Context(), grpcReq)
	if err != nil {
		writeGRPCErrorTraining(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildVideoResponse(resp))
}

// SearchVideos handles POST /api/v2/tutorials/search
func (h *TrainingHandler) SearchVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		SearchTerm string `json:"searchTerm"`
	}

	if err := decodeJSONBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	if req.SearchTerm == "" {
		writeValidationError(w, "searchTerm is required")
		return
	}

	page, perPage := parsePagination(r, 1, 18)

	grpcReq := &trainingpb.SearchVideosRequest{
		Query: req.SearchTerm,
		Pagination: &commonpb.PaginationRequest{
			Page:    page,
			PerPage: perPage,
		},
	}

	resp, err := h.trainingClient.SearchVideos(r.Context(), grpcReq)
	if err != nil {
		writeGRPCErrorTraining(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildVideosResponse(resp))
}

// AddInteraction handles POST /api/v2/tutorials/{video}/interactions
func (h *TrainingHandler) AddInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := extractTokenFromHeader(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	validateResp, err := h.authClient.ValidateToken(r.Context(), &pb.ValidateTokenRequest{Token: token})
	if err != nil || !validateResp.Valid {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	videoID := extractIDFromPathWithSuffix(r.URL.Path, "/api/v2/tutorials/", "/interactions")
	if videoID == 0 {
		writeError(w, http.StatusBadRequest, "invalid video ID")
		return
	}

	var req struct {
		Liked bool `json:"liked"`
	}

	// Try query parameter first (per API spec)
	likedStr := r.URL.Query().Get("liked")
	if likedStr != "" {
		req.Liked = likedStr == "1" || likedStr == "true"
	} else {
		// Try JSON body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	ipAddress := getIPAddress(r)

	grpcReq := &trainingpb.AddInteractionRequest{
		VideoId:   videoID,
		UserId:    validateResp.UserId,
		Liked:     req.Liked,
		IpAddress: ipAddress,
	}

	_, err = h.trainingClient.AddInteraction(r.Context(), grpcReq)
	if err != nil {
		writeGRPCErrorTraining(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

// Helper functions
func parsePagination(r *http.Request, defaultPage, defaultPerPage int32) (int32, int32) {
	page := defaultPage
	perPage := defaultPerPage

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 32); err == nil && parsed > 0 {
			page = int32(parsed)
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.ParseInt(pp, 10, 32); err == nil && parsed > 0 {
			perPage = int32(parsed)
		}
	}

	return page, perPage
}

func extractSlugFromPathTraining(path, prefix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.Trim(path, "/")
	return path
}

func extractIDFromPathWithSuffix(path, prefix, suffix string) uint64 {
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	path = strings.Trim(path, "/")
	id, _ := strconv.ParseUint(path, 10, 64)
	return id
}

func getIPAddress(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func buildVideoResponse(video *trainingpb.VideoResponse) map[string]interface{} {
	// Build response matching VideoTutorialResource format
	return map[string]interface{}{
		"id":          video.Id,
		"title":       video.Title,
		"slug":        video.Slug,
		"description": video.Description,
		"image_url":   video.ImageUrl,
		"video_url":   video.VideoUrl,
		"created_at":  video.CreatedAt,
		// Add creator, category, subcategory, stats as needed
	}
}

func buildVideosResponse(resp *trainingpb.VideosResponse) map[string]interface{} {
	videos := make([]map[string]interface{}, 0, len(resp.Videos))
	for _, video := range resp.Videos {
		videos = append(videos, buildVideoResponse(video))
	}

	result := map[string]interface{}{
		"data": videos,
	}

	if resp.Pagination != nil {
		result["meta"] = map[string]interface{}{
			"current_page": resp.Pagination.CurrentPage,
			"per_page":     resp.Pagination.PerPage,
			"total":        resp.Pagination.Total,
			"last_page":    resp.Pagination.LastPage,
		}
	}

	return result
}

func writeGRPCErrorTraining(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch st.Code() {
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.InvalidArgument:
		writeValidationError(w, st.Message())
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	default:
		writeError(w, http.StatusInternalServerError, st.Message())
	}
}
