package audit_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/tdenkov123/file-metadata-service/internal/audit"
)

func TestLogger_Log(t *testing.T) {
	log := audit.New(slog.Default())
	log.Log(context.Background(), "file.created", "file_id", "abc")
}
