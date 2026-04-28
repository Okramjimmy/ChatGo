package file

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type Repository interface {
	Create(ctx context.Context, f *File) error
	GetByID(ctx context.Context, id uuid.UUID) (*File, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *ListFilter) ([]*File, int64, error)
	UpdateScanResult(ctx context.Context, id uuid.UUID, result ScanResult) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

const fileCols = `id, name, original_name, mime_type, size, storage_path, uploader_id, conversation_id, is_scanned, scan_result, access_level, created_at, deleted_at`

func scanFile(row pgx.Row) (*File, error) {
	f := &File{}
	err := row.Scan(&f.ID, &f.Name, &f.OriginalName, &f.MIMEType, &f.Size, &f.StoragePath,
		&f.UploaderID, &f.ConversationID, &f.IsScanned, &f.ScanResult, &f.AccessLevel,
		&f.CreatedAt, &f.DeletedAt)
	if err == pgx.ErrNoRows {
		return nil, apperr.NotFound("file")
	}
	return f, err
}

func (r *postgresRepository) Create(ctx context.Context, f *File) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO files (id, name, original_name, mime_type, size, storage_path, uploader_id, conversation_id, is_scanned, scan_result, access_level, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		f.ID, f.Name, f.OriginalName, f.MIMEType, f.Size, f.StoragePath,
		f.UploaderID, f.ConversationID, f.IsScanned, f.ScanResult, f.AccessLevel, f.CreatedAt,
	)
	return err
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*File, error) {
	row := r.db.QueryRow(ctx, "SELECT "+fileCols+" FROM files WHERE id=$1 AND deleted_at IS NULL", id)
	return scanFile(row)
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		"UPDATE files SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
	return err
}

func (r *postgresRepository) List(ctx context.Context, f *ListFilter) ([]*File, int64, error) {
	where := "deleted_at IS NULL"
	args := []interface{}{}
	i := 1

	if f.UploaderID != nil {
		where += fmt.Sprintf(" AND uploader_id = $%d", i)
		args = append(args, *f.UploaderID)
		i++
	}
	if f.ConversationID != nil {
		where += fmt.Sprintf(" AND conversation_id = $%d", i)
		args = append(args, *f.ConversationID)
		i++
	}
	if f.MIMEType != nil {
		where += fmt.Sprintf(" AND mime_type = $%d", i)
		args = append(args, *f.MIMEType)
		i++
	}

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM files WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(
		"SELECT %s FROM files WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		fileCols, where, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		file := &File{}
		if err := rows.Scan(&file.ID, &file.Name, &file.OriginalName, &file.MIMEType,
			&file.Size, &file.StoragePath, &file.UploaderID, &file.ConversationID,
			&file.IsScanned, &file.ScanResult, &file.AccessLevel, &file.CreatedAt, &file.DeletedAt); err != nil {
			return nil, 0, err
		}
		files = append(files, file)
	}
	return files, total, rows.Err()
}

func (r *postgresRepository) UpdateScanResult(ctx context.Context, id uuid.UUID, result ScanResult) error {
	_, err := r.db.Exec(ctx,
		"UPDATE files SET is_scanned=true, scan_result=$1 WHERE id=$2", result, id)
	return err
}
