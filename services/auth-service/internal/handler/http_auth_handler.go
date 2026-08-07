// Package handler provides Kong-facing HTTP handlers for auth-service.
package handler

import (
	"net/http"

	"metarang/auth-service/internal/middleware"
	pb "metarang/shared/pb/auth"
	levelspb "metarang/shared/pb/levels"
)

type HTTPAuthHandler struct {
	authClient              pb.AuthServiceClient
	userClient              pb.UserServiceClient
	kycClient               pb.KYCServiceClient
	citizenClient           pb.CitizenServiceClient
	personalInfoClient      pb.PersonalInfoServiceClient
	profileLimitationClient pb.ProfileLimitationServiceClient
	profilePhotoClient      pb.ProfilePhotoServiceClient
	settingsClient          pb.SettingsServiceClient
	userEventsClient        pb.UserEventsServiceClient
	searchClient            pb.SearchServiceClient
	levelClient             levelspb.LevelServiceClient
	locale                  string
}

func NewHTTPAuthHandler(clients LocalClients, levelClient levelspb.LevelServiceClient, locale string) *HTTPAuthHandler {
	return &HTTPAuthHandler{
		authClient:              clients.Auth,
		userClient:              clients.User,
		kycClient:               clients.KYC,
		citizenClient:           clients.Citizen,
		personalInfoClient:      clients.PersonalInfo,
		profileLimitationClient: clients.ProfileLimitation,
		profilePhotoClient:      clients.ProfilePhoto,
		settingsClient:          clients.Settings,
		userEventsClient:        clients.UserEvents,
		searchClient:            clients.Search,
		levelClient:             levelClient,
		locale:                  locale,
	}
}

// writeGRPCErrorLocale writes gRPC errors using the handler's locale.
func (h *HTTPAuthHandler) writeGRPCErrorLocale(w http.ResponseWriter, err error) {
	writeGRPCErrorWithLocale(w, err, h.locale)
}

func (h *HTTPAuthHandler) RequireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userCtx, err := middleware.GetUserFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		user, err := h.userClient.GetUser(r.Context(), &pb.GetUserRequest{UserId: userCtx.UserID})
		if err != nil {
			h.writeGRPCErrorLocale(w, err)
			return
		}
		if user.EmailVerifiedAt == nil {
			writeError(w, http.StatusForbidden, "Your email address is not verified.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
