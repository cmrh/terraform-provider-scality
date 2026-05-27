package client

import (
	"errors"
	"fmt"
)

// APIError is a typed error returned by the IAM API when the response body
// parses as an XML ErrorResponse envelope. Callers should branch on Code via
// the IsNotFound helper (or errors.As) rather than inspecting Error().
type APIError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// IsNotFound reports whether err is (or wraps) an APIError with code NoSuchEntity.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == "NoSuchEntity"
}
