package tlsutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/tlsutil"
)

func TestLoadServerCredentials_InvalidPaths(t *testing.T) {
	_, err := tlsutil.LoadServerCredentials("missing.crt", "missing.key")
	require.Error(t, err)
}
