package provider

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorClass string

const (
	ErrorTransient      ErrorClass = "transient"
	ErrorRateLimit      ErrorClass = "rate_limit"
	ErrorAuth           ErrorClass = "auth"
	ErrorInvalidRequest ErrorClass = "invalid_request"
	ErrorUnavailable    ErrorClass = "unavailable"
	ErrorNotFound       ErrorClass = "not_found"
	ErrorSafety         ErrorClass = "safety"
	ErrorCancelled      ErrorClass = "cancelled"
	ErrorUnknown        ErrorClass = "unknown"
)

type ProviderError struct {
	Provider   string
	StatusCode int
	Class      ErrorClass
	RetryAfter time.Duration
	Message    string
	Err        error
}

func (e *ProviderError) Error() string {
	detail := strings.TrimSpace(e.Message)
	if len(detail) > 512 {
		detail = detail[:512] + "..."
	}
	if detail == "" && e.Err != nil {
		detail = e.Err.Error()
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("provider %s: %s (status %d): %s", e.Provider, e.Class, e.StatusCode, detail)
	}
	return fmt.Sprintf("provider %s: %s: %s", e.Provider, e.Class, detail)
}
func (e *ProviderError) Unwrap() error { return e.Err }

func NewHTTPError(provider string, status int, body string, header http.Header) error {
	retryAfter := time.Duration(0)
	if seconds, err := strconv.Atoi(header.Get("Retry-After")); err == nil && seconds >= 0 {
		retryAfter = time.Duration(seconds) * time.Second
	}
	return &ProviderError{Provider: provider, StatusCode: status, Class: classForStatus(status), RetryAfter: retryAfter, Message: body}
}

func classForStatus(status int) ErrorClass {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrorAuth
	case status == http.StatusNotFound:
		return ErrorNotFound
	case status == http.StatusTooManyRequests:
		return ErrorRateLimit
	case status == http.StatusRequestTimeout || status == http.StatusConflict:
		return ErrorTransient
	case status >= 400 && status < 500:
		return ErrorInvalidRequest
	case status >= 500:
		return ErrorUnavailable
	default:
		return ErrorUnknown
	}
}

func ClassifyError(err error) ErrorClass {
	if err == nil {
		return ErrorUnknown
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ErrorCancelled
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Class
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return ErrorTransient
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "rate limit"), strings.Contains(message, "status 429"):
		return ErrorRateLimit
	case strings.Contains(message, "connection refused"), strings.Contains(message, "connection reset"), strings.Contains(message, "temporary"):
		return ErrorTransient
	case strings.Contains(message, "status 500"), strings.Contains(message, "status 502"), strings.Contains(message, "status 503"), strings.Contains(message, "status 504"), strings.Contains(message, "service unavailable"):
		return ErrorUnavailable
	case strings.Contains(message, "unauthorized"), strings.Contains(message, "forbidden"), strings.Contains(message, "status 401"), strings.Contains(message, "status 403"):
		return ErrorAuth
	case strings.Contains(message, "safety"), strings.Contains(message, "content policy"):
		return ErrorSafety
	default:
		return ErrorUnknown
	}
}

func retryable(class ErrorClass) bool {
	return class == ErrorTransient || class == ErrorRateLimit || class == ErrorUnavailable
}
func failoverAllowed(class ErrorClass) bool {
	return retryable(class) || class == ErrorAuth || class == ErrorNotFound
}
