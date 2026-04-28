package notification

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
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	MarkRead(ctx context.Context, id uuid.UUID, t time.Time) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	List(ctx context.Context, f *ListFilter) ([]*Notification, int64, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

const notifCols = `id, user_id, type, title, body, reference_id, reference_type, is_read, created_at, read_at`

func scanNotification(row pgx.Row) (*Notification, error) {
	n := &Notification{}
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body,
		&n.ReferenceID, &n.ReferenceType, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("notification")
	}
	return n, err
}

func (r *postgresRepository) Create(ctx context.Context, n *Notification) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, reference_id, reference_type, is_read, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		n.ID, n.UserID, n.Type, n.Title, n.Body,
		n.ReferenceID, n.ReferenceType, n.IsRead, n.CreatedAt,
	)
	return err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	row := r.db.QueryRow(ctx, "SELECT "+notifCols+" FROM notifications WHERE id=$1", id)
	return scanNotification(row)
}

func (r *postgresRepository) MarkRead(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE notifications SET is_read=true, read_at=$1 WHERE id=$2", t, id)
	return err
}

func (r *postgresRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE notifications SET is_read=true, read_at=NOW() WHERE user_id=$1 AND is_read=false", userID)
	return err
}

func (r *postgresRepository) List(ctx context.Context, f *ListFilter) ([]*Notification, int64, error) {
	where := "user_id = $1"
	args := []interface{}{f.UserID}
	i := 2

	if f.IsRead != nil {
		where += fmt.Sprintf(" AND is_read = $%d", i)
		args = append(args, *f.IsRead)
		i++
	}
	if f.Type != nil {
		where += fmt.Sprintf(" AND type = $%d", i)
		args = append(args, *f.Type)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(
		"SELECT %s FROM notifications WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		notifCols, where, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifs []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body,
			&n.ReferenceID, &n.ReferenceType, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, 0, err
		}
		notifs = append(notifs, n)
	}
	return notifs, total, rows.Err()
}

func (r *postgresRepository) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND is_read=false", userID).Scan(&count)
	return count, err
}
