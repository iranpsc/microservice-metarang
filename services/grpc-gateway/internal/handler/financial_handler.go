package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/grpc"

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

// Helper function (different signature - takes errors map)
func writeValidationErrorWithErrors(w http.ResponseWriter, message string, errors map[string][]string) {
	response := map[string]interface{}{
		"message": message,
		"errors":  errors,
	}
	writeJSON(w, http.StatusUnprocessableEntity, response)
}
