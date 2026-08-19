// Package handler provides Kong-facing HTTP handlers for auth-service.
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"metarang/auth-service/internal/middleware"
	pb "metarang/shared/pb/auth"
)

// ListBankAccounts handles GET /api/bank-accounts
func (h *HTTPAuthHandler) ListBankAccounts(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	grpcReq := &pb.ListBankAccountsRequest{
		UserId: userCtx.UserID,
	}

	resp, err := h.kycClient.ListBankAccounts(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	data := make([]map[string]interface{}, 0, len(resp.Data))
	for _, account := range resp.Data {
		data = append(data, formatBankAccountResource(account))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

// CreateBankAccount handles POST /api/bank-accounts
func (h *HTTPAuthHandler) CreateBankAccount(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		BankName string `json:"bank_name"`
		ShabaNum string `json:"shaba_num"`
		CardNum  string `json:"card_num"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	grpcReq := &pb.CreateBankAccountRequest{
		UserId:   userCtx.UserID,
		BankName: req.BankName,
		ShabaNum: req.ShabaNum,
		CardNum:  req.CardNum,
	}

	resp, err := h.kycClient.CreateBankAccount(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, formatBankAccountResource(resp))
}

// GetBankAccount handles GET /api/bank-accounts/{bankAccount}
func (h *HTTPAuthHandler) GetBankAccount(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	bankAccountIDStr := extractIDFromPath(r.URL.Path, "/api/bank-accounts/")
	if bankAccountIDStr == "" {
		writeError(w, http.StatusBadRequest, "bank_account_id is required")
		return
	}

	bankAccountID, err := strconv.ParseUint(bankAccountIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bank_account_id")
		return
	}

	grpcReq := &pb.GetBankAccountRequest{
		UserId:        userCtx.UserID,
		BankAccountId: bankAccountID,
	}

	resp, err := h.kycClient.GetBankAccount(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, formatBankAccountResource(resp))
}

// UpdateBankAccount handles PUT/PATCH /api/bank-accounts/{bankAccount}
func (h *HTTPAuthHandler) UpdateBankAccount(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	bankAccountIDStr := extractIDFromPath(r.URL.Path, "/api/bank-accounts/")
	if bankAccountIDStr == "" {
		writeError(w, http.StatusBadRequest, "bank_account_id is required")
		return
	}

	bankAccountID, err := strconv.ParseUint(bankAccountIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bank_account_id")
		return
	}

	var req struct {
		BankName string `json:"bank_name"`
		ShabaNum string `json:"shaba_num"`
		CardNum  string `json:"card_num"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	grpcReq := &pb.UpdateBankAccountRequest{
		UserId:        userCtx.UserID,
		BankAccountId: bankAccountID,
		BankName:      req.BankName,
		ShabaNum:      req.ShabaNum,
		CardNum:       req.CardNum,
	}

	resp, err := h.kycClient.UpdateBankAccount(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, formatBankAccountResource(resp))
}

// DeleteBankAccount handles DELETE /api/bank-accounts/{bankAccount}
func (h *HTTPAuthHandler) DeleteBankAccount(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	bankAccountIDStr := extractIDFromPath(r.URL.Path, "/api/bank-accounts/")
	if bankAccountIDStr == "" {
		writeError(w, http.StatusBadRequest, "bank_account_id is required")
		return
	}

	bankAccountID, err := strconv.ParseUint(bankAccountIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid bank_account_id")
		return
	}

	grpcReq := &pb.DeleteBankAccountRequest{
		UserId:        userCtx.UserID,
		BankAccountId: bankAccountID,
	}

	_, err = h.kycClient.DeleteBankAccount(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}

func formatBankAccountResource(resp *pb.BankAccountResponse) map[string]interface{} {
	response := map[string]interface{}{
		"id":        resp.Id,
		"bank_name": resp.BankName,
		"shaba_num": resp.ShabaNum,
		"card_num":  resp.CardNum,
		"status":    resp.Status,
	}
	if parsed := parseBankAccountErrors(resp.Errors); parsed != nil {
		response["errors"] = parsed
	}
	return response
}

func parseBankAccountErrors(raw string) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		return arr
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		return obj
	}
	return raw
}
