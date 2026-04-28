package presence

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
	mw "github.com/okrammeitei/chatgo/pkg/middleware"
)

type Handler struct {
	svc      Service
	validate *validator.Validate
	log      *zap.Logger
}

func NewHandler(svc Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validate: validator.New(), log: log}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Put("/me", h.SetPresence)
	r.Get("/{userID}", h.GetPresence)
	r.Post("/bulk", h.GetBulkPresence)
	return r
}

func (h *Handler) SetPresence(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	var req UpdateRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.svc.SetPresence(r.Context(), userID, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetPresence(w http.ResponseWriter, r *http.Request) {
	_, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid user id"))
		return
	}
	p, err := h.svc.GetPresence(r.Context(), userID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) GetBulkPresence(w http.ResponseWriter, r *http.Request) {
	_, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	var body struct {
		UserIDs []uuid.UUID `json:"user_ids"`
	}
	if err := mw.DecodeJSON(r, &body); err != nil {
		mw.WriteError(w, err)
		return
	}
	presences, err := h.svc.GetBulkPresence(r.Context(), body.UserIDs)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, presences)
}
