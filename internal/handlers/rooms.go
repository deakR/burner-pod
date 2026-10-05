package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"burner-pod/internal/hub"
	"burner-pod/internal/security"
)

// NewRoomCreateHandler creates an HTTP handler for POST /rooms (and /create alias).
// It validates room ID format (^[a-zA-Z0-9_-]{1,64}$), clamps TTL (1..604800, default 60),
// handles duplicate conflicts (409), and issues a 303 See Other redirect to /chat.
func NewRoomCreateHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}

		roomID := r.FormValue("code")

		// Empty code triggers server-side random ID generation
		if roomID == "" {
			var err error
			roomID, err = security.GenerateRoomID()
			if err != nil {
				http.Error(w, "Failed to generate room ID", http.StatusInternalServerError)
				return
			}
		} else {
			// Strict server-side room ID validation
			if err := security.ValidateRoomID(roomID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		isPublic := r.FormValue("public") == "true"

		// Parse and clamp TTL
		ttl := security.DefaultTTL
		if ttlStr := r.FormValue("ttl"); ttlStr != "" {
			if sec, err := strconv.Atoi(ttlStr); err == nil {
				ttl = security.ValidateTTL(sec)
			}
		}

		username := r.FormValue("user")
		if username == "" {
			username = "Anonymous"
		}

		duration := time.Duration(ttl) * time.Second

		success := h.CreateRoom(roomID, duration, isPublic)
		if !success {
			http.Error(w, "Error: Room name '"+roomID+"' is already taken.", http.StatusConflict)
			return
		}

		q := url.Values{}
		q.Set("room", roomID)
		q.Set("user", username)
		targetURL := "/chat?" + q.Encode()

		// 303 See Other redirect to /chat per PRG pattern (PROJECT.md § Feature 10)
		http.Redirect(w, r, targetURL, http.StatusSeeOther)
	}
}

// NewRoomsListHandler creates an HTTP handler for GET /api/rooms returning public active rooms as JSON.
func NewRoomsListHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rooms := h.GetPublicRooms()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rooms)
	}
}
