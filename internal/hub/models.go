package hub

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ChatMessage represents a single chat message in a room.
type ChatMessage struct {
	Username string `json:"username"`
	Text     string `json:"text"`
}

// Client represents a connected websocket client in a room.
type Client struct {
	Hub      *Hub
	RoomID   string
	Conn     *websocket.Conn
	Send     chan ChatMessage
	Username string
	closeOnce sync.Once
}

// Close closes the client send channel safely exactly once.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.Send)
	})
}

// Room represents a chat room with connected clients, message history, and expiration time.
type Room struct {
	Clients  map[*Client]bool
	Expiry   time.Time
	History  []ChatMessage
	IsPublic bool
}

// Hub manages all chat rooms and coordinates message broadcasting across clients.
type Hub struct {
	Rooms     map[string]*Room
	Broadcast chan BroadcastMsg
	Mu        sync.Mutex
}

// BroadcastMsg wraps a message with its target room ID for broadcasting.
type BroadcastMsg struct {
	RoomID  string
	Message ChatMessage
}

// RoomSummary provides a lightweight view of a room for the public room list.
type RoomSummary struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}
