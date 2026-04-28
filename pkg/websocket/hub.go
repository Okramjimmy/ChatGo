package websocket

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Hub maintains active WebSocket connections and routes events.
type Hub struct {
	clients    map[*Client]bool
	userConns  map[uuid.UUID][]*Client // user_id → clients (multi-device)
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMsg
	mu         sync.RWMutex
	log        *zap.Logger
}

type BroadcastMsg struct {
	// UserIDs limits delivery; nil means all clients
	UserIDs []uuid.UUID
	// ConversationID limits delivery to participants of that conversation
	ConversationID *uuid.UUID
	Payload        []byte
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		userConns:  make(map[uuid.UUID][]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMsg, 1024),
		log:        log,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.userConns[c.userID] = append(h.userConns[c.userID], c)
			h.mu.Unlock()
			h.log.Debug("ws client registered", zap.String("user_id", c.userID.String()))

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				// remove from user connections
				conns := h.userConns[c.userID]
				for i, cc := range conns {
					if cc == c {
						h.userConns[c.userID] = append(conns[:i], conns[i+1:]...)
						break
					}
				}
				if len(h.userConns[c.userID]) == 0 {
					delete(h.userConns, c.userID)
				}
			}
			h.mu.Unlock()
			h.log.Debug("ws client unregistered", zap.String("user_id", c.userID.String()))

		case msg := <-h.broadcast:
			h.mu.RLock()
			if msg.UserIDs != nil {
				for _, uid := range msg.UserIDs {
					for _, c := range h.userConns[uid] {
						select {
						case c.send <- msg.Payload:
						default:
							// slow client — drop message
						}
					}
				}
			} else {
				for c := range h.clients {
					select {
					case c.send <- msg.Payload:
					default:
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUsers marshals an event and delivers it to the specified user IDs.
func (h *Hub) SendToUsers(userIDs []uuid.UUID, eventType string, payload interface{}) {
	data, err := json.Marshal(&Event{Type: eventType, Payload: payload})
	if err != nil {
		h.log.Error("failed to marshal ws event", zap.Error(err))
		return
	}
	h.broadcast <- &BroadcastMsg{UserIDs: userIDs, Payload: data}
}

// SendToAll broadcasts an event to every connected client.
func (h *Hub) SendToAll(eventType string, payload interface{}) {
	data, err := json.Marshal(&Event{Type: eventType, Payload: payload})
	if err != nil {
		h.log.Error("failed to marshal ws event", zap.Error(err))
		return
	}
	h.broadcast <- &BroadcastMsg{Payload: data}
}

// IsOnline reports whether a user has at least one active connection.
func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userConns[userID]) > 0
}

func (h *Hub) Register() chan<- *Client   { return h.register }
func (h *Hub) Unregister() chan<- *Client { return h.unregister }
