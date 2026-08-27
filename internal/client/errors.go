package client

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by the Client. Higher layers can map these to
// stable error codes without inspecting wrapped values.
var (
	// ErrTimeout is returned when the request exceeds the configured
	// per-request timeout.
	ErrTimeout = errors.New("client: request timed out")
	// ErrNetwork is returned for any non-timeout transport-level failure
	// (DNS, connection refused, TLS handshake, etc).
	ErrNetwork = errors.New("client: network error")
)

// ServiceError represents a structured error returned by the Atlas AP
// Remote service. The HTTP status and machine-readable code are preserved
// so the CLI can render stable error envelopes.
type ServiceError struct {
	Code       string
	Message    string
	HTTPStatus int
}

// Error renders "<code>: <message>". The token is never included.
func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Is reports whether target is also a *ServiceError. This lets callers use
// errors.As to recover the structured fields.
func (e *ServiceError) Is(target error) bool {
	_, ok := target.(*ServiceError)
	return ok
}
