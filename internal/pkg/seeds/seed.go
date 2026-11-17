package seeds

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var seedFiles embed.FS

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS seed_history (
			name TEXT PRIMARY KEY,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create seed history: %w", err)
	}

	files, err := fs.Glob(seedFiles, "sql/*.sql")
	if err != nil {
		return err
	}

	sort.Strings(files)

	for _, file := range files {
		name := strings.TrimPrefix(file, "sql/")

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}

		var exists bool
		err = tx.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT 1 FROM seed_history WHERE name = $1
			)`,
			name,
		).Scan(&exists)

		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if exists {
			_ = tx.Rollback(ctx)
			continue
		}

		query, err := seedFiles.ReadFile(file)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if _, err = tx.Exec(ctx, string(query)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("execute seed %s: %w", name, err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO seed_history (name) VALUES ($1)`,
			name,
		)
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}
