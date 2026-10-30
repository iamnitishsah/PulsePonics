package database

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Migrate applies every *.sql file in fsys that is not yet recorded in
// schema_migrations, in filename order. Files are split into statements on
// semicolons at the end of a line, so keep one statement-ending ";" per line.
//
// MySQL commits DDL implicitly, so a file that fails halfway is not rolled
// back. Write statements so they can be re-run (CREATE TABLE IF NOT EXISTS):
// after fixing the file, the next start applies it again from the top.
func Migrate(ctx context.Context, db *sqlx.DB, fsys fs.FS) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(255) NOT NULL PRIMARY KEY,
			applied_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
		) ENGINE=InnoDB`)
	if err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	var applied []string
	if err := db.SelectContext(ctx, &applied, `SELECT version FROM schema_migrations`); err != nil {
		return fmt.Errorf("migrate: read applied versions: %w", err)
	}
	done := make(map[string]bool, len(applied))
	for _, v := range applied {
		done[v] = true
	}

	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("migrate: list files: %w", err)
	}
	sort.Strings(files)

	for _, name := range files {
		if done[name] {
			continue
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}
		for i, stmt := range splitStatements(string(body)) {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("migrate: %s statement %d: %w", name, i+1, err)
			}
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			return fmt.Errorf("migrate: record %s: %w", name, err)
		}
		log.Printf("migrate: applied %s", name)
	}
	return nil
}

func splitStatements(sql string) []string {
	var (
		stmts   []string
		current strings.Builder
	)
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSuffix(strings.TrimSpace(current.String()), ";")
			stmts = append(stmts, stmt)
			current.Reset()
		}
	}
	if rest := strings.TrimSpace(current.String()); rest != "" {
		stmts = append(stmts, rest)
	}
	return stmts
}
