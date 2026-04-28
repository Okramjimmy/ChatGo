package user

import (
	"fmt"
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
	// Self routes
	r.Get("/me", h.GetMe)
	r.Put("/me", h.UpdateMe)
	r.Post("/me/password", h.ChangePassword)

	// Admin / general routes
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)
	r.Put("/{id}/role", h.AssignRole)

	r.Get("/roles", h.ListRoles)
	return r
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	u, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	var req UpdateUserRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	u, err := h.svc.Update(r.Context(), userID, &req)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	var req ChangePasswordRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	if err := h.svc.ChangePassword(r.Context(), userID, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	u, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusCreated, u)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := intQuery(q.Get("limit"), 50)
	offset := intQuery(q.Get("offset"), 0)
	search := q.Get("search")

	users, total, err := h.svc.List(r.Context(), &ListFilter{
		Search: search,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":   users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid user id"))
		return
	}
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid user id"))
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	actorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid user id"))
		return
	}
	var body struct {
		RoleID uuid.UUID `json:"role_id" validate:"required"`
	}
	if err := mw.DecodeJSON(r, &body); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.svc.AssignRole(r.Context(), userID, body.RoleID, actorID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, roles)
}

func intQuery(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscan(s, &n); err != nil || n < 0 {
		return def
	}
	return n
}
