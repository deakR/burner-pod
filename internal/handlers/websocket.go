package handlers

import (
	"log"
	"net/http"
	"strings"

	"burner-pod/internal/hub"
	"burner-pod/internal/security"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWebSocketHandler returns an http.HandlerFunc that wraps ServeWs with the provided hub.
func NewWebSocketHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeWs(h, w, r)
	}
}

// ServeWs handles WebSocket connection requests and manages the client lifecycle.
// Upgrades HTTP connection to WebSocket, processes tripcode authentication if username contains '#',
// registers the client with the hub, and starts per-client write and read pumps.
func ServeWs(h *hub.Hub, w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("roomID")
	if roomID == "" {
		roomID = strings.TrimPrefix(r.URL.Path, "/ws/")
	}

	if err := security.ValidateRoomID(roomID); err != nil {
		http.Error(w, "Invalid room ID: "+err.Error(), http.StatusBadRequest)
		return
	}

	rawName := r.URL.Query().Get("user")
	username := security.ProcessUsername(rawName)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &hub.Client{
		Hub:      h,
		RoomID:   roomID,
		Conn:     conn,
		Send:     make(chan hub.ChatMessage, hub.SendBufferSize),
		Username: username,
	}

	expiry, ok := h.RegisterClient(client)
	if !ok {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room Expired"))
		conn.Close()
		return
	}

	h.Broadcast <- hub.BroadcastMsg{
		RoomID: roomID,
		Message: hub.ChatMessage{
			Username:  "System",
			Text:      username + " has joined the room.",
			ExpiresAt: expiry,
		},
	}

	go client.WritePump()
	go client.ReadPump()
}
