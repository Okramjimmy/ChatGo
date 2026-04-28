package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	ws "github.com/okrammeitei/chatgo/pkg/websocket"
)

// MessageRef is a minimal interface to avoid circular imports with the message package.
type MessageRef interface {
	GetID() uuid.UUID
	GetConversationID() uuid.UUID
	GetSenderID() uuid.UUID
	GetContent() string
}

type Service interface {
	Create(ctx context.Context, req *CreateRequest) (*Notification, error)
	CreateForMessage(ctx context.Context, userID uuid.UUID, msg MessageRef) error
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	List(ctx context.Context, f *ListFilter) ([]*Notification, int64, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
}

type service struct {
	repo        Repository
	activitySvc activitylog.Service
	hub         *ws.Hub
	log         *zap.Logger
}

func NewService(repo Repository, activitySvc activitylog.Service, hub *ws.Hub, log *zap.Logger) Service {
	return &service{repo: repo, activitySvc: activitySvc, hub: hub, log: log}
}

func (s *service) Create(ctx context.Context, req *CreateRequest) (*Notification, error) {
	n := &Notification{
		ID:            uuid.New(),
		UserID:        req.UserID,
		Type:          req.Type,
		Title:         req.Title,
		Body:          req.Body,
		ReferenceID:   req.ReferenceID,
		ReferenceType: req.ReferenceType,
		IsRead:        false,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	s.hub.SendToUsers([]uuid.UUID{req.UserID}, "notification.new", n)
	return n, nil
}

func (s *service) CreateForMessage(ctx context.Context, userID uuid.UUID, msg MessageRef) error {
	n := &Notification{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          TypeMessage,
		Title:         "New message",
		Body:          truncate(msg.GetContent(), 128),
		ReferenceID:   msg.GetID().String(),
		ReferenceType: "message",
		IsRead:        false,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return err
	}
	s.hub.SendToUsers([]uuid.UUID{userID}, "notification.new", n)
	return nil
}

func (s *service) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if n.UserID != userID {
		return fmt.Errorf("forbidden")
	}
	return s.repo.MarkRead(ctx, id, time.Now().UTC())
}

func (s *service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *service) List(ctx context.Context, f *ListFilter) ([]*Notification, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *service) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
