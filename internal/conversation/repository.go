package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, c *Conversation) error
	GetByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	Update(ctx context.Context, c *Conversation) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListForUser(ctx context.Context, f *ListFilter) ([]*Conversation, int64, error)
	AddParticipant(ctx context.Context, p *Participant) error
	RemoveParticipant(ctx context.Context, convID, userID uuid.UUID) error
	GetParticipant(ctx context.Context, convID, userID uuid.UUID) (*Participant, error)
	ListParticipants(ctx context.Context, convID uuid.UUID) ([]*Participant, error)
	UpdateParticipantReadAt(ctx context.Context, convID, userID uuid.UUID, t time.Time) error
	GetDirectConversation(ctx context.Context, userA, userB uuid.UUID) (*Conversation, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, c *Conversation) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO conversations (id, type, name, description, channel_type, is_invite_only, creator_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		c.ID, c.Type, c.Name, c.Description, c.ChannelType, c.IsInviteOnly,
		c.CreatorID, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Conversation, error) {
	c := &Conversation{}
	err := r.db.QueryRow(ctx, `
		SELECT id, type, name, description, channel_type, is_invite_only, creator_id, created_at, updated_at, deleted_at
		FROM conversations WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.ChannelType, &c.IsInviteOnly,
		&c.CreatorID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("conversation")
	}
	return c, err
}

func (r *postgresRepository) Update(ctx context.Context, c *Conversation) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversations SET name=$1, description=$2, updated_at=$3
		WHERE id=$4 AND deleted_at IS NULL`,
		c.Name, c.Description, c.UpdatedAt, c.ID)
	return err
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE conversations SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
	return err
}

func (r *postgresRepository) ListForUser(ctx context.Context, f *ListFilter) ([]*Conversation, int64, error) {
	where := "p.user_id = $1 AND c.deleted_at IS NULL"
	args := []interface{}{f.UserID}
	i := 2

	if f.Type != nil {
		where += fmt.Sprintf(" AND c.type = $%d", i)
		args = append(args, *f.Type)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx, fmt.Sprintf(
		"SELECT COUNT(*) FROM conversations c JOIN participants p ON p.conversation_id = c.id WHERE %s", where),
		args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT c.id, c.type, c.name, c.description, c.channel_type, c.is_invite_only, c.creator_id, c.created_at, c.updated_at, c.deleted_at
		FROM conversations c JOIN participants p ON p.conversation_id = c.id
		WHERE %s ORDER BY c.updated_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		c := &Conversation{}
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.ChannelType, &c.IsInviteOnly,
			&c.CreatorID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, 0, err
		}
		convs = append(convs, c)
	}
	return convs, total, rows.Err()
}

func (r *postgresRepository) AddParticipant(ctx context.Context, p *Participant) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO participants (id, conversation_id, user_id, role, joined_at, last_read_at, is_muted, notifications_enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		p.ID, p.ConversationID, p.UserID, p.Role, p.JoinedAt, p.LastReadAt,
		p.IsMuted, p.NotificationsEnabled,
	)
	return err
}

func (r *postgresRepository) RemoveParticipant(ctx context.Context, convID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM participants WHERE conversation_id = $1 AND user_id = $2", convID, userID)
	return err
}

func (r *postgresRepository) GetParticipant(ctx context.Context, convID, userID uuid.UUID) (*Participant, error) {
	p := &Participant{}
	err := r.db.QueryRow(ctx, `
		SELECT id, conversation_id, user_id, role, joined_at, last_read_at, is_muted, notifications_enabled
		FROM participants WHERE conversation_id = $1 AND user_id = $2`, convID, userID,
	).Scan(&p.ID, &p.ConversationID, &p.UserID, &p.Role, &p.JoinedAt, &p.LastReadAt,
		&p.IsMuted, &p.NotificationsEnabled)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("participant")
	}
	return p, err
}

func (r *postgresRepository) ListParticipants(ctx context.Context, convID uuid.UUID) ([]*Participant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, user_id, role, joined_at, last_read_at, is_muted, notifications_enabled
		FROM participants WHERE conversation_id = $1`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*Participant
	for rows.Next() {
		p := &Participant{}
		if err := rows.Scan(&p.ID, &p.ConversationID, &p.UserID, &p.Role, &p.JoinedAt,
			&p.LastReadAt, &p.IsMuted, &p.NotificationsEnabled); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (r *postgresRepository) UpdateParticipantReadAt(ctx context.Context, convID, userID uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE participants SET last_read_at = $1 WHERE conversation_id = $2 AND user_id = $3", t, convID, userID)
	return err
}

func (r *postgresRepository) GetDirectConversation(ctx context.Context, userA, userB uuid.UUID) (*Conversation, error) {
	c := &Conversation{}
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.type, c.name, c.description, c.channel_type, c.is_invite_only, c.creator_id, c.created_at, c.updated_at, c.deleted_at
		FROM conversations c
		JOIN participants pa ON pa.conversation_id = c.id AND pa.user_id = $1
		JOIN participants pb ON pb.conversation_id = c.id AND pb.user_id = $2
		WHERE c.type = 'direct' AND c.deleted_at IS NULL
		LIMIT 1`, userA, userB,
	).Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.ChannelType, &c.IsInviteOnly,
		&c.CreatorID, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("conversation")
	}
	return c, err
}
