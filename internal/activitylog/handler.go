package activitylog

import (
	"net/http"
	"time"

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
	r.Get("/{id}", h.GetByID)
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	_, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}

	q := r.URL.Query()
	f := &Filter{
		Limit:  50,
		Offset: 0,
	}

	if v := q.Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err == nil {
			f.UserID = &uid
		}
	}
	if v := q.Get("action"); v != "" {
		a := Action(v)
		f.Action = &a
	}
	if v := q.Get("resource_type"); v != "" {
		f.ResourceType = &v
	}
	if v := q.Get("severity"); v != "" {
		s := Severity(v)
		f.Severity = &s
	}
	if v := q.Get("start"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			f.StartTime = &t
		}
	}
	if v := q.Get("end"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			f.EndTime = &t
		}
	}

	logs, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":   logs,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	_, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		mw.WriteError(w, apperr.BadRequest("invalid log id"))
		return
	}
	log, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, log)
}
