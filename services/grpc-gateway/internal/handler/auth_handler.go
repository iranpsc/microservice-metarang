package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metargb/shared/pb/auth"
)

type AuthHandler struct {
	authClient pb.AuthServiceClient
	userClient pb.UserServiceClient
	kycClient  pb.KYCServiceClient
}

func NewAuthHandler(conn *grpc.ClientConn) *AuthHandler {
	return &AuthHandler{
		authClient: pb.NewAuthServiceClient(conn),
		userClient: pb.NewUserServiceClient(conn),
		kycClient:  pb.NewKYCServiceClient(conn),
	}
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BackURL  string `json:"back_url"`
		Referral string `json:"referral"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.RegisterRequest{
		BackUrl:  req.BackURL,
		Referral: req.Referral,
	}

	resp, err := h.authClient.Register(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": resp.Url})
}

// Redirect handles GET /api/auth/redirect
func (h *AuthHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	redirectTo := r.URL.Query().Get("redirect_to")
	backURL := r.URL.Query().Get("back_url")

	grpcReq := &pb.RedirectRequest{
		RedirectTo: redirectTo,
		BackUrl:    backURL,
	}

	resp, err := h.authClient.Redirect(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": resp.Url})
}

// Callback handles GET /api/auth/callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	grpcReq := &pb.CallbackRequest{
		State: state,
		Code:  code,
	}

	resp, err := h.authClient.Callback(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":        resp.Token,
		"expires_at":   resp.ExpiresAt,
		"redirect_url": resp.RedirectUrl,
	})
}

// GetMe handles POST /api/auth/me
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.GetMeRequest{
		Token: req.Token,
	}

	resp, err := h.authClient.GetMe(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	response := map[string]interface{}{
		"id":                             resp.Id,
		"name":                           resp.Name,
		"token":                          resp.Token,
		"access_token":                   resp.AccessToken,
		"automatic_logout":               resp.AutomaticLogout,
		"code":                           resp.Code,
		"image":                          resp.Image,
		"notifications":                  resp.Notifications,
		"socre_percentage_to_next_level": resp.SocrePercentageToNextLevel,
		"unasnwered_questions_count":     resp.UnasnweredQuestionsCount,
		"hourly_profit_time_percentage":  resp.HourlyProfitTimePercentage,
		"verified_kyc":                   resp.VerifiedKyc,
		"birthdate":                      resp.Birthdate,
	}

	if resp.Level != nil {
		response["level"] = map[string]interface{}{
			"id":          resp.Level.Id,
			"title":       resp.Level.Title,
			"description": resp.Level.Description,
			"score":       resp.Level.Score,
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.LogoutRequest{
		Token: req.Token,
	}

	_, err := h.authClient.Logout(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// ValidateToken handles POST /api/auth/validate
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.ValidateTokenRequest{
		Token: req.Token,
	}

	resp, err := h.authClient.ValidateToken(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   resp.Valid,
		"user_id": resp.UserId,
		"email":   resp.Email,
	})
}

// RequestAccountSecurity handles POST /api/auth/account-security/request
func (h *AuthHandler) RequestAccountSecurity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID      uint64 `json:"user_id"`
		TimeMinutes int32  `json:"time_minutes"`
		Phone       string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.RequestAccountSecurityRequest{
		UserId:      req.UserID,
		TimeMinutes: req.TimeMinutes,
		Phone:       req.Phone,
	}

	_, err := h.authClient.RequestAccountSecurity(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "OTP sent successfully"})
}

// VerifyAccountSecurity handles POST /api/auth/account-security/verify
func (h *AuthHandler) VerifyAccountSecurity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    uint64 `json:"user_id"`
		Code      string `json:"code"`
		IP        string `json:"ip"`
		UserAgent string `json:"user_agent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Extract IP and UserAgent from request if not provided
	if req.IP == "" {
		req.IP = getClientIP(r)
	}
	if req.UserAgent == "" {
		req.UserAgent = r.UserAgent()
	}

	grpcReq := &pb.VerifyAccountSecurityRequest{
		UserId:    req.UserID,
		Code:      req.Code,
		Ip:        req.IP,
		UserAgent: req.UserAgent,
	}

	_, err := h.authClient.VerifyAccountSecurity(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "account security verified successfully"})
}

// GetUser handles GET /api/user
func (h *AuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	grpcReq := &pb.GetUserRequest{
		UserId: userID,
	}

	resp, err := h.userClient.GetUser(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateProfile handles PUT/PATCH /api/user/profile
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID uint64 `json:"user_id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Phone  string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.UpdateProfileRequest{
		UserId: req.UserID,
		Name:   req.Name,
		Email:  req.Email,
		Phone:  req.Phone,
	}

	resp, err := h.userClient.UpdateProfile(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// SubmitKYC handles POST /api/kyc/submit
func (h *AuthHandler) SubmitKYC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       uint64 `json:"user_id"`
		Fname        string `json:"fname"`
		Lname        string `json:"lname"`
		NationalCode string `json:"national_code"`
		Birthdate    string `json:"birthdate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.SubmitKYCRequest{
		UserId:       req.UserID,
		Fname:        req.Fname,
		Lname:        req.Lname,
		NationalCode: req.NationalCode,
		Birthdate:    req.Birthdate,
	}

	resp, err := h.kycClient.SubmitKYC(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetKYCStatus handles GET /api/kyc/status
func (h *AuthHandler) GetKYCStatus(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	grpcReq := &pb.GetKYCStatusRequest{
		UserId: userID,
	}

	resp, err := h.kycClient.GetKYCStatus(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// VerifyBankAccount handles POST /api/kyc/bank-account
func (h *AuthHandler) VerifyBankAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   uint64 `json:"user_id"`
		BankName string `json:"bank_name"`
		ShabaNum string `json:"shaba_num"`
		CardNum  string `json:"card_num"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &pb.VerifyBankAccountRequest{
		UserId:   req.UserID,
		BankName: req.BankName,
		ShabaNum: req.ShabaNum,
		CardNum:  req.CardNum,
	}

	resp, err := h.kycClient.VerifyBankAccount(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch st.Code() {
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, st.Message())
	case codes.FailedPrecondition:
		writeError(w, http.StatusPreconditionFailed, st.Message())
	default:
		writeError(w, http.StatusInternalServerError, st.Message())
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
