package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	featurespb "metargb/shared/pb/features"
)

type MapsHandler struct {
	mapsClient featurespb.MapsServiceClient
}

func NewMapsHandler(mapsConn *grpc.ClientConn) *MapsHandler {
	return &MapsHandler{
		mapsClient: featurespb.NewMapsServiceClient(mapsConn),
	}
}

// ListMaps handles GET /api/v2/maps
// Returns all maps with basic information (no authentication required)
func (h *MapsHandler) ListMaps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Build gRPC request
	grpcReq := &featurespb.ListMapsRequest{}

	// Call gRPC service
	resp, err := h.mapsClient.ListMaps(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert protobuf maps to JSON format matching Laravel MapResource
	maps := make([]map[string]interface{}, 0, len(resp.Maps))
	for _, m := range resp.Maps {
		mapData := map[string]interface{}{
			"id":                        m.Id,
			"name":                      m.Name,
			"color":                     m.Color,
			"central_point_coordinates": parseJSONString(m.CentralPointCoordinates),
			"sold_features_percentage":  m.SoldFeaturesPercentage,
		}
		maps = append(maps, mapData)
	}

	// Return array of maps (Laravel returns array directly for index)
	writeJSON(w, http.StatusOK, maps)
}

// GetMap handles GET /api/v2/maps/{map}
// Returns a single map with detailed information (no authentication required)
func (h *MapsHandler) GetMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract map ID from path
	// Path format: /api/v2/maps/{map}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		writeError(w, http.StatusBadRequest, "invalid path: map ID required")
		return
	}

	mapIDStr := pathParts[len(pathParts)-1]
	mapID, err := strconv.ParseUint(mapIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid map ID")
		return
	}

	// Build gRPC request
	grpcReq := &featurespb.GetMapRequest{
		MapId: mapID,
	}

	// Call gRPC service
	resp, err := h.mapsClient.GetMap(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Convert protobuf map to JSON format matching Laravel MapResource (detail view)
	mapData := map[string]interface{}{
		"id":                        resp.Map.Id,
		"name":                      resp.Map.Name,
		"color":                     resp.Map.Color,
		"central_point_coordinates": parseJSONString(resp.Map.CentralPointCoordinates),
		"sold_features_percentage":  resp.Map.SoldFeaturesPercentage,
	}

	// Add detail fields (only present in show route)
	if resp.Map.BorderCoordinates != "" {
		mapData["border_coordinates"] = parseJSONString(resp.Map.BorderCoordinates)
	}
	if resp.Map.Area > 0 {
		mapData["area"] = resp.Map.Area
	}
	if resp.Map.Address != "" {
		mapData["address"] = resp.Map.Address
	}
	if resp.Map.PublishedAt != "" {
		mapData["published_at"] = resp.Map.PublishedAt
	}
	if resp.Map.Features != nil {
		featuresMap := map[string]interface{}{}
		if resp.Map.Features.Maskoni != nil {
			featuresMap["maskoni"] = map[string]interface{}{
				"sold": resp.Map.Features.Maskoni.Sold,
			}
		}
		if resp.Map.Features.Tejari != nil {
			featuresMap["tejari"] = map[string]interface{}{
				"sold": resp.Map.Features.Tejari.Sold,
			}
		}
		if resp.Map.Features.Amoozeshi != nil {
			featuresMap["amoozeshi"] = map[string]interface{}{
				"sold": resp.Map.Features.Amoozeshi.Sold,
			}
		}
		mapData["features"] = featuresMap
	}

	// Return single map object (Laravel returns object directly for show)
	writeJSON(w, http.StatusOK, mapData)
}

// GetMapBorder handles GET /api/v2/maps/{map}/border
// Returns just the border coordinates (no authentication required)
func (h *MapsHandler) GetMapBorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract map ID from path
	// Path format: /api/v2/maps/{map}/border
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[len(pathParts)-1] != "border" {
		writeError(w, http.StatusBadRequest, "invalid path: map ID and /border required")
		return
	}

	mapIDStr := pathParts[len(pathParts)-2]
	mapID, err := strconv.ParseUint(mapIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid map ID")
		return
	}

	// Build gRPC request
	grpcReq := &featurespb.GetMapRequest{
		MapId: mapID,
	}

	// Call gRPC service
	resp, err := h.mapsClient.GetMapBorder(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return response matching Laravel format: {"data": {"border_coordinates": ...}}
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"border_coordinates": parseJSONString(resp.Data.BorderCoordinates),
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// parseJSONString parses a JSON string into an interface{} for proper JSON encoding
// This is a helper specific to maps handler for parsing coordinate JSON strings
func parseJSONString(jsonStr string) interface{} {
	if jsonStr == "" {
		return nil
	}
	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// If parsing fails, return as string
		return jsonStr
	}
	return result
}

// Note: writeJSON, writeError, and writeGRPCError are already defined in auth_handler.go
// and are available in the same package
