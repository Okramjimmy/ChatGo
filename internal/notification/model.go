package notification

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeMessage  Type = "message"
	TypeMention  Type = "mention"
	TypeSystem   Type = "system"
	TypeSecurity Type = "security"
)

type Notification struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Type          Type       `json:"type"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	ReferenceID   string     `json:"reference_id,omitempty"`
	ReferenceType string     `json:"reference_type,omitempty"`
	IsRead        bool       `json:"is_read"`
	CreatedAt     time.Time  `json:"created_at"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
}

type CreateRequest struct {
	UserID        uuid.UUID `json:"user_id"`
	Type          Type      `json:"type"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	ReferenceID   string    `json:"reference_id,omitempty"`
	ReferenceType string    `json:"reference_type,omitempty"`
}

type ListFilter struct {
	UserID uuid.UUID
	IsRead *bool
	Type   *Type
	Limit  int
	Offset int
}
