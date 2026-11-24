package middleware

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

var (
	ErrNoCORSOrigins     = errors.New("at least one CORS origin is required")
	ErrInvalidCORSOrigin = errors.New("invalid CORS origin")
)

type CORSConfig struct {
	AllowedOrigins                []string
	RequireOriginForUnsafeMethods bool
}

type CORS struct {
	allowedOrigins                map[string]struct{}
	requireOriginForUnsafeMethods bool
}

var corsAllowedMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodOptions: {},
}

var corsAllowedHeaders = map[string]struct{}{
	"accept":           {},
	"content-type":     {},
	"authorization":    {},
	"x-csrf-token":     {},
	"x-requested-with": {},
}

func NewCORS(cfg CORSConfig) (*CORS, error) {
	if len(cfg.AllowedOrigins) == 0 {
		return nil, ErrNoCORSOrigins
	}

	origins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		normalized, err := normalizeCORSOrigin(origin)
		if err != nil {
			return nil, err
		}
		origins[normalized] = struct{}{}
	}

	return &CORS{
		allowedOrigins:                origins,
		requireOriginForUnsafeMethods: cfg.RequireOriginForUnsafeMethods,
	}, nil
}

func (c *CORS) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		normalizedOrigin := ""
		if origin != "" {
			var err error
			normalizedOrigin, err = normalizeCORSOrigin(origin)
			if err != nil {
				httpx.WriteError(w, apperror.New(
					http.StatusForbidden,
					apperror.AppErrorForbidden,
					"request origin is not allowed",
					nil,
				))
				return
			}
			if _, allowed := c.allowedOrigins[normalizedOrigin]; !allowed {
				httpx.WriteError(w, apperror.New(
					http.StatusForbidden,
					apperror.AppErrorForbidden,
					"request origin is not allowed",
					nil,
				))
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		} else if c.requireOriginForUnsafeMethods && isUnsafeMethod(r.Method) {
			httpx.WriteError(w, apperror.New(
				http.StatusForbidden,
				apperror.AppErrorForbidden,
				"request origin is required",
				nil,
			))
			return
		}

		if r.Method == http.MethodOptions &&
			r.Header.Get("Access-Control-Request-Method") != "" {
			if origin == "" {
				httpx.WriteError(w, apperror.New(
					http.StatusForbidden,
					apperror.AppErrorForbidden,
					"request origin is required",
					nil,
				))
				return
			}
			if !c.validPreflight(r) {
				httpx.WriteError(w, apperror.New(
					http.StatusForbidden,
					apperror.AppErrorForbidden,
					"CORS preflight is not allowed",
					nil,
				))
				return
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization, X-CSRF-Token, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(int((10 * time.Minute).Seconds())))
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *CORS) validPreflight(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")))
	if _, ok := corsAllowedMethods[method]; !ok {
		return false
	}

	for _, requestedHeader := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header := strings.ToLower(strings.TrimSpace(requestedHeader))
		if header == "" {
			continue
		}
		if _, ok := corsAllowedHeaders[header]; !ok {
			return false
		}
	}

	return true
}

func normalizeCORSOrigin(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.Path != "" ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", ErrInvalidCORSOrigin
	}

	host := strings.ToLower(parsed.Host)
	if (parsed.Scheme == "http" && strings.HasSuffix(host, ":80")) ||
		(parsed.Scheme == "https" && strings.HasSuffix(host, ":443")) {
		host = strings.TrimSuffix(host, ":80")
		host = strings.TrimSuffix(host, ":443")
	}

	return strings.ToLower(parsed.Scheme) + "://" + host, nil
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, "TRACE":
		return false
	default:
		return true
	}
}
