package grpcx_test

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/marketing-digest/pkg/errorsx"
	"github.com/marketing-digest/pkg/grpcx"
)

func TestToStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not found", errorsx.ErrNotFound, codes.NotFound},
		{"invalid", errorsx.ErrInvalidArgument, codes.InvalidArgument},
		{"wrapped not found", fmt.Errorf("lookup: %w", errorsx.ErrNotFound), codes.NotFound},
		{"internal default", errors.New("db exploded"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := grpcx.ToStatus(tt.err)
			if status.Code(got) != tt.code {
				t.Fatalf("code=%v want=%v", status.Code(got), tt.code)
			}
			// Ensure we never leak the raw internal message for unknown errors.
			if tt.code == codes.Internal && status.Convert(got).Message() != "internal error" {
				t.Fatalf("leaked message: %s", status.Convert(got).Message())
			}
		})
	}
}
