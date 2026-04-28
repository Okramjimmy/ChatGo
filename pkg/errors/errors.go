package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) HTTPStatus() int { return e.Code }

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func NotFound(resource string) *AppError {
	return New(http.StatusNotFound, resource+" not found")
}

func Unauthorized(msg string) *AppError {
	return New(http.StatusUnauthorized, msg)
}

func Forbidden(msg string) *AppError {
	return New(http.StatusForbidden, msg)
}

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, msg)
}

func Conflict(msg string) *AppError {
	return New(http.StatusConflict, msg)
}

func Internal(err error) *AppError {
	return Wrap(http.StatusInternalServerError, "internal server error", err)
}

func IsNotFound(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == http.StatusNotFound
}

func IsConflict(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == http.StatusConflict
}

func HTTPStatus(err error) int {
	var e *AppError
	if errors.As(err, &e) {
		return e.Code
	}
	return http.StatusInternalServerError
}
