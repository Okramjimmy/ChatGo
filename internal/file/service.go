package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/okrammeitei/chatgo/internal/activitylog"
	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

// VirusScanner is an interface for pluggable AV scanning (e.g. ClamAV).
type VirusScanner interface {
	Scan(path string) (ScanResult, error)
}

type Service interface {
	Upload(ctx context.Context, uploaderID uuid.UUID, convID *uuid.UUID, originalName, mimeType string, size int64, r io.Reader) (*File, error)
	GetByID(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) (*File, error)
	Download(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) (io.ReadCloser, *File, error)
	Delete(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) error
	List(ctx context.Context, f *ListFilter) ([]*File, int64, error)
}

type service struct {
	repo        Repository
	activitySvc activitylog.Service
	scanner     VirusScanner
	storagePath string
	maxSize     int64
	log         *zap.Logger
}

func NewService(
	repo Repository,
	activitySvc activitylog.Service,
	scanner VirusScanner,
	storagePath string,
	maxSize int64,
	log *zap.Logger,
) Service {
	return &service{
		repo:        repo,
		activitySvc: activitySvc,
		scanner:     scanner,
		storagePath: storagePath,
		maxSize:     maxSize,
		log:         log,
	}
}

func (s *service) Upload(ctx context.Context, uploaderID uuid.UUID, convID *uuid.UUID, originalName, mimeType string, size int64, r io.Reader) (*File, error) {
	if size > s.maxSize {
		return nil, apperr.BadRequest(fmt.Sprintf("file exceeds maximum size of %d bytes", s.maxSize))
	}

	id := uuid.New()
	ext := filepath.Ext(originalName)
	storeName := id.String() + ext
	dir := filepath.Join(s.storagePath, id.String()[:2])
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, apperr.Internal(err)
	}
	path := filepath.Join(dir, storeName)

	dst, err := os.Create(path) // #nosec G304 -- path is constructed from UUID
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, io.LimitReader(r, s.maxSize+1)); err != nil {
		_ = os.Remove(path)
		return nil, apperr.Internal(err)
	}

	scanResult := ScanSkipped
	if s.scanner != nil {
		scanResult, err = s.scanner.Scan(path)
		if err != nil {
			s.log.Warn("virus scan failed", zap.Error(err))
			scanResult = ScanPending
		}
		if scanResult == ScanInfected {
			_ = os.Remove(path)
			return nil, apperr.BadRequest("file rejected: virus detected")
		}
	}

	f := &File{
		ID:             id,
		Name:           storeName,
		OriginalName:   originalName,
		MIMEType:       mimeType,
		Size:           size,
		StoragePath:    path,
		UploaderID:     uploaderID,
		ConversationID: convID,
		IsScanned:      s.scanner != nil,
		ScanResult:     scanResult,
		AccessLevel:    AccessPrivate,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, f); err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &uploaderID, Action: activitylog.ActionFileUploaded,
		ResourceType: "file", ResourceID: id.String(),
		Details: map[string]interface{}{"name": originalName, "size": size, "mime": mimeType},
	})
	return f, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID, _ uuid.UUID) (*File, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Download(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) (io.ReadCloser, *File, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := os.Open(f.StoragePath) // #nosec G304
	if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &requestorID, Action: activitylog.ActionFileDownloaded,
		ResourceType: "file", ResourceID: id.String(),
	})
	return rc, f, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID, requestorID uuid.UUID) error {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if f.UploaderID != requestorID {
		return apperr.Forbidden("cannot delete another user's file")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = os.Remove(f.StoragePath)
	_ = s.activitySvc.Log(ctx, &activitylog.Entry{
		UserID: &requestorID, Action: activitylog.ActionFileDeleted,
		ResourceType: "file", ResourceID: id.String(),
	})
	return nil
}

func (s *service) List(ctx context.Context, f *ListFilter) ([]*File, int64, error) {
	return s.repo.List(ctx, f)
}
