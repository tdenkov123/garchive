package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractGooseUp(t *testing.T) {
	raw := `-- +goose Up
CREATE TABLE t (id INT);

-- +goose Down
DROP TABLE t;`
	sql := extractGooseUp(raw)
	require.True(t, strings.Contains(sql, "CREATE TABLE t"))
	require.False(t, strings.Contains(sql, "DROP TABLE"))
}

func TestRunMigrations_Embedded(t *testing.T) {
	entries, err := migrationFS.ReadDir("migrations")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}
