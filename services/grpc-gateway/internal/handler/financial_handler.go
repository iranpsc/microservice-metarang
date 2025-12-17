package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metargb/shared/pb/auth"
	financialpb "metargb/shared/pb/financial"
)

type FinancialHandler struct {
	orderClient financialpb.OrderServiceClient
	storeClient financialpb.StoreServiceClient
	authClient  pb.AuthServiceClient
}

func NewFinancialHandler(financialConn, authConn *grpc.ClientConn) *FinancialHandler {
	return &FinancialHandler{
		orderClient: financialpb.NewOrderServiceClient(financialConn),
		storeClient: financialpb.NewStoreServiceClient(financialConn),
		authClient:  pb.NewAuthServiceClient(authConn),
	}
}

// CreateOrder handles POST /api/order
func (h *FinancialHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract token to get user_id
	token := extractTokenFromHeader(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Validate token to get user_id
	validateReq := &pb.ValidateTokenRequest{Token: token}
	validateResp, err := h.authClient.ValidateToken(r.Context(), validateReq)
	if err != nil || !validateResp.Valid {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	userID := validateResp.UserId

	var req struct {
		Amount int32  `json:"amount"`
		Asset  string `json:"asset"`
	}

	if err := decodeJSONBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	// Validate amount
	if req.Amount < 1 {
		writeValidationErrorWithErrors(w, "The amount field must be at least 1", map[string][]string{
			"amount": {"The amount field must be at least 1"},
		})
		return
	}

	// Validate asset
	validAssets := map[string]bool{"psc": true, "irr": true, "red": true, "blue": true, "yellow": true}
	if !validAssets[req.Asset] {
		writeValidationErrorWithErrors(w, "The selected asset is invalid", map[string][]string{
			"asset": {"The selected asset is invalid"},
		})
		return
	}

	grpcReq := &financialpb.CreateOrderRequest{
		UserId: userID,
		Amount: req.Amount,
		Asset:  req.Asset,
	}

	resp, err := h.orderClient.CreateOrder(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"link": resp.Link,
	})
}

// HandleCallback handles POST /api/parsian/callback
func (h *FinancialHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse form data (Parsian sends form-encoded data)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse form data")
		return
	}

	// Extract OrderId and status
	orderIDStr := r.FormValue("OrderId")
	if orderIDStr == "" {
		writeError(w, http.StatusBadRequest, "OrderId is required")
		return
	}

	orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid OrderId")
		return
	}

	statusStr := r.FormValue("status")
	status, err := strconv.ParseInt(statusStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	tokenStr := r.FormValue("Token")
	token, err := strconv.ParseInt(tokenStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid Token")
		return
	}

	// Collect all additional parameters
	additionalParams := make(map[string]string)
	for k, v := range r.Form {
		if k != "OrderId" && k != "status" && k != "Token" {
			if len(v) > 0 {
				additionalParams[k] = v[0]
			}
		}
	}

	grpcReq := &financialpb.HandleCallbackRequest{
		OrderId:          orderID,
		Status:           int32(status),
		Token:            token,
		AdditionalParams: additionalParams,
	}

	resp, err := h.orderClient.HandleCallback(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Redirect to frontend URL
	http.Redirect(w, r, resp.RedirectUrl, http.StatusFound)
}

// GetStorePackages handles POST /api/store
func (h *FinancialHandler) GetStorePackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Codes []string `json:"codes"`
	}

	if err := decodeJSONBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	// Validation: at least 2 codes required
	if len(req.Codes) < 2 {
		writeValidationErrorWithErrors(w, "The codes field must contain at least 2 items", map[string][]string{
			"codes": {"The codes field must contain at least 2 items"},
		})
		return
	}

	// Validate each code
	for i, code := range req.Codes {
		if len(code) < 2 {
			writeValidationErrorWithErrors(w, "The codes field must contain valid codes", map[string][]string{
				"codes": {fmt.Sprintf("The codes.%d field must be at least 2 characters", i)},
			})
			return
		}
	}

	grpcReq := &financialpb.GetStorePackagesRequest{
		Codes: req.Codes,
	}

	resp, err := h.storeClient.GetStorePackages(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert to JSON format matching Laravel PackageResource
	packages := make([]map[string]interface{}, 0, len(resp.Packages))
	for _, pkg := range resp.Packages {
		pkgData := map[string]interface{}{
			"id":        pkg.Id,
			"code":      pkg.Code,
			"asset":     pkg.Asset,
			"amount":    pkg.Amount,
			"unitPrice": pkg.UnitPrice,
		}
		if pkg.Image != nil && *pkg.Image != "" {
			pkgData["image"] = *pkg.Image
		} else {
			pkgData["image"] = nil
		}
		packages = append(packages, pkgData)
	}

	writeJSON(w, http.StatusOK, packages)
}

// Helper functions

// decodeJSONBody safely decodes JSON from request body, handling empty bodies
func decodeJSONBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return io.EOF
	}

	// Check if body is empty
	if r.ContentLength == 0 {
		return io.EOF
	}

	// Try to peek at the body to see if it's already consumed
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	
	// Restore body for potential subsequent reads
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) == 0 {
		return io.EOF
	}

	return json.Unmarshal(bodyBytes, v)
}

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

// Helper function (different signature - takes errors map)
func writeValidationErrorWithErrors(w http.ResponseWriter, message string, errors map[string][]string) {
	response := map[string]interface{}{
		"message": message,
		"errors":  errors,
	}
	writeJSON(w, http.StatusUnprocessableEntity, response)
}
