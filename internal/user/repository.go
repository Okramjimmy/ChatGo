package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f *ListFilter) ([]*User, int64, error)
	GetRole(ctx context.Context, id uuid.UUID) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

const userColumns = `id, username, email, password_hash, display_name, avatar_url, status, role_id, mfa_enabled, mfa_secret, created_at, updated_at, deleted_at`

func scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Status, &u.RoleID, &u.MFAEnabled, &u.MFASecret,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("user")
	}
	return u, err
}

func (r *postgresRepository) Create(ctx context.Context, u *User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, display_name, avatar_url, status, role_id, mfa_enabled, mfa_secret, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		u.ID, u.Username, u.Email, u.PasswordHash, u.DisplayName, u.AvatarURL,
		u.Status, u.RoleID, u.MFAEnabled, u.MFASecret, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := r.db.QueryRow(ctx,
		"SELECT "+userColumns+" FROM users WHERE id = $1 AND deleted_at IS NULL", id)
	return scanUser(row)
}

func (r *postgresRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := r.db.QueryRow(ctx,
		"SELECT "+userColumns+" FROM users WHERE username = $1 AND deleted_at IS NULL", username)
	return scanUser(row)
}

func (r *postgresRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := r.db.QueryRow(ctx,
		"SELECT "+userColumns+" FROM users WHERE email = $1 AND deleted_at IS NULL", email)
	return scanUser(row)
}

func (r *postgresRepository) Update(ctx context.Context, u *User) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET display_name=$1, avatar_url=$2, status=$3, password_hash=$4,
		mfa_enabled=$5, mfa_secret=$6, updated_at=$7 WHERE id=$8 AND deleted_at IS NULL`,
		u.DisplayName, u.AvatarURL, u.Status, u.PasswordHash,
		u.MFAEnabled, u.MFASecret, u.UpdatedAt, u.ID,
	)
	return err
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
	return err
}

func (r *postgresRepository) List(ctx context.Context, f *ListFilter) ([]*User, int64, error) {
	where := "deleted_at IS NULL"
	args := []interface{}{}
	i := 1

	if f.Search != "" {
		where += fmt.Sprintf(" AND (username ILIKE $%d OR display_name ILIKE $%d)", i, i)
		args = append(args, "%"+f.Search+"%")
		i++
	}
	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, *f.Status)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(
		"SELECT %s FROM users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		userColumns, where, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName,
			&u.AvatarURL, &u.Status, &u.RoleID, &u.MFAEnabled, &u.MFASecret,
			&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *postgresRepository) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	role := &Role{}
	err := r.db.QueryRow(ctx,
		"SELECT id, name, description FROM roles WHERE id = $1", id,
	).Scan(&role.ID, &role.Name, &role.Description)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("role")
	}
	return role, err
}

func (r *postgresRepository) ListRoles(ctx context.Context) ([]*Role, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, description FROM roles ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		role := &Role{}
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *postgresRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE users SET role_id = $1, updated_at = NOW() WHERE id = $2", roleID, userID)
	return err
}
