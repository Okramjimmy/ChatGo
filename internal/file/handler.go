package file

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
	mw "github.com/okrammeitei/chatgo/pkg/middleware"
)

type Handler struct {
	svc     Service
	maxSize int64
	log     *zap.Logger
}

func NewHandler(svc Service, maxSize int64, log *zap.Logger) *Handler {
	return &Handler{svc: svc, maxSize: maxSize, log: log}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Upload)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetMeta)
	r.Get("/{id}/download", h.Download)
	r.Delete("/{id}", h.Delete)
	return r
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	uploaderID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}

	// 32 MB parse limit
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		mw.WriteError(w, apperr.BadRequest("failed to parse form: "+err.Error()))
		return
	}

	fh, header, err := r.FormFile("file")
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("file field required"))
		return
	}
	defer fh.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Optional conversation_id
	var convID *uuid.UUID
	if cidStr := r.FormValue("conversation_id"); cidStr != "" {
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			mw.WriteError(w, apperr.BadRequest("invalid conversation_id"))
			return
		}
		convID = &cid
	}

	f, err := h.svc.Upload(r.Context(), uploaderID, convID, header.Filename, mimeType, header.Size, fh)
	if err != nil {
		mw.WriteError(w, err)
		return
	}

	downloadURL := fmt.Sprintf("/api/v1/files/%s/download", f.ID.String())
	mw.WriteJSON(w, http.StatusCreated, &UploadResponse{File: f, DownloadURL: downloadURL})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	uploaderID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	files, total, err := h.svc.List(r.Context(), &ListFilter{UploaderID: &uploaderID, Limit: 50})
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  files,
		"total": total,
	})
}

func (h *Handler) GetMeta(w http.ResponseWriter, r *http.Request) {
	requestorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid file id"))
		return
	}
	f, err := h.svc.GetByID(r.Context(), id, requestorID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, f)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	requestorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid file id"))
		return
	}
	rc, f, err := h.svc.Download(r.Context(), id, requestorID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", f.MIMEType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, f.OriginalName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", f.Size))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	requestorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid file id"))
		return
	}
	if err := h.svc.Delete(r.Context(), id, requestorID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
