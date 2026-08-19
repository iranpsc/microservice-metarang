// Package hub manages WebSocket connections and event broadcasting.
package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"

	"metarang/websocket-gateway/internal/auth"
)

var errUnauthorized = errors.New("unauthorized")

// Hub manages Socket.IO connections and Redis-driven broadcasts.
type Hub struct {
	server *socketio.Server
	users  map[uint64]map[string]struct{}
	mu     sync.RWMutex
	auth   *auth.Validator
}

// New creates a Socket.IO hub with authentication on connect.
func New(validator *auth.Validator) *Hub {
	h := &Hub{
		users: make(map[uint64]map[string]struct{}),
		auth:  validator,
	}

	opts := &engineio.Options{
		Transports: []transport.Transport{
			polling.Default,
			websocket.Default,
		},
	}

	server := socketio.NewServer(opts)

	server.OnConnect("/", func(s socketio.Conn) error {
		token := extractToken(s)
		if token == "" {
			return errUnauthorized
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		userID, err := h.auth.ValidateToken(ctx, token)
		if err != nil {
			log.Printf("token validation failed: %v", err)
			return errUnauthorized
		}

		s.SetContext(userID)
		h.track(userID, s.ID())
		room := userRoom(userID)
		s.Join(room)

		s.Emit("connected", map[string]any{
			"message":   "Connected to metarang WebSocket Gateway",
			"userId":    userID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return nil
	})

	server.OnEvent("/", "ping", func(s socketio.Conn) {
		s.Emit("pong", map[string]any{"timestamp": time.Now().UnixMilli()})
	})

	server.OnDisconnect("/", func(s socketio.Conn, _ string) {
		userID, ok := s.Context().(uint64)
		if !ok {
			return
		}
		h.untrack(userID, s.ID())
	})

	server.OnError("/", func(_ socketio.Conn, err error) {
		log.Printf("socket error: %v", err)
	})

	h.server = server
	return h
}

// ServeHTTP handles Socket.IO traffic.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

func extractToken(s socketio.Conn) string {
	reqURL := s.URL()
	if token := reqURL.Query().Get("token"); token != "" {
		return token
	}
	if authHeader := s.RemoteHeader().Get("Authorization"); authHeader != "" {
		return strings.TrimPrefix(strings.TrimSpace(authHeader), "Bearer ")
	}
	return ""
}

func (h *Hub) track(userID uint64, socketID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.users[userID] == nil {
		h.users[userID] = make(map[string]struct{})
	}
	h.users[userID][socketID] = struct{}{}
}

func (h *Hub) untrack(userID uint64, socketID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sockets, ok := h.users[userID]
	if !ok {
		return
	}
	delete(sockets, socketID)
	if len(sockets) == 0 {
		delete(h.users, userID)
	}
}

// Stats returns connection metrics for health endpoints.
func (h *Hub) Stats() (connections int, users int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sockets := range h.users {
		connections += len(sockets)
	}
	return connections, len(h.users)
}

// BroadcastUserStatus sends a user-status event to a specific user room.
func (h *Hub) BroadcastUserStatus(data map[string]any) {
	userID, ok := numericID(data["user_id"])
	if !ok {
		return
	}
	h.server.BroadcastToRoom("/", userRoom(userID), "user-status-changed", data)
}

// BroadcastFeatureStatus sends feature ownership updates to involved users.
func (h *Hub) BroadcastFeatureStatus(data map[string]any) {
	if oldOwner, ok := numericID(data["old_owner_id"]); ok {
		h.server.BroadcastToRoom("/", userRoom(oldOwner), "feature-status-changed", merge(data, map[string]any{"userType": "old_owner"}))
	}
	if newOwner, ok := numericID(data["new_owner_id"]); ok {
		h.server.BroadcastToRoom("/", userRoom(newOwner), "feature-status-changed", merge(data, map[string]any{"userType": "new_owner"}))
	}
}

// BroadcastNotification sends a notification to a user room.
func (h *Hub) BroadcastNotification(data map[string]any) {
	userID, ok := numericID(data["user_id"])
	if !ok {
		return
	}
	payload := map[string]any{
		"id":         data["id"],
		"type":       data["type"],
		"title":      data["title"],
		"message":    data["message"],
		"data":       data["data"],
		"created_at": data["created_at"],
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	h.server.BroadcastToRoom("/", userRoom(userID), "notification-received", payload)
}

func userRoom(userID uint64) string {
	return "user:" + formatUint(userID)
}

func merge(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func numericID(value any) (uint64, bool) {
	switch v := value.(type) {
	case float64:
		return uint64(v), true
	case json.Number:
		n, err := v.Int64()
		return uint64(n), err == nil
	case int:
		return uint64(v), true
	case int64:
		return uint64(v), true
	case uint64:
		return v, true
	default:
		return 0, false
	}
}

func formatUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
