package search

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
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
	r.Get("/", h.Search)
	return r
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	_, ok := mw.UserIDFromCtx(r.Context())
	if !ok {
		mw.WriteError(w, apperr.Unauthorized("unauthenticated"))
		return
	}

	q := r.URL.Query()
	term := q.Get("q")
	if term == "" {
		mw.WriteError(w, apperr.BadRequest("query parameter 'q' is required"))
		return
	}

	query := &Query{
		Term:         term,
		ResourceType: ResourceType(q.Get("type")),
		Limit:        20,
	}
	if err := h.validate.Struct(query); err != nil {
		mw.WriteError(w, apperr.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.Search(r.Context(), query)
	if err != nil {
		mw.WriteError(w, err)
		return
	}
	mw.WriteJSON(w, http.StatusOK, resp)
}
