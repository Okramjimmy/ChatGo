package search

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Service interface {
	Search(ctx context.Context, q *Query) (*Response, error)
}

type service struct {
	db  *pgxpool.Pool
	log *zap.Logger
}

func NewService(db *pgxpool.Pool, log *zap.Logger) Service {
	return &service{db: db, log: log}
}

func (s *service) Search(ctx context.Context, q *Query) (*Response, error) {
	limit := q.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := q.Offset

	var results []*Result
	var total int64

	switch q.ResourceType {
	case ResourceMessage, "":
		r, t, err := s.searchMessages(ctx, q.Term, q.ConversationID, limit, offset)
		if err != nil {
			return nil, err
		}
		results = append(results, r...)
		total += t
		if q.ResourceType != "" {
			break
		}
		fallthrough
	case ResourceUser:
		r, t, err := s.searchUsers(ctx, q.Term, limit, offset)
		if err != nil {
			s.log.Warn("user search error", zap.Error(err))
		} else {
			results = append(results, r...)
			total += t
		}
		if q.ResourceType != "" {
			break
		}
		fallthrough
	case ResourceChannel:
		r, t, err := s.searchChannels(ctx, q.Term, limit, offset)
		if err != nil {
			s.log.Warn("channel search error", zap.Error(err))
		} else {
			results = append(results, r...)
			total += t
		}
	}

	return &Response{Results: results, Total: total}, nil
}

func (s *service) searchMessages(ctx context.Context, term string, convID interface{}, limit, offset int) ([]*Result, int64, error) {
	query := `
		SELECT id::text, content, ts_rank(to_tsvector('english', content), plainto_tsquery('english', $1)) AS score
		FROM messages
		WHERE to_tsvector('english', content) @@ plainto_tsquery('english', $1)
		  AND is_deleted = false
		ORDER BY score DESC LIMIT $2 OFFSET $3`
	rows, err := s.db.Query(ctx, query, term, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		r := &Result{ResourceType: ResourceMessage}
		var score float64
		if err := rows.Scan(&r.ID, &r.Excerpt, &score); err != nil {
			return nil, 0, err
		}
		r.Score = score
		r.Title = truncate(r.Excerpt, 80)
		results = append(results, r)
	}
	return results, int64(len(results)), rows.Err()
}

func (s *service) searchUsers(ctx context.Context, term string, limit, offset int) ([]*Result, int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, username, display_name FROM users
		WHERE (username ILIKE $1 OR display_name ILIKE $1) AND deleted_at IS NULL
		LIMIT $2 OFFSET $3`, "%"+term+"%", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		r := &Result{ResourceType: ResourceUser}
		var username, displayName string
		if err := rows.Scan(&r.ID, &username, &displayName); err != nil {
			return nil, 0, err
		}
		r.Title = displayName
		r.Excerpt = username
		results = append(results, r)
	}
	return results, int64(len(results)), rows.Err()
}

func (s *service) searchChannels(ctx context.Context, term string, limit, offset int) ([]*Result, int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, description FROM conversations
		WHERE type = 'channel' AND (name ILIKE $1 OR description ILIKE $1) AND deleted_at IS NULL
		LIMIT $2 OFFSET $3`, "%"+term+"%", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		r := &Result{ResourceType: ResourceChannel}
		if err := rows.Scan(&r.ID, &r.Title, &r.Excerpt); err != nil {
			return nil, 0, err
		}
		results = append(results, r)
	}
	return results, int64(len(results)), rows.Err()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
