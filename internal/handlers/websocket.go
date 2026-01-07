package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ChatMessage represents a single chat message in a room.
type ChatMessage struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

// Room represents a chat room with connected clients, message history, and expiration time.
type Room struct {
	Clients  map[*websocket.Conn]bool
	Expiry   time.Time
	History  []ChatMessage
	IsPublic bool
}

// Hub manages all chat rooms and coordinates message broadcasting across clients.
type Hub struct {
	Rooms     map[string]*Room
	Broadcast chan BroadcastMsg
	mu        sync.Mutex
}

// BroadcastMsg wraps a message with its target room ID for broadcasting.
type BroadcastMsg struct {
	RoomID  string
	Message ChatMessage
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewHub creates and initializes a new Hub instance for managing chat rooms.
func NewHub() *Hub {
	return &Hub{
		Rooms:     make(map[string]*Room),
		Broadcast: make(chan BroadcastMsg),
	}
}

// StartCleanup runs a background process that removes expired rooms every minute.
// Closes all client connections in expired rooms before deletion.
func (h *Hub) StartCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for roomID, room := range h.Rooms {
			if now.After(room.Expiry) {
				log.Printf("Room %s expired. Deleting...", roomID)
				for client := range room.Clients {
					client.Close()
				}
				delete(h.Rooms, roomID)
			}
		}
		h.mu.Unlock()
	}
}

// CreateRoom creates a new chat room with the given ID and time-to-live duration.
// Returns false if a room with the same ID already exists.
// isPublic determines if the room appears in the public room list.
func (h *Hub) CreateRoom(id string, ttl time.Duration, isPublic bool) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.Rooms[id]; exists {
		return false
	}

	h.Rooms[id] = &Room{
		Clients:  make(map[*websocket.Conn]bool),
		Expiry:   time.Now().Add(ttl),
		History:  make([]ChatMessage, 0),
		IsPublic: isPublic,
	}
	log.Printf("Created room %s (Public: %t) with TTL %v", id, isPublic, ttl)
	return true
}

// Run processes broadcast messages and distributes them to all clients in the target room.
// Maintains message history (up to 50 messages) and removes disconnected clients.
// This should be run as a goroutine.
func (h *Hub) Run() {
	for bMsg := range h.Broadcast {
		h.mu.Lock()
		if room, ok := h.Rooms[bMsg.RoomID]; ok {
			room.History = append(room.History, bMsg.Message)
			if len(room.History) > 50 {
				room.History = room.History[1:]
			}

			for client := range room.Clients {
				if err := client.WriteJSON(bMsg.Message); err != nil {
					client.Close()
					delete(room.Clients, client)
				}
			}
		}
		h.mu.Unlock()
	}
}

// ServeWs handles WebSocket connection requests and manages the client lifecycle.
// Upgrades HTTP connection to WebSocket, processes tripcode authentication if username contains '#',
// sends message history to new clients, and broadcasts join/leave notifications.
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/ws/")
	roomID := path

	rawName := r.URL.Query().Get("user")
	if rawName == "" {
		rawName = "Anonymous"
	}

	username := rawName
	if strings.Contains(rawName, "#") {
		parts := strings.SplitN(rawName, "#", 2)
		namePart := parts[0]
		passPart := parts[1]

		hash := sha256.Sum256([]byte(passPart))
		encoded := base64.StdEncoding.EncodeToString(hash[:])
		trip := encoded[:8]
		username = namePart + " !" + trip
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	h.mu.Lock()
	room, exists := h.Rooms[roomID]
	if !exists {
		h.mu.Unlock()
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room Expired"))
		conn.Close()
		return
	}
	room.Clients[conn] = true

	for _, msg := range room.History {
		conn.WriteJSON(msg)
	}
	h.mu.Unlock()

	h.Broadcast <- BroadcastMsg{
		RoomID: roomID,
		Message: ChatMessage{
			Username: "System",
			Text:     username + " has joined the room.",
		},
	}

	defer func() {
		h.mu.Lock()
		if room, ok := h.Rooms[roomID]; ok {
			delete(room.Clients, conn)
		}
		h.mu.Unlock()
		conn.Close()

		h.Broadcast <- BroadcastMsg{
			RoomID: roomID,
			Message: ChatMessage{
				Username: "System",
				Text:     username + " has left the room.",
			},
		}
	}()

	for {
		var msg ChatMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		msg.Username = username
		h.Broadcast <- BroadcastMsg{RoomID: roomID, Message: msg}
	}
}

// RoomSummary provides a lightweight view of a room for the public room list.
type RoomSummary struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// GetPublicRooms returns a list of all public rooms with their current client count.
// Only rooms marked as public are included in the result.
func (h *Hub) GetPublicRooms() []RoomSummary {
	h.mu.Lock()
	defer h.mu.Unlock()

	var publicRooms []RoomSummary
	for id, room := range h.Rooms {
		if room.IsPublic {
			publicRooms = append(publicRooms, RoomSummary{
				ID:    id,
				Count: len(room.Clients),
			})
		}
	}
	return publicRooms
}
