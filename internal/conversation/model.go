package conversation

import (
	"time"

	"github.com/google/uuid"
)

type Type string
type ChannelType string

const (
	TypeDirect  Type = "direct"
	TypeGroup   Type = "group"
	TypeChannel Type = "channel"

	ChannelPublic       ChannelType = "public"
	ChannelPrivate      ChannelType = "private"
	ChannelModerated    ChannelType = "moderated"
	ChannelAnnouncement ChannelType = "announcement"
)

type Conversation struct {
	ID           uuid.UUID    `json:"id"`
	Type         Type         `json:"type"`
	Name         string       `json:"name,omitempty"`
	Description  string       `json:"description,omitempty"`
	ChannelType  *ChannelType `json:"channel_type,omitempty"`
	IsInviteOnly bool         `json:"is_invite_only"`
	CreatorID    uuid.UUID    `json:"creator_id"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	DeletedAt    *time.Time   `json:"deleted_at,omitempty"`

	Participants []*Participant `json:"participants,omitempty"`
}

type Participant struct {
	ID                   uuid.UUID `json:"id"`
	ConversationID       uuid.UUID `json:"conversation_id"`
	UserID               uuid.UUID `json:"user_id"`
	Role                 string    `json:"role"`
	JoinedAt             time.Time `json:"joined_at"`
	LastReadAt           time.Time `json:"last_read_at"`
	IsMuted              bool      `json:"is_muted"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
}

type CreateRequest struct {
	Type         Type         `json:"type" validate:"required,oneof=direct group channel"`
	Name         string       `json:"name,omitempty" validate:"omitempty,min=1,max=128"`
	Description  string       `json:"description,omitempty" validate:"omitempty,max=512"`
	ChannelType  *ChannelType `json:"channel_type,omitempty" validate:"omitempty,oneof=public private moderated announcement"`
	IsInviteOnly bool         `json:"is_invite_only"`
	MemberIDs    []uuid.UUID  `json:"member_ids" validate:"required,min=1"`
}

type UpdateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=128"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=512"`
}

type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	Role   string    `json:"role" validate:"required,oneof=member admin"`
}

type ListFilter struct {
	UserID uuid.UUID
	Type   *Type
	Limit  int
	Offset int
}
