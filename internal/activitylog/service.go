package activitylog

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	Log(ctx context.Context, e *Entry) error
	List(ctx context.Context, filter *Filter) ([]*ActivityLog, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error)
}

type service struct {
	repo Repository
	log  *zap.Logger
}

func NewService(repo Repository, log *zap.Logger) Service {
	return &service{repo: repo, log: log}
}

func (s *service) Log(ctx context.Context, e *Entry) error {
	var details json.RawMessage
	if e.Details != nil {
		b, err := json.Marshal(e.Details)
		if err != nil {
			s.log.Warn("failed to marshal activity log details", zap.Error(err))
		} else {
			details = b
		}
	}

	sev := e.Severity
	if sev == "" {
		sev = SeverityInfo
	}

	l := &ActivityLog{
		ID:           uuid.New(),
		UserID:       e.UserID,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		Details:      details,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		Severity:     sev,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, l); err != nil {
		s.log.Error("failed to write activity log", zap.Error(err),
			zap.String("action", string(e.Action)))
		return err
	}
	return nil
}

func (s *service) List(ctx context.Context, filter *Filter) ([]*ActivityLog, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error) {
	return s.repo.GetByID(ctx, id)
}
