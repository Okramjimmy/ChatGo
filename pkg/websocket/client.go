package websocket

import (
	"time"

	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 45 * time.Second
	maxMessageSize = 65536 // 64 KiB
)

type Client struct {
	hub    *Hub
	conn   *gws.Conn
	send   chan []byte
	userID uuid.UUID
	log    *zap.Logger
}

func NewClient(hub *Hub, conn *gws.Conn, userID uuid.UUID, log *zap.Logger) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		log:    log,
	}
}

func (c *Client) UserID() uuid.UUID { return c.userID }

// ReadPump pumps messages from the WebSocket to the hub.
// Each connection has one ReadPump goroutine.
func (c *Client) ReadPump(onMessage func([]byte)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if gws.IsUnexpectedCloseError(err, gws.CloseGoingAway, gws.CloseAbnormalClosure) {
				c.log.Warn("ws unexpected close", zap.Error(err))
			}
			break
		}
		if onMessage != nil {
			onMessage(msg)
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket.
// Each connection has one WritePump goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(gws.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(gws.TextMessage, msg); err != nil {
				c.log.Warn("ws write error", zap.Error(err))
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(gws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
