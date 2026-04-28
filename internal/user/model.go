package user

import (
	"time"

	"github.com/google/uuid"
)

type Status string
type ConversationRole string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusLocked   Status = "locked"

	RoleMember ConversationRole = "member"
	RoleAdmin  ConversationRole = "admin"
	RoleOwner  ConversationRole = "owner"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	AvatarURL    string     `json:"avatar_url,omitempty"`
	Status       Status     `json:"status"`
	RoleID       uuid.UUID  `json:"role_id"`
	MFAEnabled   bool       `json:"mfa_enabled"`
	MFASecret    string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type Permission struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Resource string    `json:"resource"`
	Action   string    `json:"action"`
}

type CreateUserRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=32"`
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=64"`
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name,omitempty" validate:"omitempty,min=1,max=64"`
	AvatarURL   string `json:"avatar_url,omitempty" validate:"omitempty,url"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type ListFilter struct {
	Search string
	Status *Status
	Limit  int
	Offset int
}
