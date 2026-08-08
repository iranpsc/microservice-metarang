package handler

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"

	"metarang/training-service/internal/middleware"
)

func trainingContextWithUser(r *http.Request) context.Context {
	ctx := r.Context()
	md := metadata.MD{}

	if ip := clientIPFromRequest(r); ip != "" {
		md.Set("x-forwarded-for", ip)
	}

	userCtx, err := middleware.GetUserFromRequest(r)
	if err == nil && userCtx.UserID != 0 {
		md.Set(headerUserID, strconv.FormatUint(userCtx.UserID, 10))
	}

	if len(md) == 0 {
		return ctx
	}
	return metadata.NewIncomingContext(ctx, md)
}

func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func getIPAddress(r *http.Request) string {
	ip := clientIPFromRequest(r)
	if ip == "" {
		return r.RemoteAddr
	}
	return ip
}

func parseLikedFromRequest(r *http.Request) (bool, error) {
	if likedStr := r.URL.Query().Get("liked"); likedStr != "" {
		return likedStr == "1" || likedStr == "true", nil
	}

	var body struct {
		Liked bool `json:"liked"`
	}
	if err := decodeRequestBody(r, &body); err != nil {
		return false, err
	}
	return body.Liked, nil
}

func requireUser(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return 0, false
	}
	return userCtx.UserID, true
}

func isEOF(err error) bool {
	return err == io.EOF
}
