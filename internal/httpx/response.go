package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

type Response[T any] struct {
	Success bool           `json:"success"`
	Data    *T             `json:"data,omitempty"`
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

func (r Response[T]) WithAppError(appErr *apperror.Error) Response[T] {
	r.Success = false
	r.Data = nil

	if appErr == nil || appErr.HTTPStatus < http.StatusBadRequest ||
		appErr.HTTPStatus > 599 || appErr.HTTPStatus >= http.StatusInternalServerError ||
		appErr.Code == apperror.AppErrorInternal {
		r.Code = apperror.AppErrorInternal
		r.Message = "an internal error occurred"
		r.Details = nil
		return r
	}

	r.Code = appErr.Code
	r.Message = appErr.Message
	r.Details = appErr.Detail
	return r
}

func WriteSuccess[T any](w http.ResponseWriter, status int, data T) error {
	return writeJSON(w, status, Response[T]{
		Success: true,
		Data:    &data,
	})
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr == nil {
		log.Printf("unclassified HTTP error: %v", err)
		appErr = apperror.New(
			http.StatusInternalServerError,
			apperror.AppErrorInternal,
			"an internal error occurred",
			nil,
		)
	}

	status := appErr.HTTPStatus
	if status < http.StatusBadRequest || status > 599 || status >= http.StatusInternalServerError ||
		appErr.Code == apperror.AppErrorInternal {
		log.Printf("internal HTTP error: %v", err)
		status = http.StatusInternalServerError
	}

	response := (Response[any]{}).WithAppError(appErr)
	if writeErr := writeJSON(w, status, response); writeErr != nil {
		log.Printf("write HTTP error response: %v", writeErr)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("write JSON response: %w", err)
	}
	return nil
}
