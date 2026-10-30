package database

import (
	"io/fs"
	"strings"
	"testing"

	"pulse-backend/migrations"
)

func TestSplitStatements(t *testing.T) {
	sql := `-- comment
CREATE TABLE a (
    id INT
);

CREATE TABLE b (id INT);
INSERT INTO a VALUES (1)`
	got := splitStatements(sql)
	if len(got) != 3 {
		t.Fatalf("want 3 statements, got %d: %q", len(got), got)
	}
	if strings.HasSuffix(got[0], ";") {
		t.Errorf("statement should not keep its semicolon: %q", got[0])
	}
}

// Every embedded migration must split into statements cleanly.
func TestEmbeddedMigrationsSplit(t *testing.T) {
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	for _, name := range files {
		body, _ := fs.ReadFile(migrations.FS, name)
		for i, stmt := range splitStatements(string(body)) {
			if !strings.HasPrefix(strings.ToUpper(stmt), "CREATE") {
				t.Errorf("%s statement %d does not start with CREATE: %.40q", name, i+1, stmt)
			}
		}
	}
}
