package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

func TestWriteErrorMapsApplicationErrorAndDoesNotExposeCause(t *testing.T) {
	err := apperror.Wrap(
		http.StatusUnauthorized,
		apperror.AppErrorUnauthenticated,
		"credentials are invalid",
		nil,
		errors.New("sensitive database detail"),
	)
	recorder := httptest.NewRecorder()

	WriteError(recorder, err)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	var response Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success ||
		response.Code != apperror.AppErrorUnauthenticated ||
		response.Message != "credentials are invalid" ||
		len(response.Details) != 0 {
		t.Fatalf("unexpected error response: %+v", response)
	}
	if recorder.Body.String() == "" || strings.Contains(recorder.Body.String(), "sensitive database detail") {
		t.Fatal("response exposed internal error cause")
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("error response missing Cache-Control: no-store")
	}
}

func TestWriteErrorMapsUnknownErrorToSafeInternalResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, errors.New("database password leaked here"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if strings.Contains(recorder.Body.String(), "database password") {
		t.Fatal("response exposed internal error")
	}
	var response Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success || response.Code != apperror.AppErrorInternal {
		t.Fatalf("unexpected internal error response: %+v", response)
	}
}

func TestWriteJSONEncodesBeforeWritingHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := WriteSuccess(recorder, http.StatusOK, make(chan int))
	if err == nil {
		t.Fatal("WriteSuccess() error = nil, want unsupported value error")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want recorder default status before write", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "" {
		t.Fatal("WriteSuccess() wrote headers despite encoding failure")
	}
}

func TestWriteSuccessUsesDataEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := WriteSuccess(recorder, http.StatusOK, map[string]string{"id": "user-1"}); err != nil {
		t.Fatalf("WriteSuccess() error = %v", err)
	}

	var response Response[map[string]string]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data == nil || (*response.Data)["id"] != "user-1" ||
		response.Code != "" || response.Message != "" {
		t.Fatalf("unexpected success response: %+v", response)
	}
}

func TestWriteErrorReturnsStandardFailureResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"invalid input",
		map[string]any{"field": "email"},
	))

	var response Response[json.RawMessage]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success || response.Data != nil {
		t.Fatalf("unexpected error envelope: %+v", response)
	}
	if response.Code != apperror.AppErrorInvalidArgument ||
		response.Message != "invalid input" ||
		response.Details["field"] != "email" {
		t.Fatalf("unexpected error response fields: %+v", response)
	}
}

func TestResponseWithAppErrorReturnsStandardFailureResponse(t *testing.T) {
	response := (Response[any]{}).WithAppError(apperror.New(
		http.StatusForbidden,
		apperror.AppErrorForbidden,
		"access denied",
		map[string]any{"resource": "profile"},
	))

	if response.Success || response.Data != nil ||
		response.Code != apperror.AppErrorForbidden ||
		response.Message != "access denied" ||
		response.Details["resource"] != "profile" {
		t.Fatalf("unexpected response from WithAppError: %+v", response)
	}
}
