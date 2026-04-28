package activitylog

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, log *ActivityLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error)
	List(ctx context.Context, filter *Filter) ([]*ActivityLog, int64, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, l *ActivityLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO activity_logs
			(id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, severity, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		l.ID, l.UserID, l.Action, l.ResourceType, l.ResourceID,
		l.Details, l.IPAddress, l.UserAgent, l.Severity, l.CreatedAt,
	)
	return err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*ActivityLog, error) {
	l := &ActivityLog{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, severity, created_at
		FROM activity_logs WHERE id = $1`, id,
	).Scan(&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID,
		&l.Details, &l.IPAddress, &l.UserAgent, &l.Severity, &l.CreatedAt)
	return l, err
}

func (r *postgresRepository) List(ctx context.Context, f *Filter) ([]*ActivityLog, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	i := 1

	if f.UserID != nil {
		where = append(where, fmt.Sprintf("user_id = $%d", i))
		args = append(args, f.UserID)
		i++
	}
	if f.Action != nil {
		where = append(where, fmt.Sprintf("action = $%d", i))
		args = append(args, *f.Action)
		i++
	}
	if f.ResourceType != nil {
		where = append(where, fmt.Sprintf("resource_type = $%d", i))
		args = append(args, *f.ResourceType)
		i++
	}
	if f.Severity != nil {
		where = append(where, fmt.Sprintf("severity = $%d", i))
		args = append(args, *f.Severity)
		i++
	}
	if f.StartTime != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, f.StartTime)
		i++
	}
	if f.EndTime != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", i))
		args = append(args, f.EndTime)
		i++
	}

	clause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM activity_logs WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, severity, created_at
		FROM activity_logs WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		clause, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*ActivityLog
	for rows.Next() {
		l := &ActivityLog{}
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID,
			&l.Details, &l.IPAddress, &l.UserAgent, &l.Severity, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}
