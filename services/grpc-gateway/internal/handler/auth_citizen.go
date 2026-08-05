// Package handler provides HTTP handlers for the gRPC gateway service.
package handler

import (
	"net/http"
	"strconv"
	"strings"

	pb "metarang/shared/pb/auth"
)

// ============================================================================
// Citizen Service Handlers (Public endpoints - no auth required)
// ============================================================================

// HandleCitizenRoutes handles all citizen-related routes
func (h *AuthHandler) HandleCitizenRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Path format: /api/citizen/{code} or /api/citizen/{code}/referrals or /api/citizen/{code}/referrals/chart

	// Extract code from path
	parts := strings.Split(strings.TrimPrefix(path, "/api/citizen/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "citizen code is required")
		return
	}

	code := parts[0]

	if len(parts) > 1 {
		switch parts[1] {
		case "referrals":
			if len(parts) > 2 && parts[2] == "chart" {
				// /api/citizen/{code}/referrals/chart
				h.GetCitizenReferralChart(w, r, code)
			} else {
				// /api/citizen/{code}/referrals
				h.GetCitizenReferrals(w, r, code)
			}
		default:
			writeError(w, http.StatusNotFound, "invalid citizen endpoint")
		}
	} else {
		// /api/citizen/{code}
		h.GetCitizenProfile(w, r, code)
	}
}

// GetCitizenProfile handles GET /api/citizen/{code}
func (h *AuthHandler) GetCitizenProfile(w http.ResponseWriter, r *http.Request, code string) {
	grpcReq := &pb.GetCitizenProfileRequest{
		Code: code,
	}

	resp, err := h.citizenClient.GetCitizenProfile(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, citizenProfileJSONRoundTrip(buildCitizenProfileHTTPResponse(resp)))
}

// GetCitizenReferrals handles GET /api/citizen/{code}/referrals
func (h *AuthHandler) GetCitizenReferrals(w http.ResponseWriter, r *http.Request, code string) {
	search := r.URL.Query().Get("search")
	pageStr := r.URL.Query().Get("page")
	page := int32(1)
	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
			page = int32(p)
		}
	}

	grpcReq := &pb.GetCitizenReferralsRequest{
		Code:   code,
		Search: search,
		Page:   page,
	}

	resp, err := h.citizenClient.GetCitizenReferrals(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildCitizenReferralsHTTPResponse(r, resp))
}

// GetCitizenReferralChart handles GET /api/citizen/{code}/referrals/chart
func (h *AuthHandler) GetCitizenReferralChart(w http.ResponseWriter, r *http.Request, code string) {
	rangeType := r.URL.Query().Get("range")
	if rangeType == "" {
		rangeType = "daily"
	}

	grpcReq := &pb.GetCitizenReferralChartRequest{
		Code:  code,
		Range: rangeType,
	}

	resp, err := h.citizenClient.GetCitizenReferralChart(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildCitizenReferralChartHTTPResponse(resp))
}

func buildCitizenReferralsHTTPResponse(r *http.Request, resp *pb.CitizenReferralsResponse) map[string]interface{} {
	const perPage int32 = 10

	currentPage := int32(1)
	hasMore := false
	if resp.Meta != nil {
		currentPage = resp.Meta.CurrentPage
		if currentPage <= 0 {
			currentPage = 1
		}
		hasMore = resp.Meta.NextPageUrl != ""
	}

	referrals := make([]map[string]interface{}, 0, len(resp.Data))
	for _, ref := range resp.Data {
		item := map[string]interface{}{
			"id":   ref.Id,
			"code": ref.Code,
			"name": ref.Name,
		}
		if ref.Image != "" {
			item["image"] = ref.Image
		}
		orders := make([]map[string]interface{}, 0, len(ref.ReferrerOrders))
		for _, order := range ref.ReferrerOrders {
			orders = append(orders, map[string]interface{}{
				"id":         order.Id,
				"amount":     order.Amount,
				"created_at": order.CreatedAt,
			})
		}
		item["referrerOrders"] = orders
		referrals = append(referrals, item)
	}

	response := map[string]interface{}{
		"data": referrals,
	}

	response["links"] = buildSimplePaginationLinks(r, currentPage, hasMore)

	itemCount := len(referrals)
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

	return response
}

func buildCitizenReferralChartHTTPResponse(resp *pb.CitizenReferralChartResponse) map[string]interface{} {
	chartPayload := map[string]interface{}{
		"total_referrals_count":        "0",
		"total_referral_orders_amount": "0",
		"chart_data":                   []interface{}{},
	}

	if resp.Data != nil {
		chartPayload["total_referrals_count"] = resp.Data.TotalReferralsCount
		chartPayload["total_referral_orders_amount"] = resp.Data.TotalReferralOrdersAmount

		chartData := make([]map[string]interface{}, 0, len(resp.Data.ChartData))
		for _, point := range resp.Data.ChartData {
			chartData = append(chartData, map[string]interface{}{
				"label":        point.Label,
				"count":        point.Count,
				"total_amount": point.TotalAmount,
			})
		}
		chartPayload["chart_data"] = chartData
	}

	return map[string]interface{}{
		"data": chartPayload,
	}
}
