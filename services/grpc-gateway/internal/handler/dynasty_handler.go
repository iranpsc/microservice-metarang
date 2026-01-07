package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	"metargb/grpc-gateway/internal/middleware"
	pb "metargb/shared/pb/auth"
	commonpb "metargb/shared/pb/common"
	dynastypb "metargb/shared/pb/dynasty"
)

type DynastyHandler struct {
	dynastyClient     dynastypb.DynastyServiceClient
	joinRequestClient dynastypb.JoinRequestServiceClient
	familyClient      dynastypb.FamilyServiceClient
	prizeClient       dynastypb.DynastyPrizeServiceClient
	authClient        pb.AuthServiceClient
}

func NewDynastyHandler(dynastyConn, authConn *grpc.ClientConn) *DynastyHandler {
	return &DynastyHandler{
		dynastyClient:     dynastypb.NewDynastyServiceClient(dynastyConn),
		joinRequestClient: dynastypb.NewJoinRequestServiceClient(dynastyConn),
		familyClient:      dynastypb.NewFamilyServiceClient(dynastyConn),
		prizeClient:       dynastypb.NewDynastyPrizeServiceClient(dynastyConn),
		authClient:        pb.NewAuthServiceClient(authConn),
	}
}

// GetDynasty handles GET /api/dynasty
func (h *DynastyHandler) GetDynasty(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	grpcReq := &dynastypb.GetUserDynastyRequest{
		UserId: userCtx.UserID,
	}

	resp, err := h.dynastyClient.GetUserDynasty(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// CreateDynasty handles POST /api/dynasty/create/{feature}
func (h *DynastyHandler) CreateDynasty(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract feature ID from path: /api/dynasty/create/{feature}
	featureIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/create/")
	if featureIDStr == "" {
		writeError(w, http.StatusBadRequest, "feature_id is required")
		return
	}

	featureID, err := strconv.ParseUint(featureIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid feature_id")
		return
	}

	grpcReq := &dynastypb.CreateDynastyRequest{
		UserId:    userCtx.UserID,
		FeatureId: featureID,
	}

	resp, err := h.dynastyClient.CreateDynasty(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"data": resp})
}

// UpdateDynastyFeature handles POST /api/dynasty/{dynasty}/update/{feature}
func (h *DynastyHandler) UpdateDynastyFeature(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract dynasty and feature IDs from path: /api/dynasty/{dynasty}/update/{feature}
	path := strings.TrimPrefix(r.URL.Path, "/api/dynasty/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "update" {
		writeError(w, http.StatusBadRequest, "invalid path format: expected /api/dynasty/{dynasty}/update/{feature}")
		return
	}

	dynastyID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dynasty_id")
		return
	}

	featureID, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid feature_id")
		return
	}

	grpcReq := &dynastypb.UpdateDynastyFeatureRequest{
		DynastyId: dynastyID,
		FeatureId: featureID,
		UserId:    userCtx.UserID,
	}

	resp, err := h.dynastyClient.UpdateDynastyFeature(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// GetFamily handles GET /api/dynasty/{dynasty}/family/{family}
func (h *DynastyHandler) GetFamily(w http.ResponseWriter, r *http.Request) {
	token := extractTokenFromHeader(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	validateReq := &pb.ValidateTokenRequest{Token: token}
	_, err := h.authClient.ValidateToken(r.Context(), validateReq)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	// Extract dynasty and family IDs from path
	path := strings.TrimPrefix(r.URL.Path, "/api/dynasty/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "family" {
		writeError(w, http.StatusBadRequest, "invalid path format")
		return
	}

	dynastyID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dynasty_id")
		return
	}

	familyID, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid family_id")
		return
	}

	grpcReq := &dynastypb.GetFamilyRequest{
		DynastyId: dynastyID,
		FamilyId:  familyID,
	}

	resp, err := h.familyClient.GetFamily(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// GetSentRequests handles GET /api/dynasty/requests/sent
func (h *DynastyHandler) GetSentRequests(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	page := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	grpcReq := &dynastypb.GetSentRequestsRequest{
		UserId: userCtx.UserID,
		Pagination: &commonpb.PaginationRequest{
			Page:    page,
			PerPage: 10,
		},
	}

	resp, err := h.joinRequestClient.GetSentRequests(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// GetReceivedRequests handles GET /api/dynasty/requests/recieved
func (h *DynastyHandler) GetReceivedRequests(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	page := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	grpcReq := &dynastypb.GetReceivedRequestsRequest{
		UserId: userCtx.UserID,
		Pagination: &commonpb.PaginationRequest{
			Page:    page,
			PerPage: 10,
		},
	}

	resp, err := h.joinRequestClient.GetReceivedRequests(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// SendJoinRequest handles POST /api/dynasty/add/member
func (h *DynastyHandler) SendJoinRequest(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		User         uint64          `json:"user"`
		Relationship string          `json:"relationship"`
		Message      string          `json:"message,omitempty"`
		Permissions  map[string]bool `json:"permissions,omitempty"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	var permissions *dynastypb.ChildPermissions
	if req.Relationship == "offspring" && req.Permissions != nil {
		permissions = &dynastypb.ChildPermissions{
			BFR:  req.Permissions["BFR"],
			SF:   req.Permissions["SF"],
			W:    req.Permissions["W"],
			JU:   req.Permissions["JU"],
			DM:   req.Permissions["DM"],
			PIUP: req.Permissions["PIUP"],
			PITC: req.Permissions["PITC"],
			PIC:  req.Permissions["PIC"],
			ESOO: req.Permissions["ESOO"],
			COTB: req.Permissions["COTB"],
		}
	}

	grpcReq := &dynastypb.SendJoinRequestRequest{
		FromUserId:   userCtx.UserID,
		ToUserId:     req.User,
		Relationship: req.Relationship,
		Message:      req.Message,
		Permissions:  permissions,
	}

	resp, err := h.joinRequestClient.SendJoinRequest(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"data": resp})
}

// AcceptJoinRequest handles POST /api/dynasty/requests/recieved/{joinRequest}
func (h *DynastyHandler) AcceptJoinRequest(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	requestIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/requests/recieved/")
	if requestIDStr == "" {
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	}

	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request_id")
		return
	}

	grpcReq := &dynastypb.AcceptJoinRequestRequest{
		RequestId: requestID,
		UserId:    userCtx.UserID,
	}

	_, err = h.joinRequestClient.AcceptJoinRequest(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RejectJoinRequest handles DELETE /api/dynasty/requests/recieved/{joinRequest}
func (h *DynastyHandler) RejectJoinRequest(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	requestIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/requests/recieved/")
	if requestIDStr == "" {
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	}

	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request_id")
		return
	}

	grpcReq := &dynastypb.RejectJoinRequestRequest{
		RequestId: requestID,
		UserId:    userCtx.UserID,
	}

	_, err = h.joinRequestClient.RejectJoinRequest(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSentRequest handles GET /api/dynasty/requests/sent/{joinRequest}
func (h *DynastyHandler) GetSentRequest(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	requestIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/requests/sent/")
	if requestIDStr == "" {
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	}

	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request_id")
		return
	}

	grpcReq := &dynastypb.GetJoinRequestRequest{
		RequestId: requestID,
		UserId:    userCtx.UserID,
	}

	resp, err := h.joinRequestClient.GetJoinRequest(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// GetReceivedRequest handles GET /api/dynasty/requests/recieved/{joinRequest}
func (h *DynastyHandler) GetReceivedRequest(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	requestIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/requests/recieved/")
	if requestIDStr == "" {
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	}

	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request_id")
		return
	}

	grpcReq := &dynastypb.GetJoinRequestRequest{
		RequestId: requestID,
		UserId:    userCtx.UserID,
	}

	resp, err := h.joinRequestClient.GetJoinRequest(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// DeleteJoinRequest handles DELETE /api/dynasty/requests/sent/{joinRequest}
func (h *DynastyHandler) DeleteJoinRequest(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	requestIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/requests/sent/")
	if requestIDStr == "" {
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	}

	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request_id")
		return
	}

	grpcReq := &dynastypb.DeleteJoinRequestRequest{
		RequestId: requestID,
		UserId:    userCtx.UserID,
	}

	_, err = h.joinRequestClient.DeleteJoinRequest(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPrizes handles GET /api/dynasty/prizes
func (h *DynastyHandler) GetPrizes(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	page := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	grpcReq := &dynastypb.GetPrizesRequest{
		UserId: userCtx.UserID,
		Pagination: &commonpb.PaginationRequest{
			Page:    page,
			PerPage: 10,
		},
	}

	resp, err := h.prizeClient.GetPrizes(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// ClaimPrize handles POST /api/dynasty/prizes/{recievedPrize}
func (h *DynastyHandler) ClaimPrize(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	prizeIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/prizes/")
	if prizeIDStr == "" {
		writeError(w, http.StatusBadRequest, "prize_id is required")
		return
	}

	prizeID, err := strconv.ParseUint(prizeIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid prize_id")
		return
	}

	grpcReq := &dynastypb.ClaimPrizeRequest{
		PrizeId: prizeID,
		UserId:  userCtx.UserID,
	}

	_, err = h.prizeClient.ClaimPrize(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateChildPermissions handles POST /api/dynasty/children/{user}
func (h *DynastyHandler) UpdateChildPermissions(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	childUserIDStr := extractIDFromPath(r.URL.Path, "/api/dynasty/children/")
	if childUserIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	childUserID, err := strconv.ParseUint(childUserIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	var req struct {
		Permission string `json:"permission"`
		Status     bool   `json:"status"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	// Validate permission code
	validPermissions := map[string]bool{
		"BFR": true, "SF": true, "W": true, "JU": true, "DM": true,
		"PIUP": true, "PITC": true, "PIC": true, "ESOO": true, "COTB": true,
	}
	if !validPermissions[req.Permission] {
		writeError(w, http.StatusUnprocessableEntity, "invalid permission code")
		return
	}

	// Build permissions object with only the specified permission set
	permissions := &dynastypb.ChildPermissions{}
	switch req.Permission {
	case "BFR":
		permissions.BFR = req.Status
	case "SF":
		permissions.SF = req.Status
	case "W":
		permissions.W = req.Status
	case "JU":
		permissions.JU = req.Status
	case "DM":
		permissions.DM = req.Status
	case "PIUP":
		permissions.PIUP = req.Status
	case "PITC":
		permissions.PITC = req.Status
	case "PIC":
		permissions.PIC = req.Status
	case "ESOO":
		permissions.ESOO = req.Status
	case "COTB":
		permissions.COTB = req.Status
	}

	grpcReq := &dynastypb.SetChildPermissionsRequest{
		ChildUserId:  childUserID,
		ParentUserId: userCtx.UserID,
		Permissions:  permissions,
	}

	_, err = h.familyClient.SetChildPermissions(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return empty JSON array per API spec
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}})
}

// SearchUsers handles POST /api/dynasty/search
func (h *DynastyHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	token := extractTokenFromHeader(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	validateReq := &pb.ValidateTokenRequest{Token: token}
	_, err := h.authClient.ValidateToken(r.Context(), validateReq)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	var req struct {
		SearchTerm string `json:"searchTerm"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	if req.SearchTerm == "" {
		writeError(w, http.StatusUnprocessableEntity, "searchTerm is required")
		return
	}

	// Note: User search would need to be implemented via a gRPC call
	// For now, return empty array - this needs to be added to proto or handled differently
	grpcReq := &dynastypb.SearchUsersRequest{
		SearchTerm: req.SearchTerm,
	}

	resp, err := h.joinRequestClient.SearchUsers(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp.Data})
}

// GetDefaultPermissions handles POST /api/dynasty/add/member/get/permissions
func (h *DynastyHandler) GetDefaultPermissions(w http.ResponseWriter, r *http.Request) {
	token := extractTokenFromHeader(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	validateReq := &pb.ValidateTokenRequest{Token: token}
	_, err := h.authClient.ValidateToken(r.Context(), validateReq)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	var req struct {
		Relationship string `json:"relationship"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	if req.Relationship != "offspring" {
		writeError(w, http.StatusUnprocessableEntity, "relationship must be 'offspring'")
		return
	}

	grpcReq := &dynastypb.GetDefaultPermissionsRequest{
		Relationship: req.Relationship,
	}

	resp, err := h.joinRequestClient.GetDefaultPermissions(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp.Permissions})
}
