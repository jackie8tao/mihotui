package mihomo

import (
	"errors"
	"fmt"
)

var (
	// ErrUnauthorized indicates invalid or missing secret token.
	ErrUnauthorized = errors.New("mihomo: unauthorized, invalid or missing secret token")
	// ErrNotFound indicates the requested proxy or resource was not found.
	ErrNotFound = errors.New("mihomo: resource not found")
)

// APIError represents an error returned by Mihomo's External Controller API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("mihomo api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("mihomo api error (status %d)", e.StatusCode)
}

// Is allows APIError to be matched using errors.Is.
func (e *APIError) Is(target error) bool {
	if e.StatusCode == 401 && errors.Is(target, ErrUnauthorized) {
		return true
	}
	if e.StatusCode == 404 && errors.Is(target, ErrNotFound) {
		return true
	}
	return false
}
