package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ContentType string
type StatusType string

const (
	ContentTypeText   ContentType = "text"
	ContentTypeFile   ContentType = "file"
	ContentTypeSystem ContentType = "system"

	StatusSent      StatusType = "sent"
	StatusDelivered StatusType = "delivered"
	StatusRead      StatusType = "read"
)

type Message struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	SenderID       uuid.UUID       `json:"sender_id"`
	Content        string          `json:"content"`
	ContentType    ContentType     `json:"content_type"`
	ParentID       *uuid.UUID      `json:"parent_id,omitempty"`
	IsEdited       bool            `json:"is_edited"`
	IsDeleted      bool            `json:"is_deleted"`
	IsPinned       bool            `json:"is_pinned"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	Reactions []*Reaction `json:"reactions,omitempty"`
	Status    []*Status   `json:"status,omitempty"`
}

type Reaction struct {
	ID        uuid.UUID `json:"id"`
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type Status struct {
	MessageID uuid.UUID  `json:"message_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Status    StatusType `json:"status"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SendRequest struct {
	Content     string          `json:"content" validate:"required,min=1,max=65536"`
	ContentType ContentType     `json:"content_type" validate:"required,oneof=text file system"`
	ParentID    *uuid.UUID      `json:"parent_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type EditRequest struct {
	Content string `json:"content" validate:"required,min=1,max=65536"`
}

type AddReactionRequest struct {
	Emoji string `json:"emoji" validate:"required,min=1,max=8"`
}

type ListFilter struct {
	ConversationID uuid.UUID
	BeforeID       *uuid.UUID
	AfterID        *uuid.UUID
	Limit          int
}

type WSEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	WSEventMessageNew      = "message.new"
	WSEventMessageEdit     = "message.edit"
	WSEventMessageDelete   = "message.delete"
	WSEventMessageReaction = "message.reaction"
	WSEventTypingStart     = "typing.start"
	WSEventTypingStop      = "typing.stop"
	WSEventPresenceUpdate  = "presence.update"
	WSEventNotification    = "notification.new"
)

// MessageRef interface implementation (used by notification.Service).
func (m *Message) GetID() uuid.UUID             { return m.ID }
func (m *Message) GetConversationID() uuid.UUID { return m.ConversationID }
func (m *Message) GetSenderID() uuid.UUID       { return m.SenderID }
func (m *Message) GetContent() string           { return m.Content }

