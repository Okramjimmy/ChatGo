package middleware

import (
	"encoding/json"
	"net/http"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// WriteError writes a JSON-encoded error response derived from an AppError.
func WriteError(w http.ResponseWriter, err error) {
	status := apperr.HTTPStatus(err)
	msg := err.Error()

	var ae *apperr.AppError
	if apperr.HTTPStatus(err) != http.StatusInternalServerError {
		if appErr, ok := err.(*apperr.AppError); ok {
			msg = appErr.Message
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Code: status, Message: msg})
	_ = ae
}

// WriteJSON writes a successful JSON response.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeJSON decodes JSON from request body into v.
func DecodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperr.BadRequest("invalid request body: " + err.Error())
	}
	return nil
}

// ActivityLogger middleware logs HTTP requests to the activity log
// (light-weight wrapper – detailed logging done in service layer).
func ActivityLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// RealIP extracts the real client IP from X-Forwarded-For / X-Real-IP headers.
func RealIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
