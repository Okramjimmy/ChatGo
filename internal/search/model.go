package search

import "github.com/google/uuid"

type ResourceType string

const (
	ResourceMessage      ResourceType = "message"
	ResourceUser         ResourceType = "user"
	ResourceChannel      ResourceType = "channel"
	ResourceFile         ResourceType = "file"
	ResourceActivityLog  ResourceType = "activity_log"
)

type Query struct {
	Term         string       `json:"term" validate:"required,min=1,max=256"`
	ResourceType ResourceType `json:"resource_type,omitempty" validate:"omitempty,oneof=message user channel file activity_log"`
	// Scoped to a conversation for message searches
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
}

type Result struct {
	ResourceType ResourceType `json:"resource_type"`
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Excerpt      string       `json:"excerpt"`
	Score        float64      `json:"score,omitempty"`
}

type Response struct {
	Results []*Result `json:"results"`
	Total   int64     `json:"total"`
}
