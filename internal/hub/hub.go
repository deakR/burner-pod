package hub

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// NewHub creates and initializes a new Hub instance for managing chat rooms.
func NewHub() *Hub {
	return &Hub{
		Rooms:     make(map[string]*Room),
		Broadcast: make(chan BroadcastMsg, 256),
	}
}

// StartCleanup runs a background process that removes expired rooms every minute.
// Closes all client connections in expired rooms before deletion.
func (h *Hub) StartCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.Mu.Lock()
		now := time.Now()
		for roomID, room := range h.Rooms {
			if now.After(room.Expiry) {
				log.Printf("Room %s expired. Deleting...", roomID)
				for client := range room.Clients {
					client.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room Expired"))
					client.Conn.Close()
					client.Close()
				}
				delete(h.Rooms, roomID)
			}
		}
		h.Mu.Unlock()
	}
}

// CreateRoom creates a new chat room with the given ID and time-to-live duration.
// Returns false if a room with the same ID already exists.
// isPublic determines if the room appears in the public room list.
func (h *Hub) CreateRoom(id string, ttl time.Duration, isPublic bool) bool {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if _, exists := h.Rooms[id]; exists {
		return false
	}

	h.Rooms[id] = &Room{
		Clients:  make(map[*Client]bool),
		Expiry:   time.Now().Add(ttl),
		History:  make([]ChatMessage, 0),
		IsPublic: isPublic,
	}
	log.Printf("Created room %s (Public: %t) with TTL %v", id, isPublic, ttl)
	return true
}

// RegisterClient adds a client to its target room and sends message history.
// Returns false if the room does not exist.
func (h *Hub) RegisterClient(c *Client) bool {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	room, exists := h.Rooms[c.RoomID]
	if !exists {
		return false
	}
	room.Clients[c] = true

	for _, msg := range room.History {
		select {
		case c.Send <- msg:
		default:
		}
	}
	return true
}

// UnregisterClient removes a client from its target room and closes its send channel.
func (h *Hub) UnregisterClient(c *Client) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if room, ok := h.Rooms[c.RoomID]; ok {
		if _, exists := room.Clients[c]; exists {
			delete(room.Clients, c)
			c.Close()
		}
	}
}

// Run processes broadcast messages and distributes them to all clients in the target room.
// Maintains message history (up to 50 messages) and evicts slow clients whose send buffer is full.
// This should be run as a goroutine.
func (h *Hub) Run() {
	for bMsg := range h.Broadcast {
		h.Mu.Lock()
		room, ok := h.Rooms[bMsg.RoomID]
		if !ok {
			h.Mu.Unlock()
			continue
		}

		if bMsg.Message.Username != "System" {
			room.History = append(room.History, bMsg.Message)
			if len(room.History) > 50 {
				room.History = room.History[1:]
			}
		}

		// Snapshot clients to release lock before channel sends
		clients := make([]*Client, 0, len(room.Clients))
		for client := range room.Clients {
			clients = append(clients, client)
		}
		h.Mu.Unlock()

		for _, client := range clients {
			select {
			case client.Send <- bMsg.Message:
			default:
				// Evict slow or unresponsive client without holding the lock
				h.UnregisterClient(client)
				client.Conn.Close()
			}
		}
	}
}

// GetPublicRooms returns a list of all public rooms with their current client count.
// Only rooms marked as public are included in the result.
func (h *Hub) GetPublicRooms() []RoomSummary {
	h.Mu.Lock()
	defer h.Mu.Unlock()

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
