package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	socialpb "metarang/shared/pb/social"
	"metarang/shared/pkg/sentry"
	"metarang/social-service/internal/middleware"
)

const followListPerPage int32 = 10

// followAPI is the subset of follow RPCs used by the HTTP layer.
type followAPI interface {
	GetFollowers(context.Context, *socialpb.GetFollowersRequest) (*socialpb.GetFollowersResponse, error)
	GetFollowing(context.Context, *socialpb.GetFollowingRequest) (*socialpb.GetFollowingResponse, error)
	Follow(context.Context, *socialpb.FollowRequest) (*emptypb.Empty, error)
	Unfollow(context.Context, *socialpb.UnfollowRequest) (*emptypb.Empty, error)
	Remove(context.Context, *socialpb.RemoveRequest) (*emptypb.Empty, error)
}

// challengeAPI is the subset of challenge RPCs used by the HTTP layer.
type challengeAPI interface {
	GetTimings(context.Context, *socialpb.GetTimingsRequest) (*socialpb.GetTimingsResponse, error)
	GetQuestion(context.Context, *socialpb.GetQuestionRequest) (*socialpb.GetQuestionResponse, error)
	SubmitAnswer(context.Context, *socialpb.SubmitAnswerRequest) (*socialpb.SubmitAnswerResponse, error)
	GetAdvertisement(context.Context, *socialpb.GetAdvertisementRequest) (*socialpb.GetAdvertisementResponse, error)
}

// HTTPSocialHandler serves Kong-facing REST routes for social-service.
type HTTPSocialHandler struct {
	follow    followAPI
	challenge challengeAPI
}

// NewHTTPSocialHandler wraps the gRPC handlers for local HTTP use.
func NewHTTPSocialHandler(follow followAPI, challenge challengeAPI) *HTTPSocialHandler {
	return &HTTPSocialHandler{follow: follow, challenge: challenge}
}

// RegisterHTTPRoutes registers social REST routes and /health.
func (h *HTTPSocialHandler) RegisterHTTPRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("/api/challenge/timings", authMiddleware(http.HandlerFunc(h.GetTimings)))
	mux.Handle("/api/challenge/question", authMiddleware(http.HandlerFunc(h.GetQuestion)))
	mux.Handle("/api/challenge/answer", authMiddleware(http.HandlerFunc(h.SubmitAnswer)))
	mux.Handle("/api/challenge/advertisement", authMiddleware(http.HandlerFunc(h.GetAdvertisement)))
	mux.Handle("/api/followers", authMiddleware(http.HandlerFunc(h.GetFollowers)))
	mux.Handle("/api/following", authMiddleware(http.HandlerFunc(h.GetFollowing)))
	mux.Handle("/api/follow/", authMiddleware(http.HandlerFunc(h.Follow)))
	mux.Handle("/api/unfollow/", authMiddleware(http.HandlerFunc(h.Unfollow)))
	mux.Handle("/api/remove/", authMiddleware(http.HandlerFunc(h.Remove)))
}

// StartHTTPServer starts the public HTTP server (behind Kong).
func StartHTTPServer(
	httpHandler *HTTPSocialHandler,
	port string,
	authMiddleware func(http.Handler) http.Handler,
) error {
	mux := http.NewServeMux()
	httpHandler.RegisterHTTPRoutes(mux, authMiddleware)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: sentry.HTTPMiddleware(mux),
	}
	return server.ListenAndServe()
}

func (h *HTTPSocialHandler) getUserIDFromRequest(r *http.Request) (uint64, error) {
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		return 0, err
	}
	return userCtx.UserID, nil
}

// GetFollowers handles GET /api/followers
func (h *HTTPSocialHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.follow.GetFollowers(r.Context(), &socialpb.GetFollowersRequest{UserId: userID})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildFollowListHTTPResponse(r, resp.Data))
}

// GetFollowing handles GET /api/following
func (h *HTTPSocialHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.follow.GetFollowing(r.Context(), &socialpb.GetFollowingRequest{UserId: userID})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildFollowListHTTPResponse(r, resp.Data))
}

// Follow handles GET /api/follow/{user}
func (h *HTTPSocialHandler) Follow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID, err := parsePathUserID(r.URL.Path, "/api/follow/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.follow.Follow(r.Context(), &socialpb.FollowRequest{
		UserId:       userID,
		TargetUserId: targetUserID,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Unfollow handles GET /api/unfollow/{user}
func (h *HTTPSocialHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID, err := parsePathUserID(r.URL.Path, "/api/unfollow/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.follow.Unfollow(r.Context(), &socialpb.UnfollowRequest{
		UserId:       userID,
		TargetUserId: targetUserID,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Remove handles GET /api/remove/{user}
func (h *HTTPSocialHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID, err := parsePathUserID(r.URL.Path, "/api/remove/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.follow.Remove(r.Context(), &socialpb.RemoveRequest{
		UserId:       userID,
		TargetUserId: targetUserID,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetTimings handles GET /api/challenge/timings
func (h *HTTPSocialHandler) GetTimings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.challenge.GetTimings(r.Context(), &socialpb.GetTimingsRequest{UserId: userID})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildTimingsHTTPResponse(resp.Data))
}

// GetQuestion handles GET /api/challenge/question
func (h *HTTPSocialHandler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.challenge.GetQuestion(r.Context(), &socialpb.GetQuestionRequest{UserId: userID})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildQuestionHTTPResponse(resp.Data, false))
}

// SubmitAnswer handles POST /api/challenge/answer
func (h *HTTPSocialHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := h.getUserIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		QuestionID uint64 `json:"question_id"`
		AnswerID   uint64 `json:"answer_id"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	if req.QuestionID == 0 {
		writeError(w, http.StatusUnprocessableEntity, "question_id is required")
		return
	}
	if req.AnswerID == 0 {
		writeError(w, http.StatusUnprocessableEntity, "answer_id is required")
		return
	}

	resp, err := h.challenge.SubmitAnswer(r.Context(), &socialpb.SubmitAnswerRequest{
		UserId:     userID,
		QuestionId: req.QuestionID,
		AnswerId:   req.AnswerID,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildQuestionHTTPResponse(resp.Data, true))
}

// GetAdvertisement handles GET /api/challenge/advertisement
func (h *HTTPSocialHandler) GetAdvertisement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if _, err := h.getUserIDFromRequest(r); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.challenge.GetAdvertisement(r.Context(), &socialpb.GetAdvertisementRequest{})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	ads := make([]map[string]interface{}, 0, len(resp.Advertisements))
	for _, ad := range resp.Advertisements {
		ads = append(ads, map[string]interface{}{
			"code":               ad.Code,
			"title":              ad.Title,
			"description":        ad.Description,
			"investment_value":   ad.InvestmentValue,
			"ends_at":            ad.EndsAt,
			"video_url":          ad.VideoUrl,
			"image_url":          ad.ImageUrl,
			"url":                ad.Url,
			"investment_asset":   ad.InvestmentAsset,
			"prize_per_question": ad.PrizePerQuestion,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": ads})
}

func parsePathUserID(path, prefix string) (uint64, error) {
	pathParts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		return 0, errUserIDRequired
	}

	targetUserID, err := strconv.ParseUint(pathParts[0], 10, 64)
	if err != nil {
		return 0, errInvalidUserID
	}
	return targetUserID, nil
}

var (
	errUserIDRequired = &pathError{"user ID is required"}
	errInvalidUserID  = &pathError{"invalid user ID"}
)

type pathError struct {
	msg string
}

func (e *pathError) Error() string {
	return e.msg
}

func buildFollowListHTTPResponse(r *http.Request, resources []*socialpb.FollowResource) map[string]interface{} {
	page := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	total := int32(len(resources))
	start := (page - 1) * followListPerPage
	if start > total {
		start = total
	}
	end := start + followListPerPage
	if end > total {
		end = total
	}
	pageSlice := resources[start:end]
	hasMore := end < total

	data := make([]map[string]interface{}, 0, len(pageSlice))
	for _, resource := range pageSlice {
		data = append(data, followResourceJSON(resource))
	}

	response := map[string]interface{}{
		"data":  data,
		"links": buildSimplePaginationLinks(r, page, hasMore),
	}

	itemCount := len(data)
	var from interface{}
	var to interface{}
	if itemCount > 0 {
		fromVal := int((page-1)*followListPerPage) + 1
		from = fromVal
		to = fromVal + itemCount - 1
	}

	response["meta"] = map[string]interface{}{
		"current_page": page,
		"from":         from,
		"path":         requestPath(r),
		"per_page":     followListPerPage,
		"to":           to,
	}

	return response
}

func followResourceJSON(resource *socialpb.FollowResource) map[string]interface{} {
	canFollow := false
	canUnfollow := false
	canRemoveFollower := false
	if resource.Can != nil {
		canFollow = resource.Can.Follow
		canUnfollow = resource.Can.Unfollow
		canRemoveFollower = resource.Can.RemoveFollower
	}

	return map[string]interface{}{
		"id":            resource.Id,
		"name":          resource.Name,
		"code":          resource.Code,
		"profile_photo": nullableString(resource.ProfilePhoto),
		"level":         resource.Level,
		"online":        resource.Online,
		"followed":      resource.Followed,
		"can": map[string]bool{
			"follow":          canFollow,
			"unfollow":        canUnfollow,
			"remove_follower": canRemoveFollower,
		},
	}
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func buildTimingsHTTPResponse(data *socialpb.TimingsData) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{
			"data": map[string]interface{}{},
		}
	}

	return map[string]interface{}{
		"data": map[string]interface{}{
			"display_ad_interval":       data.DisplayAdInterval,
			"display_question_interval": data.DisplayQuestionInterval,
			"display_answer_interval":   data.DisplayAnswerInterval,
			"participants":              data.Participants,
			"correct_answers":           data.CorrectAnswers,
			"wrong_answers":             data.WrongAnswers,
			"views":                     data.Views,
		},
	}
}

// buildQuestionHTTPResponse maps a QuestionResource for HTTP JSON.
// Proto json tags use omitempty, which drops zero participants/views; always emit them.
func buildQuestionHTTPResponse(resource *socialpb.QuestionResource, includeAnswerReveal bool) map[string]interface{} {
	if resource == nil {
		return map[string]interface{}{}
	}

	answers := make([]map[string]interface{}, 0, len(resource.Answers))
	for _, answer := range resource.Answers {
		item := map[string]interface{}{
			"id":    answer.Id,
			"title": answer.Title,
			"image": answer.Image,
		}
		if includeAnswerReveal {
			item["is_correct"] = answer.IsCorrect
			item["vote_percentage"] = answer.VotePercentage
		}
		answers = append(answers, item)
	}

	return map[string]interface{}{
		"id":           resource.Id,
		"title":        resource.Title,
		"image":        resource.Image,
		"prize":        resource.Prize,
		"prize_type":   resource.PrizeType,
		"participants": resource.Participants,
		"views":        resource.Views,
		"creator_code": resource.CreatorCode,
		"answers":      answers,
	}
}

func decodeRequestBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return io.EOF
	}
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}
