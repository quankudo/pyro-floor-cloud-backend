package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var ErrInvalidBodyLimit = errors.New("request body limit must be positive")

// DecodeJSON validates the request content type and decodes exactly one JSON
// value into dst. Unknown object fields are rejected and body reads are capped.
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) error {
	if maxBytes <= 0 {
		return ErrInvalidBodyLimit
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return apperror.New(
			http.StatusUnsupportedMediaType,
			apperror.AppErrorUnsupportedMediaType,
			"content type must be application/json",
			nil,
		)
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return classifyJSONDecodeError(err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return classifyJSONDecodeError(err)
		}
		return apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"request body must contain exactly one JSON value",
			nil,
		)
	}

	return nil
}

func classifyJSONDecodeError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return apperror.New(
			http.StatusRequestEntityTooLarge,
			apperror.AppErrorRequestTooLarge,
			"request body is too large",
			nil,
		)
	}
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"request body must contain valid JSON",
		nil,
	)
}
