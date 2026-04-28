package auth

import (
	"net/http"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

func badValidation(err error) error {
	return apperr.BadRequest("validation failed: " + err.Error())
}

func errUnauthenticated() error {
	return apperr.Unauthorized("unauthenticated")
}

func errForbidden() error {
	return apperr.Forbidden("forbidden")
}

func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
