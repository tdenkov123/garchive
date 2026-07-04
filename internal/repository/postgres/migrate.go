package postgres

import (
	"context"
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

var gooseUpRE = regexp.MustCompile(`(?s)-- \+goose Up\s*(.*?)(?:-- \+goose Down|$)`)

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		raw, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		sql := extractGooseUp(string(raw))
		if strings.TrimSpace(sql) == "" {
			continue
		}
		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func extractGooseUp(raw string) string {
	match := gooseUpRE.FindStringSubmatch(raw)
	if len(match) < 2 {
		return strings.TrimSpace(raw)
	}
	return strings.TrimSpace(match[1])
}
