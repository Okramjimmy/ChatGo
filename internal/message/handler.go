package message

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

// Routes mounts under /conversations/{convID}/messages
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Send)
	r.Get("/pinned", h.GetPinned)

	r.Put("/{msgID}", h.Edit)
	r.Delete("/{msgID}", h.Delete)
	r.Post("/{msgID}/pin", h.Pin)
	r.Delete("/{msgID}/pin", h.Unpin)
	r.Post("/{msgID}/reactions", h.AddReaction)
	r.Delete("/{msgID}/reactions/{emoji}", h.RemoveReaction)
	r.Post("/{msgID}/read", h.MarkRead)

	// typing indicator (WebSocket events only, but also exposed via REST)
	r.Post("/typing", h.Typing)
	return r
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	senderID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	convID, err := uuid.Parse(chi.URLParam(r, "convID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	var req SendRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	msg, err := h.svc.Send(r.Context(), convID, senderID, &req)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusCreated, msg)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(chi.URLParam(r, "convID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	msgs, _, err := h.svc.List(r.Context(), &ListFilter{ConversationID: convID, Limit: 50})
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, msgs)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	senderID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	var req EditRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}
	msg, err := h.svc.Edit(r.Context(), msgID, senderID, &req)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, msg)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	senderID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	if err := h.svc.Delete(r.Context(), msgID, senderID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetPinned(w http.ResponseWriter, r *http.Request) {
	convID, err := uuid.Parse(chi.URLParam(r, "convID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	msgs, err := h.svc.GetPinned(r.Context(), convID)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, msgs)
}

func (h *Handler) Pin(w http.ResponseWriter, r *http.Request) {
	actorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	if err := h.svc.Pin(r.Context(), msgID, actorID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Unpin(w http.ResponseWriter, r *http.Request) {
	actorID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	if err := h.svc.Unpin(r.Context(), msgID, actorID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	var req AddReactionRequest
	if err := mw.DecodeJSON(r, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	if err := h.svc.AddReaction(r.Context(), msgID, userID, &req); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	emoji := chi.URLParam(r, "emoji")
	if emoji == "" {
		mw.WriteError(w, apperr.BadRequest("emoji required"))
		return
	}
	if err := h.svc.RemoveReaction(r.Context(), msgID, userID, emoji); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "msgID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid message id"))
		return
	}
	if err := h.svc.MarkRead(r.Context(), msgID, userID); err != nil {
		mw.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Typing(w http.ResponseWriter, r *http.Request) {
	userID, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	convID, err := uuid.Parse(chi.URLParam(r, "convID"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid conversation id"))
		return
	}
	var body struct {
		IsTyping bool `json:"is_typing"`
	}
	_ = mw.DecodeJSON(r, &body)
	h.svc.BroadcastTyping(convID, userID, body.IsTyping)
	w.WriteHeader(http.StatusNoContent)
}
