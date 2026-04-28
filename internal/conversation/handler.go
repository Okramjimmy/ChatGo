package conversation

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
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	r.Post("/{id}/members", h.AddMember)
	r.Delete("/{id}/members/{userID}", h.RemoveMember)
	return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	var req CreateRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	conv, err := h.svc.Create(r.Context(), userID, &req)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusCreated, conv)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	convs, _, err := h.svc.ListForUser(r.Context(), &ListFilter{UserID: userID, Limit: 50})
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, convs)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	conv, err := h.svc.GetByID(r.Context(), id, userID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, conv)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	var req UpdateRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	conv, err := h.svc.Update(r.Context(), id, &req, userID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, conv)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	if err := h.svc.Delete(r.Context(), id, userID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	var req AddMemberRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	if err := h.svc.AddMember(r.Context(), convID, &req, actorID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid user id"))
		return
	}
	if err := h.svc.RemoveMember(r.Context(), convID, targetID, actorID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
