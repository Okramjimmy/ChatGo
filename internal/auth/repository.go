package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Repository interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id uuid.UUID) (*Session, error)
	GetSessionByRefreshHash(ctx context.Context, hash string) (*Session, error)
	RevokeSession(ctx context.Context, id uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error
	ListActiveSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateSession(ctx context.Context, s *Session) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		s.ID, s.UserID, s.RefreshTokenHash, s.IPAddress, s.UserAgent, s.ExpiresAt, s.CreatedAt,
	)
	return err
}

func (r *postgresRepository) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	s := &Session{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at, revoked_at
		FROM sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.IPAddress, &s.UserAgent,
		&s.ExpiresAt, &s.CreatedAt, &s.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("session")
	}
	return s, err
}

func (r *postgresRepository) GetSessionByRefreshHash(ctx context.Context, hash string) (*Session, error) {
	s := &Session{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at, revoked_at
		FROM sessions WHERE refresh_token_hash = $1`, hash,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.IPAddress, &s.UserAgent,
		&s.ExpiresAt, &s.CreatedAt, &s.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("session")
	}
	return s, err
}

func (r *postgresRepository) RevokeSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		"UPDATE sessions SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL", now, id)
	return err
}

func (r *postgresRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		"UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL", now, userID)
	return err
}

func (r *postgresRepository) ListActiveSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, refresh_token_hash, ip_address, user_agent, expires_at, created_at, revoked_at
		FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		s := &Session{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.IPAddress, &s.UserAgent,
			&s.ExpiresAt, &s.CreatedAt, &s.RevokedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
