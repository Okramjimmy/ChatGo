package conversation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Service interface {
	Create(ctx context.Context, creatorID uuid.UUID, req *CreateRequest) (*Conversation, error)
	GetByID(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) (*Conversation, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateRequest, actorID uuid.UUID) (*Conversation, error)
	Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
	ListForUser(ctx context.Context, f *ListFilter) ([]*Conversation, int64, error)
	AddMember(ctx context.Context, convID uuid.UUID, req *AddMemberRequest, actorID uuid.UUID) error
	RemoveMember(ctx context.Context, convID, targetID uuid.UUID, actorID uuid.UUID) error
	GetOrCreateDirect(ctx context.Context, userA, userB uuid.UUID) (*Conversation, error)
}

type service struct {
	repo        Repository
	activitySvc activitylog.Service
	log         *zap.Logger
}

func NewService(repo Repository, activitySvc activitylog.Service, log *zap.Logger) Service {
	return &service{repo: repo, activitySvc: activitySvc, log: log}
}

func (s *service) Create(ctx context.Context, creatorID uuid.UUID, req *CreateRequest) (*Conversation, error) {
	now := time.Now().UTC()
	conv := &Conversation{
		ID:           uuid.New(),
		Type:         req.Type,
		Name:         req.Name,
		Description:  req.Description,
		ChannelType:  req.ChannelType,
		IsInviteOnly: req.IsInviteOnly,
		CreatorID:    creatorID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, conv); err != nil {
		return nil, err
	}

	// Add creator as owner
	if err := s.repo.AddParticipant(ctx, &Participant{
		ID: uuid.New(), ConversationID: conv.ID, UserID: creatorID,
		Role: "owner", JoinedAt: now, LastReadAt: now, NotificationsEnabled: true,
	}); err != nil {
		return nil, err
	}

	// Add other members
	for _, memberID := range req.MemberIDs {
		if memberID == creatorID {
			continue
		}
		_ = s.repo.AddParticipant(ctx, &Participant{
			ID: uuid.New(), ConversationID: conv.ID, UserID: memberID,
			Role: "member", JoinedAt: now, LastReadAt: now, NotificationsEnabled: true,
		})
	}

	action := activitylog.ActionGroupCreated
	if req.Type == TypeChannel {
		action = activitylog.ActionChannelCreated
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &creatorID, Action: action,
		ResourceType: "conversation", ResourceID: conv.ID.String(),
		Details: map[string]string{"name": conv.Name, "type": string(conv.Type)},
	})
	return conv, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) (*Conversation, error) {
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Ensure requestor is a participant
	if _, err := s.repo.GetParticipant(ctx, id, requestorID); err != nil {
		return nil, apperr.Forbidden("not a participant")
	}
	participants, _ := s.repo.ListParticipants(ctx, id)
	conv.Participants = participants
	return conv, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req *UpdateRequest, actorID uuid.UUID) (*Conversation, error) {
	if err := s.requireRole(ctx, id, actorID, "admin", "owner"); err != nil {
		return nil, err
	}
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		conv.Name = *req.Name
	}
	if req.Description != nil {
		conv.Description = *req.Description
	}
	conv.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	if err := s.requireRole(ctx, id, actorID, "owner"); err != nil {
		return err
	}
	conv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	action := activitylog.ActionGroupDeleted
	if conv.Type == TypeChannel {
		action = activitylog.ActionChannelDeleted
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &actorID, Action: action,
		ResourceType: "conversation", ResourceID: id.String(),
	})
	return nil
}

func (s *service) ListForUser(ctx context.Context, f *ListFilter) ([]*Conversation, int64, error) {
	return s.repo.ListForUser(ctx, f)
}

func (s *service) AddMember(ctx context.Context, convID uuid.UUID, req *AddMemberRequest, actorID uuid.UUID) error {
	if err := s.requireRole(ctx, convID, actorID, "admin", "owner"); err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := s.repo.AddParticipant(ctx, &Participant{
		ID: uuid.New(), ConversationID: convID, UserID: req.UserID,
		Role: req.Role, JoinedAt: now, LastReadAt: now, NotificationsEnabled: true,
	}); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &actorID, Action: activitylog.ActionMemberAdded,
		ResourceType: "conversation", ResourceID: convID.String(),
		Details: map[string]string{"member_id": req.UserID.String()},
	})
	return nil
}

func (s *service) RemoveMember(ctx context.Context, convID, targetID uuid.UUID, actorID uuid.UUID) error {
	// Allow self-removal or admin/owner
	if targetID != actorID {
		if err := s.requireRole(ctx, convID, actorID, "admin", "owner"); err != nil {
			return err
		}
	}
	if err := s.repo.RemoveParticipant(ctx, convID, targetID); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &actorID, Action: activitylog.ActionMemberRemoved,
		ResourceType: "conversation", ResourceID: convID.String(),
		Details: map[string]string{"member_id": targetID.String()},
	})
	return nil
}

func (s *service) GetOrCreateDirect(ctx context.Context, userA, userB uuid.UUID) (*Conversation, error) {
	conv, err := s.repo.GetDirectConversation(ctx, userA, userB)
	if err == nil {
		return conv, nil
	}
	if !apperr.IsNotFound(err) {
		return nil, err
	}
	return s.Create(ctx, userA, &CreateRequest{
		Type: TypeDirect, MemberIDs: []uuid.UUID{userA, userB},
	})
}

func (s *service) requireRole(ctx context.Context, convID, userID uuid.UUID, roles ...string) error {
	p, err := s.repo.GetParticipant(ctx, convID, userID)
	if err != nil {
		return apperr.Forbidden("not a participant")
	}
	for _, r := range roles {
		if p.Role == r {
			return nil
		}
	}
	return apperr.Forbidden("insufficient role")
}
