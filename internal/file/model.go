package file

import (
	"time"

	"github.com/google/uuid"
)

type AccessLevel string
type ScanResult string

const (
	AccessPublic     AccessLevel = "public"
	AccessPrivate    AccessLevel = "private"
	AccessRestricted AccessLevel = "restricted"

	ScanClean    ScanResult = "clean"
	ScanInfected ScanResult = "infected"
	ScanPending  ScanResult = "pending"
	ScanSkipped  ScanResult = "skipped"
)

type File struct {
	ID             uuid.UUID   `json:"id"`
	Name           string      `json:"name"`
	OriginalName   string      `json:"original_name"`
	MIMEType       string      `json:"mime_type"`
	Size           int64       `json:"size"`
	StoragePath    string      `json:"-"`
	UploaderID     uuid.UUID   `json:"uploader_id"`
	ConversationID *uuid.UUID  `json:"conversation_id,omitempty"`
	IsScanned      bool        `json:"is_scanned"`
	ScanResult     ScanResult  `json:"scan_result"`
	AccessLevel    AccessLevel `json:"access_level"`
	CreatedAt      time.Time   `json:"created_at"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
}

type UploadResponse struct {
	File    *File  `json:"file"`
	DownloadURL string `json:"download_url"`
}

type ListFilter struct {
	UploaderID     *uuid.UUID
	ConversationID *uuid.UUID
	MIMEType       *string
	Limit          int
	Offset         int
}
