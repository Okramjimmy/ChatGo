package message

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
	Create(ctx context.Context, m *Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*Message, error)
	Update(ctx context.Context, m *Message) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f *ListFilter) ([]*Message, int64, error)
	AddReaction(ctx context.Context, r *Reaction) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	ListReactions(ctx context.Context, messageID uuid.UUID) ([]*Reaction, error)
	UpsertStatus(ctx context.Context, s *Status) error
	GetPinned(ctx context.Context, conversationID uuid.UUID) ([]*Message, error)
	UpdatePin(ctx context.Context, messageID uuid.UUID, pinned bool, t time.Time) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

const msgColumns = `id, conversation_id, sender_id, content, content_type, parent_id, is_edited, is_deleted, is_pinned, metadata, created_at, updated_at`

func scanMessage(row pgx.Row) (*Message, error) {
	m := &Message{}
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.ContentType,
		&m.ParentID, &m.IsEdited, &m.IsDeleted, &m.IsPinned, &m.Metadata,
		&m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("message")
	}
	return m, err
}

func (r *postgresRepository) Create(ctx context.Context, m *Message) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO messages (id, conversation_id, sender_id, content, content_type, parent_id, is_edited, is_deleted, is_pinned, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		m.ID, m.ConversationID, m.SenderID, m.Content, m.ContentType, m.ParentID,
		m.IsEdited, m.IsDeleted, m.IsPinned, m.Metadata, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Message, error) {
	row := r.db.QueryRow(ctx, "SELECT "+msgColumns+" FROM messages WHERE id = $1", id)
	return scanMessage(row)
}

func (r *postgresRepository) Update(ctx context.Context, m *Message) error {
	_, err := r.db.Exec(ctx, `
		UPDATE messages SET content=$1, is_edited=$2, is_deleted=$3, is_pinned=$4, updated_at=$5
		WHERE id=$6`,
		m.Content, m.IsEdited, m.IsDeleted, m.IsPinned, m.UpdatedAt, m.ID)
	return err
}

func (r *postgresRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE messages SET is_deleted=true, content='', updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *postgresRepository) List(ctx context.Context, f *ListFilter) ([]*Message, int64, error) {
	where := fmt.Sprintf("conversation_id = $1 AND is_deleted = false")
	args := []interface{}{f.ConversationID}
	i := 2

	if f.BeforeID != nil {
		where += fmt.Sprintf(" AND created_at < (SELECT created_at FROM messages WHERE id = $%d)", i)
		args = append(args, *f.BeforeID)
		i++
	}
	if f.AfterID != nil {
		where += fmt.Sprintf(" AND created_at > (SELECT created_at FROM messages WHERE id = $%d)", i)
		args = append(args, *f.AfterID)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args = append(args, limit)
	rows, err := r.db.Query(ctx, fmt.Sprintf(
		"SELECT %s FROM messages WHERE %s ORDER BY created_at DESC LIMIT $%d",
		msgColumns, where, i), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.ContentType,
			&m.ParentID, &m.IsEdited, &m.IsDeleted, &m.IsPinned, &m.Metadata,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, err
		}
		msgs = append(msgs, m)
	}
	return msgs, total, rows.Err()
}

func (r *postgresRepository) AddReaction(ctx context.Context, rc *Reaction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO message_reactions (id, message_id, user_id, emoji, created_at)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (message_id, user_id, emoji) DO NOTHING`,
		rc.ID, rc.MessageID, rc.UserID, rc.Emoji, rc.CreatedAt,
	)
	return err
}

func (r *postgresRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM message_reactions WHERE message_id=$1 AND user_id=$2 AND emoji=$3",
		messageID, userID, emoji)
	return err
}

func (r *postgresRepository) ListReactions(ctx context.Context, messageID uuid.UUID) ([]*Reaction, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, message_id, user_id, emoji, created_at FROM message_reactions WHERE message_id=$1",
		messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []*Reaction
	for rows.Next() {
		rc := &Reaction{}
		if err := rows.Scan(&rc.ID, &rc.MessageID, &rc.UserID, &rc.Emoji, &rc.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, rc)
	}
	return reactions, rows.Err()
}

func (r *postgresRepository) UpsertStatus(ctx context.Context, s *Status) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO message_status (message_id, user_id, status, updated_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (message_id, user_id) DO UPDATE SET status=EXCLUDED.status, updated_at=EXCLUDED.updated_at`,
		s.MessageID, s.UserID, s.Status, s.UpdatedAt,
	)
	return err
}

func (r *postgresRepository) GetPinned(ctx context.Context, conversationID uuid.UUID) ([]*Message, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+msgColumns+" FROM messages WHERE conversation_id=$1 AND is_pinned=true AND is_deleted=false ORDER BY updated_at DESC",
		conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.ContentType,
			&m.ParentID, &m.IsEdited, &m.IsDeleted, &m.IsPinned, &m.Metadata,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *postgresRepository) UpdatePin(ctx context.Context, messageID uuid.UUID, pinned bool, t time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE messages SET is_pinned=$1, updated_at=$2 WHERE id=$3", pinned, t, messageID)
	return err
}
