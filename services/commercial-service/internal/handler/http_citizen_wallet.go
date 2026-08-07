package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "metarang/shared/pb/auth"
	commercialpb "metarang/shared/pb/commercial"
)

var citizenWalletAssetSet = map[string]struct{}{
	"psc": {}, "irr": {}, "red": {}, "blue": {}, "yellow": {}, "satisfaction": {}, "effect": {},
}

type walletHistoryAPI interface {
	GetWalletHistorySummary(context.Context, *commercialpb.GetWalletHistorySummaryRequest) (*commercialpb.GetWalletHistorySummaryResponse, error)
	GetWalletHistoryChart(context.Context, *commercialpb.GetWalletHistoryChartRequest) (*commercialpb.GetWalletHistoryChartResponse, error)
}

type citizenUserInfoAPI interface {
	GetCitizenUserInfo(context.Context, *authpb.GetCitizenUserInfoRequest, ...grpc.CallOption) (*authpb.GetCitizenUserInfoResponse, error)
}

// GetCitizenWalletHistorySummary handles GET /api/citizen/{code}/wallet/history/summary
func (h *HTTPCommercialHandler) GetCitizenWalletHistorySummary(w http.ResponseWriter, r *http.Request) {
	h.handleCitizenWalletHistory(w, r, "summary")
}

// GetCitizenWalletHistoryChart handles GET /api/citizen/{code}/wallet/history/chart
func (h *HTTPCommercialHandler) GetCitizenWalletHistoryChart(w http.ResponseWriter, r *http.Request) {
	h.handleCitizenWalletHistory(w, r, "chart")
}

func (h *HTTPCommercialHandler) handleCitizenWalletHistory(w http.ResponseWriter, r *http.Request, kind string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := r.PathValue("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "citizen code is required")
		return
	}

	if h.citizen == nil || h.walletHistory == nil {
		writeError(w, http.StatusServiceUnavailable, "commercial service unavailable")
		return
	}

	userID, privacy, ok := h.resolveCitizen(w, r, code)
	if !ok {
		return
	}
	period, ok := resolveWalletHistoryPeriod(w, r)
	if !ok {
		return
	}
	assets, ok := parseWalletAssetsQuery(w, r.URL.Query())
	if !ok {
		return
	}

	switch kind {
	case "summary":
		h.writeWalletHistorySummary(w, r, userID, period, assets, privacy)
	case "chart":
		h.writeWalletHistoryChart(w, r, userID, period, assets, privacy)
	default:
		writeError(w, http.StatusNotFound, "invalid citizen wallet history endpoint")
	}
}

func (h *HTTPCommercialHandler) writeWalletHistorySummary(
	w http.ResponseWriter,
	r *http.Request,
	userID uint64,
	period string,
	assets []string,
	privacy map[string]int32,
) {
	resp, err := h.walletHistory.GetWalletHistorySummary(r.Context(), &commercialpb.GetWalletHistorySummaryRequest{
		UserId:  userID,
		Period:  period,
		Assets:  assets,
		Privacy: privacy,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	data := make([]map[string]interface{}, 0, len(resp.Data))
	for _, card := range resp.Data {
		if card.PrivacyRestricted {
			data = append(data, map[string]interface{}{
				"asset":              card.Asset,
				"privacy_restricted": true,
			})
			continue
		}
		data = append(data, map[string]interface{}{
			"asset":              card.Asset,
			"current_balance":    card.CurrentBalance,
			"period_income":      card.PeriodIncome,
			"period_spending":    card.PeriodSpending,
			"growth_percent":     card.GrowthPercent,
			"direction":          card.Direction,
			"privacy_restricted": false,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (h *HTTPCommercialHandler) writeWalletHistoryChart(
	w http.ResponseWriter,
	r *http.Request,
	userID uint64,
	period string,
	assets []string,
	privacy map[string]int32,
) {
	resp, err := h.walletHistory.GetWalletHistoryChart(r.Context(), &commercialpb.GetWalletHistoryChartRequest{
		UserId:  userID,
		Period:  period,
		Assets:  assets,
		Privacy: privacy,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	data := make(map[string]interface{}, len(resp.Data))
	for asset, series := range resp.Data {
		data[asset] = map[string]interface{}{
			"income":   chartPointsJSON(series.Income),
			"spending": chartPointsJSON(series.Spending),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (h *HTTPCommercialHandler) resolveCitizen(w http.ResponseWriter, r *http.Request, code string) (uint64, map[string]int32, bool) {
	info, err := h.citizen.GetCitizenUserInfo(r.Context(), &authpb.GetCitizenUserInfoRequest{Code: code})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			writeError(w, http.StatusNotFound, "citizen not found")
			return 0, nil, false
		}
		writeHandlerError(w, err)
		return 0, nil, false
	}
	return info.UserId, info.Privacy, true
}

func resolveWalletHistoryPeriod(w http.ResponseWriter, r *http.Request) (string, bool) {
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	switch period {
	case "daily", "weekly", "monthly", "yearly":
		return period, true
	case "":
		writeError(w, http.StatusUnprocessableEntity, "The period field is required.")
		return "", false
	default:
		writeError(w, http.StatusUnprocessableEntity, "The selected period is invalid.")
		return "", false
	}
}

func parseWalletAssetsQuery(w http.ResponseWriter, query url.Values) ([]string, bool) {
	raw := []string{}
	if indexed := parseIndexedQueryArray(query, "assets"); len(indexed) > 0 {
		raw = indexed
	} else if vals, ok := query["assets[]"]; ok {
		raw = vals
	} else if vals, ok := query["assets"]; ok {
		raw = vals
	} else if asset := strings.TrimSpace(query.Get("asset")); asset != "" {
		raw = []string{asset}
	}

	if len(raw) == 0 {
		return nil, true
	}

	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, asset := range raw {
		asset = strings.ToLower(strings.TrimSpace(asset))
		if asset == "" {
			continue
		}
		if _, ok := citizenWalletAssetSet[asset]; !ok {
			writeError(w, http.StatusUnprocessableEntity, "The selected assets is invalid.")
			return nil, false
		}
		if _, dup := seen[asset]; dup {
			continue
		}
		seen[asset] = struct{}{}
		out = append(out, asset)
	}
	return out, true
}

func chartPointsJSON(points []*commercialpb.WalletChartPoint) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(points))
	for _, p := range points {
		out = append(out, map[string]interface{}{
			"label":  p.Label,
			"amount": p.Amount,
		})
	}
	return out
}
