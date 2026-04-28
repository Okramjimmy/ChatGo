package notification

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
	mw "github.com/okrammeitei/chatgo/pkg/middleware"
)

type Handler struct {
	svc Service
	log *zap.Logger
}

func NewHandler(svc Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/unread-count", h.UnreadCount)
	r.Post("/read-all", h.MarkAllRead)
	r.Post("/{id}/read", h.MarkRead)
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	notifs, total, err := h.svc.List(r.Context(), &ListFilter{UserID: userID, Limit: 50})
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  notifs,
		"total": total,
	})
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	count, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, map[string]int64{"count": count})
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid notification id"))
		return
	}
	if err := h.svc.MarkRead(r.Context(), id, userID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	if err := h.svc.MarkAllRead(r.Context(), userID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
