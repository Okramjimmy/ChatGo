package activitylog

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Action string
type Severity string

const (
	ActionUserLogin         Action = "user.login"
	ActionUserLogout        Action = "user.logout"
	ActionUserCreate        Action = "user.create"
	ActionUserUpdate        Action = "user.update"
	ActionUserDelete        Action = "user.delete"
	ActionPasswordChange    Action = "user.password_change"
	ActionMFAEnabled        Action = "user.mfa_enabled"
	ActionMFADisabled       Action = "user.mfa_disabled"
	ActionSessionRevoked    Action = "user.session_revoked"
	ActionMessageSent       Action = "message.sent"
	ActionMessageEdited     Action = "message.edited"
	ActionMessageDeleted    Action = "message.deleted"
	ActionMessagePinned     Action = "message.pinned"
	ActionReactionAdded     Action = "message.reaction_added"
	ActionFileUploaded      Action = "file.uploaded"
	ActionFileDownloaded    Action = "file.downloaded"
	ActionFileDeleted       Action = "file.deleted"
	ActionGroupCreated      Action = "group.created"
	ActionGroupDeleted      Action = "group.deleted"
	ActionChannelCreated    Action = "channel.created"
	ActionChannelDeleted    Action = "channel.deleted"
	ActionMemberAdded       Action = "member.added"
	ActionMemberRemoved     Action = "member.removed"
	ActionRoleChanged       Action = "user.role_changed"
	ActionPermissionUpdated Action = "permission.updated"
	ActionSystemError       Action = "system.error"
	ActionSecurityEvent     Action = "security.event"
	ActionConfigChange      Action = "config.change"
)

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

type ActivityLog struct {
	ID           uuid.UUID       `json:"id"`
	UserID       *uuid.UUID      `json:"user_id,omitempty"`
	Action       Action          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    string          `json:"user_agent"`
	Severity     Severity        `json:"severity"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Entry struct {
	UserID       *uuid.UUID
	Action       Action
	ResourceType string
	ResourceID   string
	Details      interface{}
	IPAddress    string
	UserAgent    string
	Severity     Severity
}

type Filter struct {
	UserID       *uuid.UUID
	Action       *Action
	ResourceType *string
	Severity     *Severity
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}
