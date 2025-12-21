package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	"metargb/grpc-gateway/internal/middleware"
	pb "metargb/shared/pb/auth"
	featurespb "metargb/shared/pb/features"
)

type ProfitHandler struct {
	profitClient featurespb.FeatureProfitServiceClient
	authClient   pb.AuthServiceClient
}

func NewProfitHandler(featuresConn, authConn *grpc.ClientConn) *ProfitHandler {
	return &ProfitHandler{
		profitClient: featurespb.NewFeatureProfitServiceClient(featuresConn),
		authClient:   pb.NewAuthServiceClient(authConn),
	}
}

// GetHourlyProfits handles GET /api/hourly-profits
// Implements Laravel's FeatureHourlyProfitController@index
func (h *ProfitHandler) GetHourlyProfits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract authenticated user ID from token
	userID, err := h.getAuthenticatedUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse pagination parameters (default: page=1, page_size=10)
	page := int32(1)
	if p := r.URL.Query().Get("page"); p != "" {
		if pInt, err := strconv.ParseInt(p, 10, 32); err == nil && pInt > 0 {
			page = int32(pInt)
		}
	}

	pageSize := int32(10)
	if ps := r.URL.Query().Get("per_page"); ps != "" {
		if psInt, err := strconv.ParseInt(ps, 10, 32); err == nil && psInt > 0 {
			pageSize = int32(psInt)
		}
	}

	// Build gRPC request
	grpcReq := &featurespb.GetHourlyProfitsRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	}

	// Call gRPC service
	resp, err := h.profitClient.GetHourlyProfits(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert to Laravel HourlyProfitResource format
	profits := make([]map[string]interface{}, 0, len(resp.Profits))
	for _, profit := range resp.Profits {
		profitMap := map[string]interface{}{
			"id":         profit.Id,
			"feature_id": profit.FeatureId,
			"amount":     profit.Amount,
			"dead_line":  profit.DeadLine,
			"is_active":  profit.IsActive,
		}
		profits = append(profits, profitMap)
	}

	// Build response matching Laravel format
	responseData := map[string]interface{}{
		"data": profits,
		"additional": map[string]interface{}{
			"total_maskoni_profit":  resp.TotalMaskoniProfit,
			"total_tejari_profit":   resp.TotalTejariProfit,
			"total_amozeshi_profit": resp.TotalAmozeshiProfit,
		},
	}

	writeJSON(w, http.StatusOK, responseData)
}

// GetProfitsByApplication handles POST /api/hourly-profits
// Implements Laravel's FeatureHourlyProfitController@getProfitsByApplication
func (h *ProfitHandler) GetProfitsByApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract authenticated user ID from token
	userID, err := h.getAuthenticatedUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse request body
	var req struct {
		Karbari string `json:"karbari"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeValidationError(w, "request body is required")
		} else {
			writeValidationError(w, "invalid request body")
		}
		return
	}

	// Validate karbari (required|in:m,t,a)
	if req.Karbari == "" {
		writeValidationError(w, "karbari field is required")
		return
	}
	if req.Karbari != "m" && req.Karbari != "t" && req.Karbari != "a" {
		writeValidationError(w, "karbari must be one of: m, t, a")
		return
	}

	// Build gRPC request
	grpcReq := &featurespb.GetProfitsByApplicationRequest{
		UserId:  userID,
		Karbari: req.Karbari,
	}

	// Call gRPC service
	_, err = h.profitClient.GetProfitsByApplication(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return empty JSON object {} (matching Laravel implementation)
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

// GetSingleProfit handles POST /api/hourly-profits/{featureHourlyProfit}
// Implements Laravel's FeatureHourlyProfitController@getSingleProfit
func (h *ProfitHandler) GetSingleProfit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract authenticated user ID from token
	userID, err := h.getAuthenticatedUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract profit ID from path: /api/hourly-profits/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/hourly-profits/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusBadRequest, "profit ID is required in path")
		return
	}

	profitID, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profit ID")
		return
	}

	// Build gRPC request
	grpcReq := &featurespb.GetSingleProfitRequest{
		ProfitId: profitID,
		UserId:   userID,
	}

	// Call gRPC service
	resp, err := h.profitClient.GetSingleProfit(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert to Laravel HourlyProfitResource format
	profitMap := map[string]interface{}{
		"id":         resp.Profit.Id,
		"feature_id": resp.Profit.FeatureId,
		"amount":     resp.Profit.Amount,
		"dead_line":  resp.Profit.DeadLine,
		"is_active":  resp.Profit.IsActive,
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": profitMap})
}

// getAuthenticatedUserID extracts user ID from context (set by auth middleware)
func (h *ProfitHandler) getAuthenticatedUserID(r *http.Request) (uint64, error) {
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		return 0, err
	}
	return userCtx.UserID, nil
}
