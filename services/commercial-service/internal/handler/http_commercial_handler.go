package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"metarang/commercial-service/internal/middleware"
	commercialpb "metarang/shared/pb/commercial"
	"metarang/shared/pkg/helpers"
	"metarang/shared/pkg/sentry"
)

type transactionAPI interface {
	ListTransactions(context.Context, *commercialpb.ListTransactionsRequest) (*commercialpb.ListTransactionsResponse, error)
	GetLatestTransaction(context.Context, *commercialpb.GetLatestTransactionRequest) (*commercialpb.LatestTransactionResponse, error)
}

// HTTPCommercialHandler serves Kong-facing REST routes for commercial-service.
type HTTPCommercialHandler struct {
	transactions  transactionAPI
	walletHistory walletHistoryAPI
	citizen       citizenUserInfoAPI
}

// NewHTTPCommercialHandler wraps local gRPC handlers for HTTP use.
func NewHTTPCommercialHandler(
	transactions transactionAPI,
	walletHistory walletHistoryAPI,
	citizen citizenUserInfoAPI,
) *HTTPCommercialHandler {
	return &HTTPCommercialHandler{
		transactions:  transactions,
		walletHistory: walletHistory,
		citizen:       citizen,
	}
}

// RegisterHTTPRoutes registers commercial REST routes and /health.
func (h *HTTPCommercialHandler) RegisterHTTPRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// More specific path first.
	mux.Handle("/api/user/transactions/latest", authMiddleware(http.HandlerFunc(h.GetLatestTransaction)))
	mux.Handle("/api/user/transactions", authMiddleware(http.HandlerFunc(h.ListTransactions)))

	// Public citizen wallet history (privacy applied via auth GetCitizenUserInfo).
	mux.HandleFunc("GET /api/citizen/{code}/wallet/history/summary", h.GetCitizenWalletHistorySummary)
	mux.HandleFunc("GET /api/citizen/{code}/wallet/history/chart", h.GetCitizenWalletHistoryChart)
}

// StartHTTPServer starts the public HTTP server (behind Kong).
func StartHTTPServer(
	httpHandler *HTTPCommercialHandler,
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

// ListTransactions handles GET /api/user/transactions
func (h *HTTPCommercialHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	query := r.URL.Query()

	page := int32(1)
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	perPage := int32(15)
	if perPageStr := query.Get("per_page"); perPageStr != "" {
		if p, err := strconv.ParseInt(perPageStr, 10, 32); err == nil && p > 0 {
			perPage = int32(p)
		}
	}

	grpcReq := &commercialpb.ListTransactionsRequest{
		UserId:        userCtx.UserID,
		Page:          page,
		PerPage:       perPage,
		Search:        query.Get("search"),
		StartDateTime: query.Get("start_date_time"),
		EndDateTime:   query.Get("end_date_time"),
		Action:        query.Get("action"),
		Asset:         query.Get("asset"),
		Type:          query.Get("type"),
		Status:        parseStatusQueryParams(query),
	}

	resp, err := h.transactions.ListTransactions(r.Context(), grpcReq)
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	transactions := make([]map[string]interface{}, 0, len(resp.Transactions))
	for _, tx := range resp.Transactions {
		transactions = append(transactions, map[string]interface{}{
			"id":     tx.Id,
			"type":   tx.Type,
			"asset":  tx.Asset,
			"amount": tx.Amount,
			"action": tx.Action,
			"status": tx.Status,
			"date":   tx.Date,
			"time":   tx.Time,
		})
	}

	currentPage := page
	if resp.CurrentPage > 0 {
		currentPage = resp.CurrentPage
	}

	response := map[string]interface{}{
		"data":  transactions,
		"links": buildSimplePaginationLinks(r, currentPage, resp.HasMorePages),
	}

	itemCount := len(transactions)
	var from interface{}
	var to interface{}
	if itemCount > 0 {
		fromVal := int((currentPage-1)*perPage) + 1
		from = fromVal
		to = fromVal + itemCount - 1
	}

	response["meta"] = map[string]interface{}{
		"current_page": currentPage,
		"from":         from,
		"path":         requestPath(r),
		"per_page":     perPage,
		"to":           to,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetLatestTransaction handles GET /api/user/transactions/latest
func (h *HTTPCommercialHandler) GetLatestTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	resp, err := h.transactions.GetLatestTransaction(r.Context(), &commercialpb.GetLatestTransactionRequest{
		UserId: userCtx.UserID,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	data := map[string]interface{}{}
	if resp.LatestTransaction != nil {
		data["id"] = resp.LatestTransaction.Id
		data["amount"] = resp.LatestTransaction.Amount
		data["status"] = resp.LatestTransaction.Status
		data["asset"] = resp.LatestTransaction.Asset

		date := ""
		timeValue := ""
		if resp.LatestTransaction.CreatedAt != nil {
			createdAt := resp.LatestTransaction.CreatedAt.AsTime()
			date = helpers.FormatJalaliDate(createdAt)
			timeValue = helpers.FormatJalaliTime(createdAt)
		}
		data["date"] = date
		data["time"] = timeValue
	}

	if resp.LatestPayment != nil {
		paymentInfo := map[string]interface{}{
			"ref_id": resp.LatestPayment.RefId,
		}
		if resp.LatestPayment.CreatedAt != nil {
			createdAt := resp.LatestPayment.CreatedAt.AsTime()
			paymentInfo["date"] = helpers.FormatJalaliDate(createdAt)
			paymentInfo["hour"] = helpers.FormatJalaliTime(createdAt)
		}
		data["payment_info"] = paymentInfo
	}

	if resp.LatestOrder != nil {
		data["product"] = resp.LatestOrder.Asset
		data["count"] = resp.LatestOrder.Amount
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func parseStatusQueryParams(query map[string][]string) []int32 {
	statusValues := parseIndexedQueryArray(query, "status")
	if len(statusValues) == 0 {
		if values, ok := query["status"]; ok {
			statusValues = values
		} else if values, ok := query["status[]"]; ok {
			statusValues = values
		}
	}

	statuses := make([]int32, 0, len(statusValues))
	for _, value := range statusValues {
		if value == "" {
			continue
		}
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if status, err := strconv.ParseInt(part, 10, 32); err == nil {
				statuses = append(statuses, int32(status))
			}
		}
	}

	return statuses
}
