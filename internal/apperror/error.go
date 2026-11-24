package apperror

import (
	"errors"
	"fmt"
)

const (
	AppErrorInvalidArgument      = "invalid_argument"
	AppErrorUnauthenticated      = "unauthenticated"
	AppErrorForbidden            = "forbidden"
	AppErrorNotFound             = "not_found"
	AppErrorConflict             = "conflict"
	AppErrorUnavailable          = "unavailable"
	AppErrorUnsupportedMediaType = "unsupported_media_type"
	AppErrorRequestTooLarge      = "request_too_large"
	AppErrorInternal             = "internal"
)

type Error struct {
	HTTPStatus int            `json:"http_status"`
	Detail     map[string]any `json:"detail"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Err        error          `json:"-"`
}

func New(httpStatus int, code, message string, detail map[string]any) *Error {
	return &Error{
		HTTPStatus: httpStatus,
		Detail:     cloneDetail(detail),
		Code:       code,
		Message:    message,
	}
}

func Wrap(httpStatus int, code, message string, detail map[string]any, cause error) *Error {
	return &Error{
		HTTPStatus: httpStatus,
		Detail:     cloneDetail(detail),
		Code:       code,
		Message:    message,
		Err:        cause,
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsCode(err error, code string) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr != nil && appErr.Code == code
}

func cloneDetail(detail map[string]any) map[string]any {
	if detail == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(detail))
	for key, value := range detail {
		cloned[key] = value
	}
	return cloned
}
