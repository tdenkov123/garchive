package grpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"not found", domain.ErrNotFound, codes.NotFound},
		{"access denied", domain.ErrAccessDenied, codes.PermissionDenied},
		{"invalid input", domain.ErrInvalidInput, codes.InvalidArgument},
		{"already exists", domain.ErrAlreadyExists, codes.AlreadyExists},
		{"internal", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapError(tt.err)
			require.Equal(t, tt.code, status.Code(err))
		})
	}
}
