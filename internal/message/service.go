package message

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	"github.com/okrammeitei/chatgo/internal/conversation"
	"github.com/okrammeitei/chatgo/internal/notification"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
	ws "github.com/okrammeitei/chatgo/pkg/websocket"
)

type Service interface {
	Send(ctx context.Context, convID, senderID uuid.UUID, req *SendRequest) (*Message, error)
	Edit(ctx context.Context, msgID, senderID uuid.UUID, req *EditRequest) (*Message, error)
	Delete(ctx context.Context, msgID, senderID uuid.UUID) error
	List(ctx context.Context, f *ListFilter) ([]*Message, int64, error)
	AddReaction(ctx context.Context, msgID, userID uuid.UUID, req *AddReactionRequest) error
	RemoveReaction(ctx context.Context, msgID, userID uuid.UUID, emoji string) error
	Pin(ctx context.Context, msgID, actorID uuid.UUID) error
	Unpin(ctx context.Context, msgID, actorID uuid.UUID) error
	GetPinned(ctx context.Context, convID uuid.UUID) ([]*Message, error)
	MarkDelivered(ctx context.Context, msgID, userID uuid.UUID) error
	MarkRead(ctx context.Context, msgID, userID uuid.UUID) error
	BroadcastTyping(convID, userID uuid.UUID, isTyping bool)
}

type service struct {
	repo        Repository
	convRepo    conversation.Repository
	notifSvc    notification.Service
	activitySvc activitylog.Service
	hub         *ws.Hub
	log         *zap.Logger
}

func NewService(
	repo Repository,
	convRepo conversation.Repository,
	notifSvc notification.Service,
	activitySvc activitylog.Service,
	hub *ws.Hub,
	log *zap.Logger,
) Service {
	return &service{
		repo:        repo,
		convRepo:    convRepo,
		notifSvc:    notifSvc,
		activitySvc: activitySvc,
		hub:         hub,
		log:         log,
	}
}

func (s *service) Send(ctx context.Context, convID, senderID uuid.UUID, req *SendRequest) (*Message, error) {
	// Verify sender is a participant
	if _, err := s.convRepo.GetParticipant(ctx, convID, senderID); err != nil {
		return nil, apperr.Forbidden("not a conversation participant")
	}

	now := time.Now().UTC()
	msg := &Message{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       senderID,
		Content:        req.Content,
		ContentType:    req.ContentType,
		ParentID:       req.ParentID,
		Metadata:       req.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, msg); err != nil {
		return nil, err
	}

	// Update conversation last activity
	_ = s.convRepo.UpdateParticipantReadAt(ctx, convID, senderID, now)

	// Fan out to participants via WebSocket
	participants, _ := s.convRepo.ListParticipants(ctx, convID)
	recipientIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		recipientIDs = append(recipientIDs, p.UserID)
	}
	s.hub.SendToUsers(recipientIDs, WSEventMessageNew, msg)

	// Persist delivery status + send in-app notifications for offline users
	for _, p := range participants {
		if p.UserID == senderID {
			continue
		}
		_ = s.repo.UpsertStatus(ctx, &Status{
			MessageID: msg.ID, UserID: p.UserID,
			Status: StatusSent, UpdatedAt: now,
		})
		if p.NotificationsEnabled && !s.hub.IsOnline(p.UserID) {
			_ = s.notifSvc.CreateForMessage(ctx, p.UserID, msg)
		}
	}

	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &senderID, Action: activitylog.ActionMessageSent,
		ResourceType: "message", ResourceID: msg.ID.String(),
		Details: map[string]string{"conversation_id": convID.String()},
	})
	return msg, nil
}

func (s *service) Edit(ctx context.Context, msgID, senderID uuid.UUID, req *EditRequest) (*Message, error) {
	msg, err := s.repo.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg.SenderID != senderID {
		return nil, apperr.Forbidden("cannot edit another user's message")
	}
	if msg.IsDeleted {
		return nil, apperr.BadRequest("cannot edit a deleted message")
	}

	msg.Content = req.Content
	msg.IsEdited = true
	msg.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, msg); err != nil {
		return nil, err
	}

	participants, _ := s.convRepo.ListParticipants(ctx, msg.ConversationID)
	recipientIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		recipientIDs = append(recipientIDs, p.UserID)
	}
	s.hub.SendToUsers(recipientIDs, WSEventMessageEdit, msg)

	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &senderID, Action: activitylog.ActionMessageEdited,
		ResourceType: "message", ResourceID: msgID.String(),
	})
	return msg, nil
}

func (s *service) Delete(ctx context.Context, msgID, senderID uuid.UUID) error {
	msg, err := s.repo.GetByID(ctx, msgID)
	if err != nil {
		return err
	}
	if msg.SenderID != senderID {
		return apperr.Forbidden("cannot delete another user's message")
	}
	if err := s.repo.SoftDelete(ctx, msgID); err != nil {
		return err
	}

	participants, _ := s.convRepo.ListParticipants(ctx, msg.ConversationID)
	recipientIDs := make([]uuid.UUID, 0, len(participants))
	for _, p := range participants {
		recipientIDs = append(recipientIDs, p.UserID)
	}
	s.hub.SendToUsers(recipientIDs, WSEventMessageDelete, map[string]string{
		"message_id": msgID.String(), "conversation_id": msg.ConversationID.String(),
	})

	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &senderID, Action: activitylog.ActionMessageDeleted,
		ResourceType: "message", ResourceID: msgID.String(),
	})
	return nil
}

func (s *service) List(ctx context.Context, f *ListFilter) ([]*Message, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *service) AddReaction(ctx context.Context, msgID, userID uuid.UUID, req *AddReactionRequest) error {
	now := time.Now().UTC()
	reaction := &Reaction{
		ID: uuid.New(), MessageID: msgID, UserID: userID,
		Emoji: req.Emoji, CreatedAt: now,
	}
	if err := s.repo.AddReaction(ctx, reaction); err != nil {
		return err
	}
	msg, _ := s.repo.GetByID(ctx, msgID)
	if msg != nil {
		participants, _ := s.convRepo.ListParticipants(ctx, msg.ConversationID)
		ids := make([]uuid.UUID, 0, len(participants))
		for _, p := range participants {
			ids = append(ids, p.UserID)
		}
		s.hub.SendToUsers(ids, WSEventMessageReaction, reaction)
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &userID, Action: activitylog.ActionReactionAdded,
		ResourceType: "message", ResourceID: msgID.String(),
	})
	return nil
}

func (s *service) RemoveReaction(ctx context.Context, msgID, userID uuid.UUID, emoji string) error {
	return s.repo.RemoveReaction(ctx, msgID, userID, emoji)
}

func (s *service) Pin(ctx context.Context, msgID, actorID uuid.UUID) error {
	t := time.Now().UTC()
	if err := s.repo.UpdatePin(ctx, msgID, true, t); err != nil {
		return err
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &actorID, Action: activitylog.ActionMessagePinned,
		ResourceType: "message", ResourceID: msgID.String(),
	})
	return nil
}

func (s *service) Unpin(ctx context.Context, msgID, actorID uuid.UUID) error {
	return s.repo.UpdatePin(ctx, msgID, false, time.Now().UTC())
}

func (s *service) GetPinned(ctx context.Context, convID uuid.UUID) ([]*Message, error) {
	return s.repo.GetPinned(ctx, convID)
}

func (s *service) MarkDelivered(ctx context.Context, msgID, userID uuid.UUID) error {
	return s.repo.UpsertStatus(ctx, &Status{
		MessageID: msgID, UserID: userID,
		Status: StatusDelivered, UpdatedAt: time.Now().UTC(),
	})
}

func (s *service) MarkRead(ctx context.Context, msgID, userID uuid.UUID) error {
	return s.repo.UpsertStatus(ctx, &Status{
		MessageID: msgID, UserID: userID,
		Status: StatusRead, UpdatedAt: time.Now().UTC(),
	})
}

func (s *service) BroadcastTyping(convID, userID uuid.UUID, isTyping bool) {
	eventType := WSEventTypingStart
	if !isTyping {
		eventType = WSEventTypingStop
	}
	s.hub.SendToAll(eventType, map[string]string{
		"user_id": userID.String(), "conversation_id": convID.String(),
	})
}
