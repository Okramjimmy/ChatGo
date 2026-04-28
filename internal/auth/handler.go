package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"

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
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)

	// authenticated routes
	r.Group(func(r chi.Router) {
		r.Post("/logout", h.Logout)
		r.Post("/mfa/setup", h.SetupMFA)
		r.Post("/mfa/enable", h.EnableMFA)
		r.Post("/mfa/disable", h.DisableMFA)
		r.Delete("/sessions", h.RevokeAllSessions)
	})
	return r
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, badValidation(err))
		return
	}

	pair, err := h.svc.Login(r.Context(), &req, mw.RealIP(r), r.UserAgent())
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, pair)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, badValidation(err))
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken, mw.RealIP(r), r.UserAgent())
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, pair)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := mw.SessionIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, errUnauthenticated())
		return
	}
	if err := h.svc.Logout(r.Context(), sessionID, mw.RealIP(r), r.UserAgent()); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, errUnauthenticated())
		return
	}
	resp, err := h.svc.SetupMFA(r.Context(), userID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) EnableMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, errUnauthenticated())
		return
	}
	var req MFAVerifyRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.svc.EnableMFA(r.Context(), userID, req.Code); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, errUnauthenticated())
		return
	}
	var req MFAVerifyRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.svc.DisableMFA(r.Context(), userID, req.Code); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, errUnauthenticated())
		return
	}
	if err := h.svc.RevokeAllSessions(r.Context(), userID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
