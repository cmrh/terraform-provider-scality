package client

import (
	"errors"
	"fmt"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{Code: "NoSuchEntity", Message: "user not found", StatusCode: 404}
	got := err.Error()
	want := "NoSuchEntity: user not found"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "direct NoSuchEntity APIError",
			err:  &APIError{Code: "NoSuchEntity", Message: "x"},
			want: true,
		},
		{
			name: "wrapped NoSuchEntity APIError",
			err:  fmt.Errorf("get user: %w", &APIError{Code: "NoSuchEntity", Message: "x"}),
			want: true,
		},
		{
			name: "double-wrapped NoSuchEntity APIError",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", &APIError{Code: "NoSuchEntity"})),
			want: true,
		},
		{
			name: "APIError with different code",
			err:  &APIError{Code: "AccessDenied", Message: "x"},
			want: false,
		},
		{
			name: "plain error containing NoSuchEntity substring",
			err:  errors.New("NoSuchEntity"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}
