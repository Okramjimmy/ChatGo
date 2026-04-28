package presence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/okrammeitei/chatgo/pkg/cache"
	ws "github.com/okrammeitei/chatgo/pkg/websocket"
)

type Service interface {
	SetPresence(ctx context.Context, userID uuid.UUID, req *UpdateRequest) error
	GetPresence(ctx context.Context, userID uuid.UUID) (*Presence, error)
	GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) ([]*Presence, error)
	SetOffline(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	cache *cache.Client
	hub   *ws.Hub
	log   *zap.Logger
}

func NewService(cache *cache.Client, hub *ws.Hub, log *zap.Logger) Service {
	return &service{cache: cache, hub: hub, log: log}
}

func presenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("presence:%s", userID.String())
}

func (s *service) SetPresence(ctx context.Context, userID uuid.UUID, req *UpdateRequest) error {
	p := &Presence{
		UserID:   userID,
		Status:   req.Status,
		LastSeen: time.Now().UTC(),
		DeviceID: req.DeviceID,
	}
	if err := s.cache.Set(ctx, presenceKey(userID), p, 5*time.Minute); err != nil {
		return err
	}
	s.hub.SendToAll("presence.update", p)
	return nil
}

func (s *service) GetPresence(ctx context.Context, userID uuid.UUID) (*Presence, error) {
	p := &Presence{}
	err := s.cache.Get(ctx, presenceKey(userID), p)
	if s.cache.IsNotFound(err) {
		// Derive from hub connection state
		status := StatusOffline
		if s.hub.IsOnline(userID) {
			status = StatusOnline
		}
		return &Presence{UserID: userID, Status: status, LastSeen: time.Now().UTC()}, nil
	}
	return p, err
}

func (s *service) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) ([]*Presence, error) {
	presences := make([]*Presence, 0, len(userIDs))
	for _, id := range userIDs {
		p, err := s.GetPresence(ctx, id)
		if err != nil {
			s.log.Warn("failed to get presence", zap.String("user_id", id.String()), zap.Error(err))
			continue
		}
		presences = append(presences, p)
	}
	return presences, nil
}

func (s *service) SetOffline(ctx context.Context, userID uuid.UUID) error {
	p := &Presence{
		UserID:   userID,
		Status:   StatusOffline,
		LastSeen: time.Now().UTC(),
	}
	if err := s.cache.Set(ctx, presenceKey(userID), p, 24*time.Hour); err != nil {
		return err
	}
	s.hub.SendToAll("presence.update", p)
	return nil
}
