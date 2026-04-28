package presence

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusAway    Status = "away"
	StatusBusy    Status = "busy"
)

type Presence struct {
	UserID    uuid.UUID `json:"user_id"`
	Status    Status    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	DeviceID  string    `json:"device_id,omitempty"`
}

type TypingEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool      `json:"is_typing"`
}

type UpdateRequest struct {
	Status   Status `json:"status" validate:"required,oneof=online offline away busy"`
	DeviceID string `json:"device_id,omitempty"`
}
