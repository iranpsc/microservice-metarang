package handler

import (
	"net/http"
	"strings"

	"google.golang.org/grpc"

	levelspb "metargb/shared/pb/levels"
)

type LevelsHandler struct {
	levelClient levelspb.LevelServiceClient
	appURL      string
}

func NewLevelsHandler(conn *grpc.ClientConn, appURL string) *LevelsHandler {
	return &LevelsHandler{
		levelClient: levelspb.NewLevelServiceClient(conn),
		appURL:      strings.TrimSuffix(appURL, "/"),
	}
}

// prefixImageURL prefixes an image/file URL with APP_URL/uploads/ if it's not already a full URL
func (h *LevelsHandler) prefixImageURL(url string) string {
	if url == "" {
		return url
	}
	// If already a full URL (starts with http:// or https://), return as-is
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	// If APP_URL is not configured, return relative path with /uploads/ prefix
	if h.appURL == "" {
		path := strings.TrimPrefix(url, "/")
		if !strings.HasPrefix(path, "uploads/") {
			return "/uploads/" + path
		}
		return "/" + path
	}
	// Prefix with APP_URL/uploads/
	path := strings.TrimPrefix(url, "/")
	if !strings.HasPrefix(path, "uploads/") {
		path = "uploads/" + path
	}
	return h.appURL + "/" + path
}

// GetAllLevels handles GET /api/v2/levels
// Implements Laravel LevelController@index
func (h *LevelsHandler) GetAllLevels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	grpcReq := &levelspb.GetAllLevelsRequest{}
	resp, err := h.levelClient.GetAllLevels(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Format response according to Laravel LevelResource collection
	levels := make([]map[string]interface{}, 0, len(resp.Levels))
	for _, level := range resp.Levels {
		levelMap := map[string]interface{}{
			"id":   level.Id,
			"name": level.Name,
			"slug": level.Slug,
		}

		// Add image URL if present (prefixed with APP_URL/uploads/)
		if level.ImageUrl != "" {
			levelMap["image"] = h.prefixImageURL(level.ImageUrl)
		}

		// Add background_image if present (prefixed with APP_URL/uploads/)
		if level.BackgroundImage != "" {
			levelMap["background_image"] = h.prefixImageURL(level.BackgroundImage)
		}

		levels = append(levels, levelMap)
	}

	writeJSON(w, http.StatusOK, levels)
}

// GetLevel handles GET /api/v2/levels/{slug}
// Implements Laravel LevelController@show
func (h *LevelsHandler) GetLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract slug from path (supports both /api/levels and /api/v2/levels)
	var slug string
	if strings.HasPrefix(r.URL.Path, "/api/v2/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/v2/levels/")
	} else if strings.HasPrefix(r.URL.Path, "/api/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/levels/")
	} else {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	
	if slug == "" {
		writeError(w, http.StatusBadRequest, "level slug is required")
		return
	}

	grpcReq := &levelspb.GetLevelRequest{
		LevelSlug: slug,
	}

	resp, err := h.levelClient.GetLevel(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	level := h.formatLevelResponse(resp.Level)
	writeJSON(w, http.StatusOK, level)
}

// GetLevelGeneralInfo handles GET /api/v2/levels/{slug}/general-info
// Implements Laravel LevelController@getGeneralInfo
func (h *LevelsHandler) GetLevelGeneralInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract slug from path (supports both /api/levels and /api/v2/levels)
	var slug string
	if strings.HasPrefix(r.URL.Path, "/api/v2/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/v2/levels/")
	} else if strings.HasPrefix(r.URL.Path, "/api/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/levels/")
	} else {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	
	if slug == "" {
		writeError(w, http.StatusBadRequest, "level slug is required")
		return
	}

	// Remove "/general-info" suffix if present
	slug = strings.TrimSuffix(slug, "/general-info")

	grpcReq := &levelspb.GetLevelGeneralInfoRequest{
		LevelSlug: slug,
	}

	resp, err := h.levelClient.GetLevelGeneralInfo(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Format response according to Laravel GeneralInfoResource
	var generalInfo map[string]interface{}
	if resp.GeneralInfo != nil {
		generalInfo = map[string]interface{}{
			"id":             resp.GeneralInfo.Id,
			"score":          resp.GeneralInfo.Score,
			"description":    resp.GeneralInfo.Description,
			"rank":           resp.GeneralInfo.Rank,
			"subcategories":  resp.GeneralInfo.Subcategories,
			"persian_font":   resp.GeneralInfo.PersianFont,
			"english_font":   resp.GeneralInfo.EnglishFont,
			"file_volume":    resp.GeneralInfo.FileVolume,
			"used_colors":    resp.GeneralInfo.UsedColors,
			"points":         resp.GeneralInfo.Points,
			"lines":          resp.GeneralInfo.Lines,
			"has_animation":  resp.GeneralInfo.HasAnimation,
			"designer":       resp.GeneralInfo.Designer,
			"model_designer": resp.GeneralInfo.ModelDesigner,
			"creation_date":  resp.GeneralInfo.CreationDate,
			"png_file":       h.prefixImageURL(resp.GeneralInfo.PngFile),
			"fbx_file":       h.prefixImageURL(resp.GeneralInfo.FbxFile),
			"gif_file":       h.prefixImageURL(resp.GeneralInfo.GifFile),
		}
	} else {
		generalInfo = nil
	}

	writeJSON(w, http.StatusOK, generalInfo)
}

// GetLevelGem handles GET /api/v2/levels/{slug}/gem
// Implements Laravel LevelController@gem
func (h *LevelsHandler) GetLevelGem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract slug from path (supports both /api/levels and /api/v2/levels)
	var slug string
	if strings.HasPrefix(r.URL.Path, "/api/v2/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/v2/levels/")
	} else if strings.HasPrefix(r.URL.Path, "/api/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/levels/")
	} else {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	
	if slug == "" {
		writeError(w, http.StatusBadRequest, "level slug is required")
		return
	}

	// Remove "/gem" suffix if present
	slug = strings.TrimSuffix(slug, "/gem")

	grpcReq := &levelspb.GetLevelGemRequest{
		LevelSlug: slug,
	}

	resp, err := h.levelClient.GetLevelGem(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Format response according to Laravel GemResource (all non-hidden columns)
	var gem map[string]interface{}
	if resp.Gem != nil {
		gem = map[string]interface{}{
			"id":            resp.Gem.Id,
			"level_id":      resp.Gem.LevelId,
			"name":          resp.Gem.Name,
			"description":   resp.Gem.Description,
			"thread":        resp.Gem.Thread,
			"points":        resp.Gem.Points,
			"volume":        resp.Gem.Volume,
			"color":         resp.Gem.Color,
			"has_animation": resp.Gem.HasAnimation,
			"lines":         resp.Gem.Lines,
			"png_file":      h.prefixImageURL(resp.Gem.PngFile),
			"fbx_file":      h.prefixImageURL(resp.Gem.FbxFile),
			"encryption":    resp.Gem.Encryption,
			"designer":      resp.Gem.Designer,
		}
		// Note: ImageUrl field not available in LevelGem proto
	} else {
		gem = nil
	}

	writeJSON(w, http.StatusOK, gem)
}

// GetLevelGift handles GET /api/v2/levels/{slug}/gift
// Implements Laravel LevelController@gift
func (h *LevelsHandler) GetLevelGift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract slug from path (supports both /api/levels and /api/v2/levels)
	var slug string
	if strings.HasPrefix(r.URL.Path, "/api/v2/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/v2/levels/")
	} else if strings.HasPrefix(r.URL.Path, "/api/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/levels/")
	} else {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	
	if slug == "" {
		writeError(w, http.StatusBadRequest, "level slug is required")
		return
	}

	// Remove "/gift" suffix if present
	slug = strings.TrimSuffix(slug, "/gift")

	grpcReq := &levelspb.GetLevelGiftRequest{
		LevelSlug: slug,
	}

	resp, err := h.levelClient.GetLevelGift(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Format response according to Laravel GiftResource (all non-hidden columns)
	var gift map[string]interface{}
	if resp.Gift != nil {
		gift = map[string]interface{}{
			"id":                        resp.Gift.Id,
			"level_id":                  resp.Gift.LevelId,
			"name":                      resp.Gift.Name,
			"description":               resp.Gift.Description,
			"monthly_capacity_count":    resp.Gift.MonthlyCapacityCount,
			"store_capacity":            resp.Gift.StoreCapacity,
			"sell_capacity":             resp.Gift.SellCapacity,
			"features":                  resp.Gift.Features,
			"sell":                      resp.Gift.Sell,
			"vod_document_registration": resp.Gift.VodDocumentRegistration,
			"seller_link":               resp.Gift.SellerLink,
			"designer":                  resp.Gift.Designer,
			"three_d_model_volume":      resp.Gift.ThreeDModelVolume,
			"three_d_model_points":      resp.Gift.ThreeDModelPoints,
			"three_d_model_lines":       resp.Gift.ThreeDModelLines,
			"has_animation":             resp.Gift.HasAnimation,
			"png_file":                  h.prefixImageURL(resp.Gift.PngFile),
			"fbx_file":                  h.prefixImageURL(resp.Gift.FbxFile),
			"gif_file":                  h.prefixImageURL(resp.Gift.GifFile),
			"rent":                      resp.Gift.Rent,
			"vod_count":                 resp.Gift.VodCount,
			"start_vod_id":              resp.Gift.StartVodId,
			"end_vod_id":                resp.Gift.EndVodId,
		}
	} else {
		gift = nil
	}

	writeJSON(w, http.StatusOK, gift)
}

// GetLevelLicenses handles GET /api/v2/levels/{slug}/licenses
// Implements Laravel LevelController@licenses
func (h *LevelsHandler) GetLevelLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract slug from path (supports both /api/levels and /api/v2/levels)
	var slug string
	if strings.HasPrefix(r.URL.Path, "/api/v2/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/v2/levels/")
	} else if strings.HasPrefix(r.URL.Path, "/api/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/levels/")
	} else {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	
	if slug == "" {
		writeError(w, http.StatusBadRequest, "level slug is required")
		return
	}

	// Remove "/licenses" suffix if present
	slug = strings.TrimSuffix(slug, "/licenses")

	grpcReq := &levelspb.GetLevelLicensesRequest{
		LevelSlug: slug,
	}

	resp, err := h.levelClient.GetLevelLicenses(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Format response according to Laravel LicensesResource (all non-hidden columns)
	var licenses map[string]interface{}
	if resp.Licenses != nil {
		licenses = map[string]interface{}{
			"id":                                     resp.Licenses.Id,
			"level_id":                               resp.Licenses.LevelId,
			"create_union":                           resp.Licenses.CreateUnion,
			"add_memeber_to_union":                   resp.Licenses.AddMemeberToUnion,
			"observation_license":                    resp.Licenses.ObservationLicense,
			"gate_license":                           resp.Licenses.GateLicense,
			"lawyer_license":                         resp.Licenses.LawyerLicense,
			"city_counsile_entry":                    resp.Licenses.CityCounsileEntry,
			"establish_special_residential_property": resp.Licenses.EstablishSpecialResidentialProperty,
			"establish_property_on_surface":          resp.Licenses.EstablishPropertyOnSurface,
			"judge_entry":                            resp.Licenses.JudgeEntry,
			"upload_image":                           resp.Licenses.UploadImage,
			"delete_image":                           resp.Licenses.DeleteImage,
			"inter_level_general_points":             resp.Licenses.InterLevelGeneralPoints,
			"inter_level_special_points":             resp.Licenses.InterLevelSpecialPoints,
			"rent_out_satisfaction":                  resp.Licenses.RentOutSatisfaction,
			"access_to_answer_questions_unit":        resp.Licenses.AccessToAnswerQuestionsUnit,
			"create_challenge_questions":             resp.Licenses.CreateChallengeQuestions,
			"upload_music":                           resp.Licenses.UploadMusic,
		}
	} else {
		licenses = nil
	}

	writeJSON(w, http.StatusOK, licenses)
}

// GetLevelPrize handles GET /api/v2/levels/{slug}/prize
// Implements Laravel LevelController@prizes
func (h *LevelsHandler) GetLevelPrize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract slug from path (supports both /api/levels and /api/v2/levels)
	var slug string
	if strings.HasPrefix(r.URL.Path, "/api/v2/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/v2/levels/")
	} else if strings.HasPrefix(r.URL.Path, "/api/levels/") {
		slug = extractSlugFromPath(r.URL.Path, "/api/levels/")
	} else {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	
	if slug == "" {
		writeError(w, http.StatusBadRequest, "level slug is required")
		return
	}

	// Remove "/prize" suffix if present
	slug = strings.TrimSuffix(slug, "/prize")

	grpcReq := &levelspb.GetLevelPrizesRequest{
		LevelSlug: slug,
	}

	resp, err := h.levelClient.GetLevelPrizes(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Format response according to Laravel PrizeResource
	var prize map[string]interface{}
	if resp.Prize != nil {
		prize = map[string]interface{}{
			"id":           resp.Prize.Id,
			"level_id":     resp.Prize.LevelId,
			"psc":          resp.Prize.Psc,
			"yellow":       resp.Prize.Yellow,
			"blue":         resp.Prize.Blue,
			"red":          resp.Prize.Red,
			"effect":       resp.Prize.Effect,
			"satisfaction": resp.Prize.Satisfaction,
		}
		// created_at is formatted in Jalali format in Laravel, but we'll return it as-is for now
		if resp.Prize.CreatedAt != "" {
			prize["created_at"] = resp.Prize.CreatedAt
		}
	} else {
		prize = nil
	}

	writeJSON(w, http.StatusOK, prize)
}

// formatLevelResponse formats a Level proto message to match Laravel LevelResource format
func (h *LevelsHandler) formatLevelResponse(level *levelspb.Level) map[string]interface{} {
	if level == nil {
		return nil
	}

	levelMap := map[string]interface{}{
		"id":   level.Id,
		"name": level.Name,
		"slug": level.Slug,
	}

	// Add image URL if present (prefixed with APP_URL/uploads/)
	if level.ImageUrl != "" {
		levelMap["image"] = h.prefixImageURL(level.ImageUrl)
	}

	// Add background_image if present (prefixed with APP_URL/uploads/)
	if level.BackgroundImage != "" {
		levelMap["background_image"] = h.prefixImageURL(level.BackgroundImage)
	}

	// Add general_info if loaded (only in show endpoint)
	if level.GeneralInfo != nil {
		levelMap["general_info"] = map[string]interface{}{
			"score":       level.GeneralInfo.Score,
			"rank":        level.GeneralInfo.Rank,
			"png_file":    h.prefixImageURL(level.GeneralInfo.PngFile),
			"fbx_file":    h.prefixImageURL(level.GeneralInfo.FbxFile),
			"gif_file":    h.prefixImageURL(level.GeneralInfo.GifFile),
			"description": level.GeneralInfo.Description,
		}
	}

	return levelMap
}

// HandleLevelsRoutes is the main router for levels endpoints
// Handles both /api/levels and /api/v2/levels prefixes
// Routes to appropriate handler based on path structure
func (h *LevelsHandler) HandleLevelsRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Determine the base prefix used (/api/levels or /api/v2/levels)
	var basePrefix string
	if strings.HasPrefix(path, "/api/v2/levels") {
		basePrefix = "/api/v2/levels"
	} else if strings.HasPrefix(path, "/api/levels") {
		basePrefix = "/api/levels"
	} else {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	
	// Handle exact match: /api/levels or /api/v2/levels
	if path == basePrefix {
		h.GetAllLevels(w, r)
		return
	}
	
	// Extract the path after the prefix
	suffix := strings.TrimPrefix(path, basePrefix)
	suffix = strings.TrimPrefix(suffix, "/")
	
	if suffix == "" {
		// This is /api/levels/ or /api/v2/levels/ - treat as list
		h.GetAllLevels(w, r)
		return
	}
	
	// Split the suffix into parts
	parts := strings.Split(suffix, "/")
	slug := parts[0]
	
	// Handle nested routes: /api/levels/{slug}/general-info, etc.
	if len(parts) == 2 {
		resource := parts[1]
		switch resource {
		case "general-info":
			// Temporarily update the path for the handler
			r.URL.Path = basePrefix + "/" + slug + "/general-info"
			h.GetLevelGeneralInfo(w, r)
			return
		case "gem":
			r.URL.Path = basePrefix + "/" + slug + "/gem"
			h.GetLevelGem(w, r)
			return
		case "gift":
			r.URL.Path = basePrefix + "/" + slug + "/gift"
			h.GetLevelGift(w, r)
			return
		case "licenses":
			r.URL.Path = basePrefix + "/" + slug + "/licenses"
			h.GetLevelLicenses(w, r)
			return
		case "prize":
			r.URL.Path = basePrefix + "/" + slug + "/prize"
			h.GetLevelPrize(w, r)
			return
		}
	}
	
	// Handle single level: /api/levels/{slug} or /api/v2/levels/{slug}
	if len(parts) == 1 {
		r.URL.Path = basePrefix + "/" + slug
		h.GetLevel(w, r)
		return
	}
	
	// No match found
	writeError(w, http.StatusNotFound, "not found")
}

// extractSlugFromPath extracts slug from URL path
// Supports both /api/levels and /api/v2/levels prefixes
// e.g., "/api/levels/my-level" -> "my-level"
// e.g., "/api/v2/levels/my-level/general-info" -> "my-level/general-info"
func extractSlugFromPath(path, prefix string) string {
	// Support both /api/levels and /api/v2/levels
	if strings.HasPrefix(path, "/api/v2/levels/") {
		if prefix == "/api/v2/levels/" || prefix == "/api/levels/" {
			return strings.TrimPrefix(path, "/api/v2/levels/")
		}
	} else if strings.HasPrefix(path, "/api/levels/") {
		if prefix == "/api/levels/" || prefix == "/api/v2/levels/" {
			return strings.TrimPrefix(path, "/api/levels/")
		}
	}
	
	// Fallback to original logic
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimPrefix(path, prefix)
}
